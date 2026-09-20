package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL  string
	AdminToken   string
	PublicAddr   string
	AdminAddr    string
	SessionDir   string
	ImageBaseURL string
	ImageAuth    string
	TGProxy      string
	Dev          bool
}

func Load() Config {
	return Config{
		DatabaseURL:  strings.TrimSpace(os.Getenv("DATABASE_URL")),
		AdminToken:   strings.TrimSpace(os.Getenv("NEXA_ADMIN_TOKEN")),
		PublicAddr:   env("NEXA_PUBLIC_ADDR", ":8080"),
		AdminAddr:    env("NEXA_ADMIN_ADDR", "127.0.0.1:8081"),
		SessionDir:   env("NEXA_SESSION_DIR", "data/sessions"),
		ImageBaseURL: env("NEXA_IMAGE_BASE_URL", "https://image.kiteyuan.info"),
		ImageAuth:    strings.TrimSpace(os.Getenv("NEXA_IMAGE_AUTH")),
		TGProxy:      strings.TrimSpace(os.Getenv("NEXA_TG_PROXY")),
		Dev:          os.Getenv("NEXA_DEV") == "1",
	}
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func (c Config) Validate() error {
	if err := validateAdminToken(c.AdminToken); err != nil {
		return err
	}
	if c.DatabaseURL == "" {
		return errNoDatabase
	}
	return nil
}

func validateAdminToken(token string) error {
	if len(token) < 24 {
		return errWeakAdminToken
	}
	lower := strings.ToLower(token)
	switch lower {
	case "change-me", "nexa", "replace-with-a-long-random-token":
		return errWeakAdminToken
	}
	if strings.HasPrefix(lower, "replace-with-") {
		return errWeakAdminToken
	}
	return nil
}

var (
	errWeakAdminToken = errors.New("NEXA_ADMIN_TOKEN 至少 24 位，且不能使用示例/占位值")
	errNoDatabase     = errors.New("DATABASE_URL 必须设置")
)
