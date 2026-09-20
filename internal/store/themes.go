package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func uniqueViolation(err error) bool {
	var e *pgconn.PgError
	return errors.As(err, &e) && e.Code == "23505"
}

func (s *Store) ListThemes(ctx context.Context) ([]kernel.Theme, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, slug, sort_order FROM themes ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []kernel.Theme
	for rows.Next() {
		var t kernel.Theme
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.attachThemeSources(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) CreateTheme(ctx context.Context, name string) (kernel.Theme, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return kernel.Theme{}, fmt.Errorf("%w: 需要名称", kernel.ErrInvalid)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return kernel.Theme{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var t kernel.Theme
	err = tx.QueryRow(ctx, `
		INSERT INTO themes(name, slug, sort_order)
		VALUES ($1, $2, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM themes))
		RETURNING id, name, slug, sort_order`, name, fmt.Sprintf("tmp-%d", time.Now().UnixNano())).
		Scan(&t.ID, &t.Name, &t.Slug, &t.SortOrder)
	if uniqueViolation(err) {
		return kernel.Theme{}, fmt.Errorf("%w: 栏目已存在", kernel.ErrConflict)
	}
	if err != nil {
		return kernel.Theme{}, err
	}
	slug := kernel.ThemeSlug(name)
	if slug == "" {
		slug = fmt.Sprintf("n%d", t.ID)
	} else {
		var taken bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM themes WHERE slug=$1 AND id<>$2)`, slug, t.ID).Scan(&taken); err != nil {
			return kernel.Theme{}, err
		}
		if taken {
			slug = fmt.Sprintf("%s-%d", slug, t.ID)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE themes SET slug=$2 WHERE id=$1`, t.ID, slug); err != nil {
		return kernel.Theme{}, err
	}
	t.Slug = slug
	if err := tx.Commit(ctx); err != nil {
		return kernel.Theme{}, err
	}
	return t, nil
}

func (s *Store) UpdateTheme(ctx context.Context, id int64, name string) (kernel.Theme, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return kernel.Theme{}, fmt.Errorf("%w: 需要名称", kernel.ErrInvalid)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return kernel.Theme{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var t kernel.Theme
	err = tx.QueryRow(ctx, `SELECT id, name, slug FROM themes WHERE id=$1`, id).Scan(&t.ID, &t.Name, &t.Slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return kernel.Theme{}, kernel.ErrNotFound
	}
	if err != nil {
		return kernel.Theme{}, err
	}

	slug := kernel.ThemeSlug(name)
	if slug == "" {
		slug = fmt.Sprintf("n%d", id)
	} else {
		var taken bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM themes WHERE slug=$1 AND id<>$2)`, slug, id).Scan(&taken); err != nil {
			return kernel.Theme{}, err
		}
		if taken {
			slug = fmt.Sprintf("%s-%d", slug, id)
		}
	}

	err = tx.QueryRow(ctx, `
		UPDATE themes SET name=$2, slug=$3 WHERE id=$1
		RETURNING id, name, slug`, id, name, slug).Scan(&t.ID, &t.Name, &t.Slug)
	if uniqueViolation(err) {
		return kernel.Theme{}, fmt.Errorf("%w: 栏目已存在", kernel.ErrConflict)
	}
	if err != nil {
		return kernel.Theme{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return kernel.Theme{}, err
	}
	return t, nil
}

func (s *Store) DeleteTheme(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM themes WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return kernel.ErrNotFound
	}
	return nil
}

func (s *Store) MoveTheme(ctx context.Context, id int64, dir int) error {
	if dir != -1 && dir != 1 {
		return fmt.Errorf("%w: 无效方向", kernel.ErrInvalid)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `SELECT id FROM themes ORDER BY sort_order, id FOR UPDATE`)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var themeID int64
		if err := rows.Scan(&themeID); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, themeID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	at := -1
	for i, themeID := range ids {
		if themeID == id {
			at = i
			break
		}
	}
	if at < 0 {
		return kernel.ErrNotFound
	}
	to := at + dir
	if to < 0 || to >= len(ids) {
		return nil
	}
	ids[at], ids[to] = ids[to], ids[at]
	order := make([]int32, len(ids))
	for i := range ids {
		order[i] = int32(i + 1)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE themes AS t
		SET sort_order = u.ord
		FROM unnest($1::bigint[], $2::int[]) AS u(id, ord)
		WHERE t.id = u.id`, ids, order); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) BindThemeSource(ctx context.Context, themeID int64, kind string, sourceID int64) error {
	kind = strings.TrimSpace(kind)
	if sourceID <= 0 {
		return fmt.Errorf("%w: 无效来源", kernel.ErrInvalid)
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM themes WHERE id=$1)`, themeID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return kernel.ErrNotFound
	}
	switch kind {
	case kernel.SourceTelegram:
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM channels WHERE id=$1)`, sourceID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return kernel.ErrNotFound
		}
		_, err := s.pool.Exec(ctx, `
			INSERT INTO theme_sources(theme_id, source_kind, channel_id)
			VALUES ($1, 'telegram', $2)
			ON CONFLICT DO NOTHING`, themeID, sourceID)
		return err
	case kernel.SourceRSS:
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM feeds WHERE id=$1)`, sourceID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return kernel.ErrNotFound
		}
		_, err := s.pool.Exec(ctx, `
			INSERT INTO theme_sources(theme_id, source_kind, feed_id)
			VALUES ($1, 'rss', $2)
			ON CONFLICT DO NOTHING`, themeID, sourceID)
		return err
	default:
		return fmt.Errorf("%w: 来源类型无效", kernel.ErrInvalid)
	}
}

func (s *Store) UnbindThemeSource(ctx context.Context, themeID int64, kind string, sourceID int64) error {
	var err error
	switch strings.TrimSpace(kind) {
	case kernel.SourceTelegram:
		_, err = s.pool.Exec(ctx, `DELETE FROM theme_sources WHERE theme_id=$1 AND channel_id=$2`, themeID, sourceID)
	case kernel.SourceRSS:
		_, err = s.pool.Exec(ctx, `DELETE FROM theme_sources WHERE theme_id=$1 AND feed_id=$2`, themeID, sourceID)
	default:
		return fmt.Errorf("%w: 来源类型无效", kernel.ErrInvalid)
	}
	return err
}

func (s *Store) attachThemeSources(ctx context.Context, themes []kernel.Theme) error {
	if len(themes) == 0 {
		return nil
	}
	ids := make([]int64, len(themes))
	index := make(map[int64]int, len(themes))
	for i, t := range themes {
		ids[i] = t.ID
		index[t.ID] = i
		themes[i].Sources = []kernel.ThemeSource{}
	}
	rows, err := s.pool.Query(ctx, `
		SELECT ts.theme_id, ts.source_kind, COALESCE(ts.channel_id, ts.feed_id, 0),
			CASE
				WHEN ts.source_kind = 'telegram' THEN COALESCE(NULLIF(c.title, ''), c.username, '')
				ELSE COALESCE(f.name, '')
			END
		FROM theme_sources ts
		LEFT JOIN channels c ON c.id = ts.channel_id
		LEFT JOIN feeds f ON f.id = ts.feed_id
		WHERE ts.theme_id = ANY($1)
		ORDER BY ts.source_kind, 4`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var themeID int64
		var src kernel.ThemeSource
		if err := rows.Scan(&themeID, &src.SourceKind, &src.SourceID, &src.SourceName); err != nil {
			return err
		}
		i, ok := index[themeID]
		if !ok {
			continue
		}
		themes[i].Sources = append(themes[i].Sources, src)
	}
	return rows.Err()
}

func (s *Store) attachChannelThemes(ctx context.Context, channels []Channel) error {
	if len(channels) == 0 {
		return nil
	}
	ids := make([]int64, len(channels))
	index := make(map[int64]int, len(channels))
	for i, c := range channels {
		ids[i] = c.ID
		index[c.ID] = i
		channels[i].Themes = []kernel.Theme{}
	}
	rows, err := s.pool.Query(ctx, `
		SELECT ts.channel_id, t.id, t.name, t.slug
		FROM theme_sources ts
		JOIN themes t ON t.id = ts.theme_id
		WHERE ts.channel_id = ANY($1)
		ORDER BY t.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var channelID int64
		var t kernel.Theme
		if err := rows.Scan(&channelID, &t.ID, &t.Name, &t.Slug); err != nil {
			return err
		}
		i, ok := index[channelID]
		if !ok {
			continue
		}
		channels[i].Themes = append(channels[i].Themes, t)
	}
	return rows.Err()
}

