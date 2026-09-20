package httpapi

import (
	"net/http"
	"strconv"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func (s *Server) listInbox(w http.ResponseWriter, r *http.Request) {
	tab := r.URL.Query().Get("tab")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	counts, err := s.Content.InboxCounts(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	items, err := s.Content.ListInbox(r.Context(), tab, limit)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []kernel.Message{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"counts": counts, "items": items})
}

func (s *Server) patchInbox(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	if err := readJSON(w, r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "无效 JSON")
		return
	}
	if err := s.Content.SetInboxAction(r.Context(), id, body.Action); err != nil {
		s.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
