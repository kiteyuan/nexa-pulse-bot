package ports

import (
	"context"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

type Logger interface {
	AddLog(ctx context.Context, level, source, message string)
}

// TelegramRepo is the only store surface Telegram collection may use.
type TelegramRepo interface {
	Logger
	GetAccount(ctx context.Context, id int64) (kernel.Account, error)
	SetAccountStatus(ctx context.Context, id int64, status string) error
	UpsertChannel(ctx context.Context, c kernel.Channel) error
	DeleteChannelByTelegram(ctx context.Context, accountID, telegramID int64) error
	ActiveChannels(ctx context.Context) ([]kernel.Channel, error)
	SetChannelCursor(ctx context.Context, id, lastID int64) error
	InsertRaw(ctx context.Context, item kernel.RawItem) (bool, error)
}

// RSSRepo is the only store surface RSS collection may use.
type RSSRepo interface {
	Logger
	ActiveFeeds(ctx context.Context) ([]kernel.Feed, error)
	InsertRaw(ctx context.Context, item kernel.RawItem) (bool, error)
}

type LoginState struct {
	URL    string `json:"url"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

type TelegramAdmin interface {
	StartQR(accountID int64, password string)
	Login(accountID int64) LoginState
	Sync(ctx context.Context, accountID int64) (int, error)
	Forget(accountID int64)
}

type Intake interface {
	Name() string
	Poll(ctx context.Context) error
}
