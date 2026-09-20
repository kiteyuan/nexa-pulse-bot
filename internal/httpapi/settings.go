package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.Content.Settings(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if st.LLMAPIKey != "" {
		st.LLMAPIKey = mask(st.LLMAPIKey)
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) {
	current, err := s.Content.Settings(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var body kernel.Settings
	if err := readJSON(w, r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "无效 JSON")
		return
	}
	if strings.Contains(body.LLMAPIKey, "…") || body.LLMAPIKey == "" {
		body.LLMAPIKey = current.LLMAPIKey
	}
	if err := s.Content.SaveSettings(r.Context(), body); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) logs(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.Content.Logs(r.Context(), limit)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []kernel.Log{}
	}
	writeJSON(w, http.StatusOK, items)
}

func pathID(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(r.PathValue(key), 10, 64)
}

func mask(s string) string {
	if len(s) <= 4 {
		return "…"
	}
	return s[:2] + "…" + s[len(s)-2:]
}
