package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
)

func (s *Service) Login(id int64) LoginState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.login == nil || s.login[id] == nil {
		return LoginState{Status: "idle"}
	}
	return *s.login[id]
}

func (s *Service) set(id int64, fn func(*LoginState)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.login == nil {
		s.login = map[int64]*LoginState{}
	}
	st := s.login[id]
	if st == nil {
		st = &LoginState{}
		s.login[id] = st
	}
	fn(st)
}

func (s *Service) abort(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop != nil {
		if cancel := s.stop[id]; cancel != nil {
			cancel()
		}
	}
}

func (s *Service) StartQR(accountID int64, password string) {
	s.abort(accountID)
	s.mu.Lock()
	if s.gen == nil {
		s.gen = map[int64]uint64{}
	}
	s.gen[accountID]++
	gen := s.gen[accountID]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	if s.stop == nil {
		s.stop = map[int64]context.CancelFunc{}
	}
	s.stop[accountID] = cancel
	s.mu.Unlock()
	s.set(accountID, func(st *LoginState) {
		st.Status = "connecting"
		st.Error = ""
		st.URL = ""
	})

	go func() {
		defer cancel()
		acc, err := s.DB.GetAccount(ctx, accountID)
		if err != nil {
			s.failIf(accountID, gen, err)
			return
		}
		s.note(accountID, "正在连接 Telegram")
		err = s.runQR(ctx, acc, func(ctx context.Context, client *telegram.Client, loggedIn qrlogin.LoggedIn) error {
			status, err := client.Auth().Status(ctx)
			if err != nil {
				return err
			}
			if status.Authorized {
				return nil
			}
			return s.qrLogin(ctx, client, loggedIn, accountID, password)
		})
		if err != nil {
			s.failIf(accountID, gen, err)
			return
		}
		if !s.current(accountID, gen) {
			return
		}
		_ = s.DB.SetAccountStatus(context.Background(), accountID, "online")
		s.set(accountID, func(st *LoginState) {
			st.Status = "online"
			st.URL = ""
			st.Error = ""
		})
		s.note(accountID, "已登录")
	}()
}

func (s *Service) qrLogin(ctx context.Context, client *telegram.Client, loggedIn qrlogin.LoggedIn, accountID int64, password string) error {
	show := func(_ context.Context, token qrlogin.Token) error {
		s.set(accountID, func(st *LoginState) {
			st.URL = token.URL()
			st.Status = "waiting"
			st.Error = ""
		})
		go s.note(accountID, "已出码，请用 Telegram：设置 → 设备 → 链接桌面设备")
		return nil
	}
	for range 3 {
		_, err := client.QR().Auth(ctx, loggedIn, show)
		if err == nil {
			return nil
		}
		if passwordNeeded(err) {
			return s.confirmPassword(ctx, client, password)
		}
		var mig *qrlogin.MigrationNeededError
		if !errors.As(err, &mig) || mig.MigrateTo == nil {
			return err
		}
		s.set(accountID, func(st *LoginState) {
			st.Status = "migrating"
			st.URL = ""
		})
		s.note(accountID, fmt.Sprintf("扫码已确认，正在切换到数据中心 %d", mig.MigrateTo.DCID))
		if err := importLogin(ctx, client, mig); err != nil {
			if passwordNeeded(err) {
				return s.confirmPassword(ctx, client, password)
			}
			s.note(accountID, "当前连接导入失败，正在重连机房")
			if err := client.MigrateTo(ctx, mig.MigrateTo.DCID); err != nil {
				return fmt.Errorf("切换数据中心失败: %w", err)
			}
			if err := importLogin(ctx, client, mig); err != nil {
				if passwordNeeded(err) {
					return s.confirmPassword(ctx, client, password)
				}
				continue
			}
		}
		return nil
	}
	return fmt.Errorf("无法切换到账号所在的 Telegram 机房")
}

func importLogin(ctx context.Context, client *telegram.Client, mig *qrlogin.MigrationNeededError) error {
	if len(mig.MigrateTo.Token) == 0 {
		return fmt.Errorf("没有登录令牌")
	}
	res, err := client.API().AuthImportLoginToken(ctx, mig.MigrateTo.Token)
	if err != nil {
		if passwordNeeded(err) {
			return auth.ErrPasswordAuthNeeded
		}
		return err
	}
	if _, ok := res.(*tg.AuthLoginTokenSuccess); !ok {
		return fmt.Errorf("导入登录令牌失败")
	}
	return nil
}

func (s *Service) confirmPassword(ctx context.Context, client *telegram.Client, password string) error {
	if password == "" {
		return fmt.Errorf("需要两步验证密码")
	}
	_, err := client.Auth().Password(ctx, password)
	return err
}

func passwordNeeded(err error) bool {
	return errors.Is(err, auth.ErrPasswordAuthNeeded) || strings.Contains(err.Error(), "SESSION_PASSWORD_NEEDED")
}

func (s *Service) current(id int64, gen uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gen[id] == gen
}

func (s *Service) failIf(id int64, gen uint64, err error) {
	if !s.current(id, gen) {
		return
	}
	s.fail(id, err)
}

func (s *Service) fail(id int64, err error) {
	msg := loginErr(err)
	s.set(id, func(st *LoginState) {
		st.Status = "error"
		st.Error = msg
		st.URL = ""
	})
	s.DB.AddLog(context.Background(), "ERROR", "telegram", fmt.Sprintf("账号 #%d 登录失败: %s", id, msg))
	_ = s.DB.SetAccountStatus(context.Background(), id, "offline")
}

func (s *Service) note(id int64, msg string) {
	s.DB.AddLog(context.Background(), "INFO", "telegram", fmt.Sprintf("账号 #%d %s", id, msg))
}

func loginErr(err error) string {
	if err == nil {
		return "登录失败"
	}
	var mig *qrlogin.MigrationNeededError
	if errors.As(err, &mig) {
		return "切换 Telegram 机房失败。国内网络请设置 NEXA_TG_PROXY（socks5://host:port）后重试"
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "登录等待超时。请用「设置 → 设备 → 链接桌面设备」扫码，或配置 NEXA_TG_PROXY 后重试"
	case errors.Is(err, auth.ErrPasswordAuthNeeded):
		return "需要两步验证密码"
	case errors.Is(err, context.Canceled):
		return "登录已取消"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "SESSION_PASSWORD_NEEDED"):
		return "需要两步验证密码"
	case strings.Contains(msg, "AUTH_TOKEN_EXPIRED"):
		return "登录码已过期，请再点扫码登录，码一出来马上扫"
	case strings.Contains(msg, "AUTH_KEY_UNREGISTERED"):
		return "尚未登录，请先扫码"
	case strings.Contains(msg, "migrate to dc"), strings.Contains(msg, "切换数据中心"):
		return "切换 Telegram 机房超时。请设置 NEXA_TG_PROXY（socks5://host:port）后重试"
	case strings.Contains(msg, "context deadline exceeded"):
		return "登录等待超时。请用「设置 → 设备 → 链接桌面设备」扫码，或配置 NEXA_TG_PROXY 后重试"
	default:
		return msg
	}
}
