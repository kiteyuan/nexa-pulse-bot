package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kiteyuan/nexa-pulse-bot/internal/config"
	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

type stubAccounts struct{}

func (stubAccounts) ListAccounts(context.Context) ([]kernel.Account, error) {
	return []kernel.Account{}, nil
}
func (stubAccounts) CreateAccount(context.Context, string, int, string, string) (kernel.Account, error) {
	return kernel.Account{}, nil
}
func (stubAccounts) DeleteAccount(context.Context, int64) error { return nil }
func (stubAccounts) ListChannels(context.Context, int64) ([]kernel.Channel, error) {
	return nil, nil
}
func (stubAccounts) SetChannelDisabled(context.Context, int64, bool) error { return nil }

func TestAdminAuth(t *testing.T) {
	t.Parallel()
	token := "abcdefghijklmnopqrstuvwx"
	srv := &Server{
		Accounts: stubAccounts{},
		Cfg:      config.Config{AdminToken: token, Dev: true},
	}
	handler := srv.AdminHandler()

	t.Run("health without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
	})

	t.Run("accounts without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status %d", rec.Code)
		}
	})

	t.Run("accounts with wrong token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
		req.Header.Set("Authorization", "Bearer wrong-token-xxxxxxxxxxxxxx")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status %d", rec.Code)
		}
	})

	t.Run("accounts with token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

func TestTokensMatch(t *testing.T) {
	t.Parallel()
	if !tokensMatch("abc", "abc") {
		t.Fatal("equal")
	}
	if tokensMatch("abc", "abd") {
		t.Fatal("unequal")
	}
}
