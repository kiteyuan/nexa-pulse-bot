package httpapi

import (
	"net/http"
	"strings"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func (s *Server) listFeeds(w http.ResponseWriter, r *http.Request) {
	items, err := s.Feeds.ListFeeds(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []kernel.Feed{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createFeed(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := readJSON(w, r, &body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeErr(w, http.StatusBadRequest, "需要 name、url")
		return
	}
	rawURL, err := kernel.SafeHTTPURL(body.URL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "订阅地址无效")
		return
	}
	feed, err := s.Feeds.CreateFeed(r.Context(), strings.TrimSpace(body.Name), rawURL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无法添加订阅")
		return
	}
	writeJSON(w, http.StatusCreated, feed)
}

func (s *Server) patchFeed(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	var body struct {
		Disabled *bool   `json:"disabled"`
		Name     *string `json:"name"`
		URL      *string `json:"url"`
	}
	if err := readJSON(w, r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "无效 JSON")
		return
	}
	if body.Disabled == nil && body.Name == nil && body.URL == nil {
		writeErr(w, http.StatusBadRequest, "无可更新字段")
		return
	}
	if body.Name != nil || body.URL != nil {
		cur, err := s.Feeds.GetFeed(r.Context(), id)
		if err != nil {
			if kernel.IsNotFound(err) {
				writeErr(w, http.StatusNotFound, "不存在")
				return
			}
			s.fail(w, r, err)
			return
		}
		name := cur.Name
		rawURL := cur.URL
		if body.Name != nil {
			name = strings.TrimSpace(*body.Name)
			if name == "" {
				writeErr(w, http.StatusBadRequest, "名称不能为空")
				return
			}
		}
		if body.URL != nil {
			parsed, err := kernel.SafeHTTPURL(*body.URL)
			if err != nil {
				writeErr(w, http.StatusBadRequest, "订阅地址无效")
				return
			}
			rawURL = parsed
		}
		if err := s.Feeds.UpdateFeed(r.Context(), id, name, rawURL); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	if body.Disabled != nil {
		if err := s.Feeds.SetFeedDisabled(r.Context(), id, *body.Disabled); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) deleteFeed(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	if err := s.Feeds.DeleteFeed(r.Context(), id); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
