package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

const messageCols = `id, source_kind, COALESCE(channel_id, feed_id, 0), COALESCE(external_id,''), content, content_hash, media_paths, llm_status, COALESCE(llm_result, 'null'::jsonb), COALESCE(error_message,''), created_at, COALESCE(link,''), COALESCE(origin_title,'')`

const channelCols = `id, account_id, telegram_id, access_hash, username, title, last_message_id, disabled`

func (s *Store) ListChannels(ctx context.Context, accountID int64) ([]Channel, error) {
	q := `SELECT ` + channelCols + ` FROM channels`
	args := []any{}
	if accountID > 0 {
		q += ` WHERE account_id=$1`
		args = append(args, accountID)
	}
	q += ` ORDER BY disabled DESC, id`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Channel
	for rows.Next() {
		var c Channel
		if err := scanChannel(rows, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.attachChannelThemes(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func scanChannel(row interface{ Scan(dest ...any) error }, c *Channel) error {
	return row.Scan(&c.ID, &c.AccountID, &c.TelegramID, &c.AccessHash, &c.Username, &c.Title, &c.LastMessageID, &c.Disabled)
}

func (s *Store) UpsertChannel(ctx context.Context, c Channel) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO channels(account_id, telegram_id, access_hash, username, title, disabled)
		VALUES ($1,$2,$3,$4,$5, TRUE)
		ON CONFLICT (account_id, telegram_id) DO UPDATE SET
			access_hash=EXCLUDED.access_hash,
			username=EXCLUDED.username,
			title=EXCLUDED.title`,
		c.AccountID, c.TelegramID, c.AccessHash, c.Username, c.Title)
	return err
}

func (s *Store) DeleteChannelByTelegram(ctx context.Context, accountID, telegramID int64) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM channels WHERE account_id=$1 AND telegram_id=$2`, accountID, telegramID)
	return err
}

func (s *Store) SetChannelDisabled(ctx context.Context, id int64, disabled bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE channels SET disabled=$2 WHERE id=$1`, id, disabled)
	return err
}

func (s *Store) SetChannelCursor(ctx context.Context, id, lastID int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE channels SET last_message_id=$2 WHERE id=$1 AND last_message_id < $2`, id, lastID)
	return err
}

func (s *Store) ActiveChannels(ctx context.Context) ([]Channel, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+channelCols+` FROM channels WHERE NOT disabled ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Channel
	for rows.Next() {
		var c Channel
		if err := scanChannel(rows, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) InsertRaw(ctx context.Context, item kernel.RawItem) (bool, error) {
	if item.Media == nil {
		item.Media = []string{}
	}
	raw, _ := json.Marshal(item.Media)
	var (
		tag interface{ RowsAffected() int64 }
		err error
	)
	switch item.SourceKind {
	case kernel.SourceTelegram:
		tag, err = s.pool.Exec(ctx, `
			INSERT INTO messages(source_kind, channel_id, external_id, content, content_hash, media_paths, link, origin_title)
			VALUES ('telegram',$1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (channel_id, external_id) WHERE channel_id IS NOT NULL DO NOTHING`,
			item.SourceID, item.ExternalID, item.Content, item.ContentHash, raw, item.Link, item.OriginTitle)
	case kernel.SourceRSS:
		tag, err = s.pool.Exec(ctx, `
			INSERT INTO messages(source_kind, feed_id, external_id, content, content_hash, media_paths, link, origin_title)
			VALUES ('rss',$1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (feed_id, external_id) WHERE feed_id IS NOT NULL DO NOTHING`,
			item.SourceID, item.ExternalID, item.Content, item.ContentHash, raw, item.Link, item.OriginTitle)
	default:
		return false, fmt.Errorf("未知来源")
	}
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) PendingMessages(ctx context.Context, limit int) ([]Message, error) {
	return s.listMessages(ctx, `SELECT `+messageCols+` FROM messages WHERE llm_status=$1 ORDER BY id ASC LIMIT $2`, "pending", limit)
}

func (s *Store) listMessages(ctx context.Context, query, status string, limit int) ([]Message, error) {
	rows, err := s.pool.Query(ctx, query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

func scanMessages(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]Message, error) {
	var out []Message
	for rows.Next() {
		var m Message
		var media []byte
		if err := rows.Scan(&m.ID, &m.SourceKind, &m.SourceID, &m.ExternalID, &m.Content, &m.ContentHash, &media, &m.LLMStatus, &m.LLMResult, &m.ErrorMessage, &m.CreatedAt, &m.Link, &m.OriginTitle); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(media, &m.MediaPaths)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) FinishMessage(ctx context.Context, id int64, llmStatus, errMsg string, result any, importance float64) error {
	var raw []byte
	if result != nil {
		raw, _ = json.Marshal(result)
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE messages SET llm_status=$2, error_message=NULLIF($3,''), llm_result=$4, importance=$5
		WHERE id=$1`, id, llmStatus, errMsg, raw, importance)
	return err
}

func (s *Store) ClaimProcessing(ctx context.Context, id int64) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE messages SET llm_status='processing' WHERE id=$1 AND llm_status='pending'`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *Store) HashExists(ctx context.Context, hash string, exceptID int64) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM messages WHERE content_hash=$1 AND id<>$2 AND llm_status IN ('approved','skipped')
		)`, hash, exceptID).Scan(&exists)
	return exists, err
}

func (s *Store) decorateMessages(ctx context.Context, items []Message) ([]Message, error) {
	if err := s.attachMessageThemes(ctx, items); err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Title, items[i].Content = kernel.Present(items[i])
		items[i].Source = s.sourceName(ctx, items[i])
	}
	return items, nil
}
