package ports

import (
	"context"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

// Accounts is the admin surface for Telegram accounts and channels.
type Accounts interface {
	ListAccounts(ctx context.Context) ([]kernel.Account, error)
	CreateAccount(ctx context.Context, name string, apiID int, apiHash, phone string) (kernel.Account, error)
	DeleteAccount(ctx context.Context, id int64) error
	ListChannels(ctx context.Context, accountID int64) ([]kernel.Channel, error)
	SetChannelDisabled(ctx context.Context, id int64, disabled bool) error
}

// Feeds is the admin surface for RSS subscriptions.
type Feeds interface {
	ListFeeds(ctx context.Context) ([]kernel.Feed, error)
	CreateFeed(ctx context.Context, name, rawURL string) (kernel.Feed, error)
	GetFeed(ctx context.Context, id int64) (kernel.Feed, error)
	UpdateFeed(ctx context.Context, id int64, name, rawURL string) error
	DeleteFeed(ctx context.Context, id int64) error
	SetFeedDisabled(ctx context.Context, id int64, disabled bool) error
}

// Themes is the admin/public surface for theme catalog bindings.
type Themes interface {
	ListThemes(ctx context.Context) ([]kernel.Theme, error)
	CreateTheme(ctx context.Context, name string) (kernel.Theme, error)
	UpdateTheme(ctx context.Context, id int64, name string) (kernel.Theme, error)
	MoveTheme(ctx context.Context, id int64, dir int) error
	DeleteTheme(ctx context.Context, id int64) error
	BindThemeSource(ctx context.Context, themeID int64, kind string, sourceID int64) error
	UnbindThemeSource(ctx context.Context, themeID int64, kind string, sourceID int64) error
}

// Content covers settings, logs, public feed, and inbox review.
type Content interface {
	Settings(ctx context.Context) (kernel.Settings, error)
	SaveSettings(ctx context.Context, st kernel.Settings) error
	Logs(ctx context.Context, limit int) ([]kernel.Log, error)
	PublicItems(ctx context.Context, limit int, theme string) ([]kernel.Message, error)
	PublicItem(ctx context.Context, id int64) (kernel.Message, error)
	InboxCounts(ctx context.Context) (kernel.InboxCounts, error)
	ListInbox(ctx context.Context, tab string, limit int) ([]kernel.Message, error)
	SetInboxAction(ctx context.Context, id int64, action string) error
}
