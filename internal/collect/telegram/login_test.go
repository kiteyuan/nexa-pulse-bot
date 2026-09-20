package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
)

func TestLoginErr(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{context.DeadlineExceeded, "登录等待超时。请用「设置 → 设备 → 链接桌面设备」扫码，或配置 NEXA_TG_PROXY 后重试"},
		{auth.ErrPasswordAuthNeeded, "需要两步验证密码"},
		{errors.New("rpc error code 401: SESSION_PASSWORD_NEEDED"), "需要两步验证密码"},
		{errors.New("rpc error code 401: AUTH_KEY_UNREGISTERED"), "尚未登录，请先扫码"},
		{&qrlogin.MigrationNeededError{MigrateTo: &tg.AuthLoginTokenMigrateTo{DCID: 1}}, "切换 Telegram 机房失败。国内网络请设置 NEXA_TG_PROXY（socks5://host:port）后重试"},
		{errors.New("migrate to dc"), "切换 Telegram 机房超时。请设置 NEXA_TG_PROXY（socks5://host:port）后重试"},
	}
	for _, tc := range cases {
		if got := loginErr(tc.err); got != tc.want {
			t.Fatalf("%v: got %q want %q", tc.err, got, tc.want)
		}
	}
}
