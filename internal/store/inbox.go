package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func (s *Store) InboxCounts(ctx context.Context) (kernel.InboxCounts, error) {
	var c kernel.InboxCounts
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE llm_status IN ('pending','processing')),
			COUNT(*) FILTER (WHERE llm_status IN ('approved','skipped')),
			COUNT(*) FILTER (WHERE llm_status = 'rejected')
		FROM messages`).Scan(&c.Pending, &c.Published, &c.Rejected)
	return c, err
}

func (s *Store) ListInbox(ctx context.Context, tab string, limit int) ([]Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	statuses := []string{"pending", "processing"}
	switch tab {
	case "published":
		statuses = []string{"approved", "skipped"}
	case "rejected":
		statuses = []string{"rejected"}
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+messageCols+`
		FROM messages
		WHERE llm_status = ANY($1)
		ORDER BY id DESC
		LIMIT $2`, statuses, limit)
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

func (s *Store) SetInboxAction(ctx context.Context, id int64, action string) error {
	rows, err := s.pool.Query(ctx, `SELECT `+messageCols+` FROM messages WHERE id=$1`, id)
	if err != nil {
		return err
	}
	defer rows.Close()
	items, err := scanMessages(rows)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return kernel.ErrNotFound
	}
	m := items[0]
	switch strings.TrimSpace(action) {
	case "approve":
		title, body := kernel.Present(m)
		raw, _ := json.Marshal(map[string]any{"send": true, "title": title, "body": body, "reason": "人工通过"})
		_, err = s.pool.Exec(ctx, `
			UPDATE messages SET llm_status='approved', llm_result=$2, error_message=NULL WHERE id=$1`, id, raw)
	case "reject":
		reason := strings.TrimSpace(m.ErrorMessage)
		if reason == "" {
			reason = "人工拒绝"
		}
		_, err = s.pool.Exec(ctx, `UPDATE messages SET llm_status='rejected', error_message=$2 WHERE id=$1`, id, reason)
	default:
		return fmt.Errorf("%w: 无效操作", kernel.ErrInvalid)
	}
	return err
}
