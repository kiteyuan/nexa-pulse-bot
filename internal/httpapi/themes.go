package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/kiteyuan/nexa-pulse-bot/internal/kernel"
)

func (s *Server) listPublicThemes(w http.ResponseWriter, r *http.Request) {
	items, err := s.Themes.ListThemes(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	out := make([]kernel.Theme, 0, len(items))
	for _, t := range items {
		out = append(out, kernel.Theme{ID: t.ID, Name: t.Name, Slug: t.Slug})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) listThemes(w http.ResponseWriter, r *http.Request) {
	items, err := s.Themes.ListThemes(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []kernel.Theme{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createTheme(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(w, r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "无效 JSON")
		return
	}
	theme, err := s.Themes.CreateTheme(r.Context(), body.Name)
	if err != nil {
		s.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, theme)
}

func (s *Server) patchTheme(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(w, r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "无效 JSON")
		return
	}
	theme, err := s.Themes.UpdateTheme(r.Context(), id, body.Name)
	if err != nil {
		s.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, theme)
}

func (s *Server) moveTheme(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	var body struct {
		Dir string `json:"dir"`
	}
	if err := readJSON(w, r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "无效 JSON")
		return
	}
	dir := 0
	switch body.Dir {
	case "up":
		dir = -1
	case "down":
		dir = 1
	default:
		writeErr(w, http.StatusBadRequest, "无效方向")
		return
	}
	if err := s.Themes.MoveTheme(r.Context(), id, dir); err != nil {
		s.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) deleteTheme(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	if err := s.Themes.DeleteTheme(r.Context(), id); err != nil {
		s.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) bindThemeSource(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	var body struct {
		SourceKind string `json:"source_kind"`
		SourceID   int64  `json:"source_id"`
	}
	if err := readJSON(w, r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "无效 JSON")
		return
	}
	if err := s.Themes.BindThemeSource(r.Context(), id, body.SourceKind, body.SourceID); err != nil {
		s.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) unbindThemeSource(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "无效 id")
		return
	}
	kind := r.URL.Query().Get("kind")
	sourceID, _ := strconv.ParseInt(r.URL.Query().Get("source_id"), 10, 64)
	if err := s.Themes.UnbindThemeSource(r.Context(), id, kind, sourceID); err != nil {
		s.writeCatalogErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) writeCatalogErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case kernel.IsInvalid(err):
		msg := err.Error()
		if i := strings.LastIndex(msg, ": "); i >= 0 {
			msg = msg[i+2:]
		}
		writeErr(w, http.StatusBadRequest, msg)
	case kernel.IsConflict(err):
		writeErr(w, http.StatusConflict, "栏目已存在")
	case kernel.IsNotFound(err):
		writeErr(w, http.StatusNotFound, "不存在")
	default:
		s.fail(w, r, err)
	}
}
