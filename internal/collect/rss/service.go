package rss

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
	"github.com/kiteyuan/nexa-pulse-bot/internal/ports"
)

type Service struct {
	DB  ports.RSSRepo
	Bed ports.ImageUploader
}

func (s *Service) Name() string { return kernel.SourceRSS }

func (s *Service) Poll(ctx context.Context) error {
	if s.DB == nil {
		return nil
	}
	feeds, err := s.DB.ActiveFeeds(ctx)
	if err != nil || len(feeds) == 0 {
		return err
	}
	var total int
	for _, feed := range feeds {
		n, err := s.pull(ctx, feed)
		if err != nil {
			s.DB.AddLog(ctx, "ERROR", "rss", feed.Name+": 抓取失败")
			continue
		}
		total += n
	}
	if total > 0 {
		s.DB.AddLog(ctx, "INFO", "rss", "本轮入库 "+itoa(int64(total))+" 条")
	}
	return nil
}

func (s *Service) pull(ctx context.Context, feed kernel.Feed) (int, error) {
	rawURL, err := kernel.SafeHTTPURL(feed.URL)
	if err != nil {
		return 0, err
	}
	body, err := get(ctx, rawURL, 2<<20)
	if err != nil {
		return 0, err
	}
	entries := Parse(body)
	if len(entries) > 15 {
		entries = entries[:15]
	}
	var n int
	for _, entry := range entries {
		key := itemKey(entry)
		if key == "" {
			continue
		}
		text := strings.TrimSpace(entry.Title)
		if entry.Body != "" {
			if text != "" {
				text += "\n\n"
			}
			text += entry.Body
		}
		if text == "" {
			continue
		}
		paths := s.saveImage(ctx, key, entry.ImageURL)
		ok, err := s.DB.InsertRaw(ctx, kernel.RawItem{
			SourceKind:  kernel.SourceRSS,
			SourceID:    feed.ID,
			ExternalID:  key,
			Content:     text,
			ContentHash: kernel.Hash(text),
			Media:       paths,
			Link:        entry.Link,
			OriginTitle: entry.Title,
		})
		if err != nil {
			return n, err
		}
		if ok {
			n++
		}
	}
	return n, nil
}

func (s *Service) saveImage(ctx context.Context, key, rawURL string) []string {
	if s.Bed == nil {
		if s.DB != nil {
			s.DB.AddLog(ctx, "ERROR", "imgbed", "未配置图床，已跳过图片")
		}
		return nil
	}
	rawURL, err := kernel.SafeHTTPURL(rawURL)
	if err != nil {
		return nil
	}
	data, err := get(ctx, rawURL, 4<<20)
	if err != nil || len(data) == 0 {
		return nil
	}
	ext, ok := kernel.SniffImageExt(data)
	if !ok {
		return nil
	}
	link, err := s.Bed.Upload(ctx, short(key)+ext, data)
	if err != nil {
		s.DB.AddLog(ctx, "ERROR", "imgbed", "上传失败")
		return nil
	}
	return []string{link}
}

func get(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "NexaPulse/1.0")
	client := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			_, err := kernel.SafeHTTPURL(req.URL.String())
			return err
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, http.ErrBodyNotAllowed
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

func itemKey(entry Entry) string {
	key := strings.TrimSpace(entry.GUID)
	if key == "" {
		key = strings.TrimSpace(entry.Link)
	}
	if key == "" {
		key = entry.Title
	}
	if len(key) > 400 {
		return short(key)
	}
	return key
}

func short(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:8])
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	u := uint64(n)
	if n < 0 {
		u = uint64(-n)
	}
	for u > 0 {
		i--
		b[i] = byte('0' + u%10)
		u /= 10
	}
	if n < 0 {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
