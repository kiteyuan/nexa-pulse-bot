package store

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
	"github.com/kiteyuan/nexa-pulse-bot/internal/ports"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Store struct {
	pool *pgxpool.Pool
}

var (
	_ ports.Pipeline     = (*Store)(nil)
	_ ports.Accounts     = (*Store)(nil)
	_ ports.Feeds        = (*Store)(nil)
	_ ports.Themes       = (*Store)(nil)
	_ ports.Content      = (*Store)(nil)
	_ ports.TelegramRepo = (*Store)(nil)
	_ ports.RSSRepo      = (*Store)(nil)
)

type (
	Account  = kernel.Account
	Channel  = kernel.Channel
	Feed     = kernel.Feed
	Message  = kernel.Message
	Settings = kernel.Settings
	Log      = kernel.Log
)

func Open(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	s := &Store{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		var applied bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, version).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate %s: %w", version, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) AddLog(ctx context.Context, level, source, message string) {
	if level == "" {
		level = "INFO"
	}
	_, _ = s.pool.Exec(ctx, `INSERT INTO runtime_logs(level, source, message) VALUES ($1,$2,$3)`, level, source, message)
}

func (s *Store) Logs(ctx context.Context, limit int) ([]Log, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `SELECT id, level, source, message, created_at FROM runtime_logs ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Log
	for rows.Next() {
		var l Log
		if err := rows.Scan(&l.ID, &l.Level, &l.Source, &l.Message, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) Settings(ctx context.Context) (Settings, error) {
	var st Settings
	var keywords []byte
	err := s.pool.QueryRow(ctx, `
		SELECT llm_enabled, llm_base_url, llm_api_key, llm_model, llm_temperature, translate_to,
		       min_length, block_keywords, poll_interval_seconds, collect_interval_seconds
		FROM app_settings WHERE id=1`).Scan(
		&st.LLMEnabled, &st.LLMBaseURL, &st.LLMAPIKey, &st.LLMModel, &st.LLMTemperature, &st.TranslateTo,
		&st.MinLength, &keywords, &st.PollInterval, &st.CollectInterval,
	)
	if err != nil {
		return st, err
	}
	_ = json.Unmarshal(keywords, &st.BlockKeywords)
	return st, nil
}

func (s *Store) SaveSettings(ctx context.Context, st Settings) error {
	if st.BlockKeywords == nil {
		st.BlockKeywords = []string{}
	}
	raw, _ := json.Marshal(st.BlockKeywords)
	_, err := s.pool.Exec(ctx, `
		UPDATE app_settings SET
			llm_enabled=$1, llm_base_url=$2, llm_api_key=$3, llm_model=$4, llm_temperature=$5, translate_to=$6,
			min_length=$7, block_keywords=$8, poll_interval_seconds=$9, collect_interval_seconds=$10
		WHERE id=1`,
		st.LLMEnabled, st.LLMBaseURL, st.LLMAPIKey, st.LLMModel, st.LLMTemperature, st.TranslateTo,
		st.MinLength, raw, st.PollInterval, st.CollectInterval,
	)
	return err
}

func (s *Store) ListAccounts(ctx context.Context) ([]Account, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, api_id, api_hash, phone, status, last_sync, created_at FROM accounts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAccounts(rows)
}

func (s *Store) GetAccount(ctx context.Context, id int64) (Account, error) {
	var a Account
	err := s.pool.QueryRow(ctx, `SELECT id, name, api_id, api_hash, phone, status, last_sync, created_at FROM accounts WHERE id=$1`, id).
		Scan(&a.ID, &a.Name, &a.APIID, &a.APIHash, &a.Phone, &a.Status, &a.LastSync, &a.CreatedAt)
	return a, err
}

func (s *Store) CreateAccount(ctx context.Context, name string, apiID int, apiHash, phone string) (Account, error) {
	var a Account
	err := s.pool.QueryRow(ctx, `
		INSERT INTO accounts(name, api_id, api_hash, phone) VALUES ($1,$2,$3,$4)
		RETURNING id, name, api_id, api_hash, phone, status, last_sync, created_at`,
		name, apiID, apiHash, phone).Scan(&a.ID, &a.Name, &a.APIID, &a.APIHash, &a.Phone, &a.Status, &a.LastSync, &a.CreatedAt)
	return a, err
}

func (s *Store) DeleteAccount(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM accounts WHERE id=$1`, id)
	return err
}

func (s *Store) SetAccountStatus(ctx context.Context, id int64, status string) error {
	_, err := s.pool.Exec(ctx, `UPDATE accounts SET status=$2, last_sync=CASE WHEN $2='online' THEN now() ELSE last_sync END WHERE id=$1`, id, status)
	return err
}

func scanAccounts(rows pgx.Rows) ([]Account, error) {
	var out []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Name, &a.APIID, &a.APIHash, &a.Phone, &a.Status, &a.LastSync, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
