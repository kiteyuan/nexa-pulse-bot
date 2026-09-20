package httpapi

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) rssFeed(w http.ResponseWriter, r *http.Request) {
	items, err := s.Content.PublicItems(r.Context(), 40, strings.TrimSpace(r.URL.Query().Get("theme")))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>NexaPulse</title><description>精选资讯</description>`)
	for _, item := range items {
		view := toFeed(item)
		b.WriteString("<item><title>")
		xmlEscape(&b, view.Title)
		b.WriteString("</title>")
		if view.Link != "" {
			b.WriteString("<link>")
			xmlEscape(&b, view.Link)
			b.WriteString("</link>")
		}
		b.WriteString("<description>")
		xmlEscape(&b, view.Content)
		b.WriteString("</description></item>")
	}
	b.WriteString("</channel></rss>")
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = w.Write([]byte(b.String()))
}

func xmlEscape(b *strings.Builder, s string) {
	_ = xml.EscapeText(b, []byte(s))
}