func (s *Store) attachFeedThemes(ctx context.Context, feeds []Feed) error {
	if len(feeds) == 0 {
		return nil
	}
	ids := make([]int64, len(feeds))
	index := make(map[int64]int, len(feeds))
	for i, f := range feeds {
		ids[i] = f.ID
		index[f.ID] = i
		feeds[i].Themes = []kernel.Theme{}
	}
	rows, err := s.pool.Query(ctx, `
		SELECT ts.feed_id, t.id, t.name, t.slug
		FROM theme_sources ts
		JOIN themes t ON t.id = ts.theme_id
		WHERE ts.feed_id = ANY($1)
		ORDER BY t.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var feedID int64
		var t kernel.Theme
		if err := rows.Scan(&feedID, &t.ID, &t.Name, &t.Slug); err != nil {
			return err
		}
		i, ok := index[feedID]
		if !ok {
			continue
		}
		feeds[i].Themes = append(feeds[i].Themes, t)
	}
	return rows.Err()
}

func (s *Store) attachMessageThemes(ctx context.Context, items []Message) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]int64, len(items))
	index := make(map[int64]int, len(items))
	for i, m := range items {
		ids[i] = m.ID
		index[m.ID] = i
		items[i].Themes = []kernel.Theme{}
	}
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, t.id, t.name, t.slug
		FROM messages m
		JOIN theme_sources ts ON (
			(m.source_kind = 'telegram' AND ts.channel_id = m.channel_id)
			OR (m.source_kind = 'rss' AND ts.feed_id = m.feed_id)
		)
		JOIN themes t ON t.id = ts.theme_id
		WHERE m.id = ANY($1)
		ORDER BY t.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var messageID int64
		var t kernel.Theme
		if err := rows.Scan(&messageID, &t.ID, &t.Name, &t.Slug); err != nil {
			return err
		}
		i, ok := index[messageID]
		if !ok {
			continue
		}
		items[i].Themes = append(items[i].Themes, t)
	}
	return rows.Err()
}
