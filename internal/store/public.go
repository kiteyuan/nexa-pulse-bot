package store

import (
	"context"
	"strings"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func (s *Store) PublicItems(ctx context.Context, limit int, theme string) ([]Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	theme = strings.TrimSpace(theme)
	rows, err := s.pool.Query(ctx, `
		SELECT `+messageCols+`
		FROM messages m
		WHERE m.llm_status IN ('approved','skipped')
		  AND ($1 = '' OR EXISTS (
			SELECT 1
			FROM theme_sources ts
			JOIN themes th ON th.id = ts.theme_id
			WHERE (th.slug = $1 OR th.id::text = $1)
			  AND (
				(m.source_kind = 'telegram' AND ts.channel_id = m.channel_id)
				OR (m.source_kind = 'rss' AND ts.feed_id = m.feed_id)
			  )
		  ))
		ORDER BY m.id DESC
		LIMIT $2`, theme, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanMessages(rows)
	if err != nil {
		return nil, err
	}
	return s.decorateMessages(ctx, items)
}

func (s *Store) PublicItem(ctx context.Context, id int64) (Message, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+messageCols+`
		FROM messages WHERE id=$1 AND llm_status IN ('approved','skipped')`, id)
	if err != nil {
		return Message{}, err
	}
	defer rows.Close()
	items, err := scanMessages(rows)
	if err != nil {
		return Message{}, err
	}
	if len(items) == 0 {
		return Message{}, kernel.ErrNotFound
	}
	items, err = s.decorateMessages(ctx, items)
	if err != nil {
		return Message{}, err
	}
	return items[0], nil
}
