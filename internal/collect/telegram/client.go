package telegram

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
	"golang.org/x/net/proxy"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func (s *Service) sessionFile(id int64) string {
	return filepath.Join(s.Sessions, fmt.Sprintf("%d.json", id))
}

// Forget aborts any in-flight QR login and removes the local session file.
func (s *Service) Forget(accountID int64) {
	s.abort(accountID)
	s.mu.Lock()
	if s.login != nil {
		delete(s.login, accountID)
	}
	if s.stop != nil {
		delete(s.stop, accountID)
	}
	s.mu.Unlock()
	path := s.sessionFile(accountID)
	_ = os.Remove(path)
	_ = os.Remove(path + "-journal")
}

func (s *Service) options(sessionPath string, handler telegram.UpdateHandler) (telegram.Options, error) {
	resolver := telegram.TDesktopResolver()
	if s.Proxy != "" {
		r, err := proxyResolver(s.Proxy)
		if err != nil {
			return telegram.Options{}, fmt.Errorf("NEXA_TG_PROXY 无效")
		}
		resolver = r
	}
	opts := telegram.Options{
		SessionStorage:   &session.FileStorage{Path: sessionPath},
		Device:           telegram.DeviceTDesktopWindows(),
		Resolver:         resolver,
		MigrationTimeout: time.Minute,
	}
	if handler != nil {
		opts.UpdateHandler = handler
	} else {
		opts.NoUpdates = true
	}
	return opts, nil
}

func (s *Service) run(ctx context.Context, acc kernel.Account, noUpdates bool, fn func(context.Context, *telegram.Client) error) error {
	sessionPath := s.sessionFile(acc.ID)
	if err := os.MkdirAll(s.Sessions, 0o700); err != nil {
		return err
	}
	_ = os.Chmod(s.Sessions, 0o700)
	var handler telegram.UpdateHandler
	if !noUpdates {
		d := tg.NewUpdateDispatcher()
		handler = &d
	}
	opts, err := s.options(sessionPath, handler)
	if err != nil {
		return err
	}
	client := telegram.NewClient(acc.APIID, acc.APIHash, opts)
	err = client.Run(ctx, func(ctx context.Context) error {
		return fn(ctx, client)
	})
	_ = os.Chmod(sessionPath, 0o600)
	return err
}

func (s *Service) runQR(ctx context.Context, acc kernel.Account, fn func(context.Context, *telegram.Client, qrlogin.LoggedIn) error) error {
	sessionPath := s.sessionFile(acc.ID)
	if err := os.MkdirAll(s.Sessions, 0o700); err != nil {
		return err
	}
	_ = os.Chmod(s.Sessions, 0o700)
	d := tg.NewUpdateDispatcher()
	loggedIn := waitLoginToken(&d)
	opts, err := s.options(sessionPath, &d)
	if err != nil {
		return err
	}
	client := telegram.NewClient(acc.APIID, acc.APIHash, opts)
	err = client.Run(ctx, func(ctx context.Context) error {
		return fn(ctx, client, loggedIn)
	})
	_ = os.Chmod(sessionPath, 0o600)
	return err
}

func waitLoginToken(d *tg.UpdateDispatcher) qrlogin.LoggedIn {
	ch := make(chan struct{}, 1)
	d.OnLoginToken(func(context.Context, tg.Entities, *tg.UpdateLoginToken) error {
		select {
		case ch <- struct{}{}:
		default:
		}
		return nil
	})
	return ch
}

func proxyResolver(raw string) (dcs.Resolver, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid proxy")
	}
	dialer, err := proxy.FromURL(u, proxy.Direct)
	if err != nil {
		return nil, err
	}
	cd, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("proxy has no DialContext")
	}
	return dcs.Plain(dcs.PlainOptions{Dial: cd.DialContext}), nil
}
