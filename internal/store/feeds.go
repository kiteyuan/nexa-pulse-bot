package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func (s *Store) ListFeeds(ctx context.Context) ([]Feed, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, url, disabled FROM feeds ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Feed
	for rows.Next() {
		var f Feed
		if err := rows.Scan(&f.ID, &f.Name, &f.URL, &f.Disabled); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.attachFeedThemes(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) CreateFeed(ctx context.Context, name, rawURL string) (Feed, error) {
	var f Feed
	err := s.pool.QueryRow(ctx, `
		INSERT INTO feeds(name, url) VALUES ($1,$2)
		RETURNING id, name, url, disabled`, name, rawURL).
		Scan(&f.ID, &f.Name, &f.URL, &f.Disabled)
	return f, err
}

func (s *Store) GetFeed(ctx context.Context, id int64) (Feed, error) {
	var f Feed
	err := s.pool.QueryRow(ctx, `SELECT id, name, url, disabled FROM feeds WHERE id=$1`, id).
		Scan(&f.ID, &f.Name, &f.URL, &f.Disabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Feed{}, kernel.ErrNotFound
		}
		return Feed{}, err
	}
	return f, nil
}

func (s *Store) DeleteFeed(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM feeds WHERE id=$1`, id)
	return err
}

func (s *Store) SetFeedDisabled(ctx context.Context, id int64, disabled bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE feeds SET disabled=$2 WHERE id=$1`, id, disabled)
	return err
}

func (s *Store) UpdateFeed(ctx context.Context, id int64, name, rawURL string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE feeds SET name=$2, url=$3 WHERE id=$1`, id, name, rawURL)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (s *Store) ActiveFeeds(ctx context.Context) ([]Feed, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, url, disabled FROM feeds WHERE NOT disabled ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Feed
	for rows.Next() {
		var f Feed
		if err := rows.Scan(&f.ID, &f.Name, &f.URL, &f.Disabled); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) sourceName(ctx context.Context, m Message) string {
	var name string
	if m.SourceKind == kernel.SourceRSS {
		_ = s.pool.QueryRow(ctx, `SELECT name FROM feeds WHERE id=$1`, m.SourceID).Scan(&name)
		return name
	}
	_ = s.pool.QueryRow(ctx, `SELECT COALESCE(NULLIF(username,''), title) FROM channels WHERE id=$1`, m.SourceID).Scan(&name)
	return name
}
