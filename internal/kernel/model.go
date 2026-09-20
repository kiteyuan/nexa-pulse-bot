package kernel

import (
	"encoding/json"
	"time"
)

const (
	SourceTelegram = "telegram"
	SourceRSS      = "rss"
)

type Account struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	APIID     int        `json:"api_id"`
	APIHash   string     `json:"api_hash"`
	Phone     string     `json:"phone"`
	Status    string     `json:"status"`
	LastSync  *time.Time `json:"last_sync"`
	CreatedAt time.Time  `json:"created_at"`
}

type Channel struct {
	ID            int64   `json:"id"`
	AccountID     int64   `json:"account_id"`
	TelegramID    int64   `json:"telegram_id"`
	AccessHash    int64   `json:"-"`
	Username      string  `json:"username"`
	Title         string  `json:"title"`
	Disabled      bool    `json:"disabled"`
	LastMessageID int64   `json:"last_message_id"`
	Themes        []Theme `json:"themes,omitempty"`
}

type Feed struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	URL      string  `json:"url"`
	Disabled bool    `json:"disabled"`
	Themes   []Theme `json:"themes,omitempty"`
}

// RawItem is what a source inserts. No delivery fields.
type RawItem struct {
	SourceKind  string
	SourceID    int64
	ExternalID  string
	Content     string
	ContentHash string
	Media       []string
	Link        string
	OriginTitle string
}

type Message struct {
	ID           int64           `json:"id"`
	SourceKind   string          `json:"source_kind"`
	SourceID     int64           `json:"source_id"`
	ExternalID   string          `json:"external_id,omitempty"`
	Content      string          `json:"content"`
	ContentHash  string          `json:"content_hash"`
	MediaPaths   []string        `json:"media_paths"`
	LLMStatus    string          `json:"llm_status"`
	LLMResult    json.RawMessage `json:"llm_result"`
	ErrorMessage string          `json:"error_message"`
	Link         string          `json:"link,omitempty"`
	OriginTitle  string          `json:"origin_title,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	Title        string          `json:"title,omitempty"`
	Source       string          `json:"source,omitempty"`
	Themes       []Theme         `json:"themes,omitempty"`
}

type InboxCounts struct {
	Pending   int `json:"pending"`
	Published int `json:"published"`
	Rejected  int `json:"rejected"`
}

type Theme struct {
	ID      int64         `json:"id"`
	Name    string        `json:"name"`
	Slug    string        `json:"slug"`
	Sources []ThemeSource `json:"sources,omitempty"`
}

type ThemeSource struct {
	SourceKind string `json:"source_kind"`
	SourceID   int64  `json:"source_id"`
	SourceName string `json:"source_name"`
}

type Settings struct {
	LLMEnabled      bool     `json:"llm_enabled"`
	LLMBaseURL      string   `json:"llm_base_url"`
	LLMAPIKey       string   `json:"llm_api_key"`
	LLMModel        string   `json:"llm_model"`
	LLMTemperature  float64  `json:"llm_temperature"`
	TranslateTo     string   `json:"translate_to"`
	MinLength       int      `json:"min_length"`
	BlockKeywords   []string `json:"block_keywords"`
	PollInterval    float64  `json:"poll_interval_seconds"`
	CollectInterval float64  `json:"collect_interval_seconds"`
}

type Log struct {
	ID        int64     `json:"id"`
	Level     string    `json:"level"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
