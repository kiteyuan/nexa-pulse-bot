package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func (s *Server) listItems(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.Content.PublicItems(r.Context(), limit, strings.TrimSpace(r.URL.Query().Get("theme")))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]feedItem, 0, len(items))
	for _, item := range items {
		out = append(out, toFeed(item))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getItem(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	item, err := s.Content.PublicItem(r.Context(), id)
	if err != nil {
		if kernel.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "不存在")
			return
		}
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toFeed(item))
}

type feedItem struct {
	ID         int64          `json:"id"`
	Title      string         `json:"title"`
	Content    string         `json:"content"`
	Source     string         `json:"source,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	Link       string         `json:"link,omitempty"`
	MediaPaths []string       `json:"media_paths"`
	Themes     []kernel.Theme `json:"themes,omitempty"`
}

func remoteMedia(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
			out = append(out, p)
		}
	}
	return out
}

func toFeed(m kernel.Message) feedItem {
	return feedItem{
		ID: m.ID, Title: m.Title, Content: m.Content, Source: m.Source, Link: m.Link,
		CreatedAt: m.CreatedAt, MediaPaths: remoteMedia(m.MediaPaths), Themes: m.Themes,
	}
}
