package rss

import (
	"encoding/xml"
	"html"
	"strings"
)

type Entry struct {
	GUID     string
	Title    string
	Link     string
	Body     string
	ImageURL string
}

func Parse(data []byte) []Entry {
	var doc rssDoc
	if err := xml.Unmarshal(data, &doc); err == nil && len(doc.Channel.Items) > 0 {
		return rssEntries(doc.Channel.Items)
	}
	var atom atomDoc
	if err := xml.Unmarshal(data, &atom); err != nil {
		return nil
	}
	return atomEntries(atom.Entries)
}

type rssDoc struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	GUID        string    `xml:"guid"`
	Description string    `xml:"description"`
	Encoded     string    `xml:"encoded"`
	Enclosure   enclosure `xml:"enclosure"`
}

type enclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type atomDoc struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	ID      string     `xml:"id"`
	Summary string     `xml:"summary"`
	Content string     `xml:"content"`
	Links   []atomLink `xml:"link"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func rssEntries(items []rssItem) []Entry {
	out := make([]Entry, 0, len(items))
	for _, item := range items {
		body := item.Encoded
		if strings.TrimSpace(body) == "" {
			body = item.Description
		}
		out = append(out, Entry{
			GUID:     strings.TrimSpace(item.GUID),
			Title:    plain(item.Title),
			Link:     strings.TrimSpace(item.Link),
			Body:     plain(body),
			ImageURL: imageURL(item.Enclosure.URL, item.Enclosure.Type),
		})
	}
	return out
}

func atomEntries(items []atomEntry) []Entry {
	out := make([]Entry, 0, len(items))
	for _, item := range items {
		body := item.Content
		if strings.TrimSpace(body) == "" {
			body = item.Summary
		}
		out = append(out, Entry{
			GUID:  strings.TrimSpace(item.ID),
			Title: plain(item.Title),
			Link:  atomHref(item.Links),
			Body:  plain(body),
		})
	}
	return out
}

func atomHref(links []atomLink) string {
	var fallback string
	for _, link := range links {
		if link.Href == "" {
			continue
		}
		if link.Rel == "" || link.Rel == "alternate" {
			return strings.TrimSpace(link.Href)
		}
		if fallback == "" {
			fallback = link.Href
		}
	}
	return strings.TrimSpace(fallback)
}

func imageURL(raw, mime string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	mime = strings.ToLower(mime)
	if strings.HasPrefix(mime, "image/") || mime == "" {
		return raw
	}
	return ""
}

func plain(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '<':
			in = true
		case r == '>':
			in = false
			b.WriteByte(' ')
		case !in:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(html.UnescapeString(b.String())), " ")
}
