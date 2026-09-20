package kernel

import "encoding/json"

// Present derives display title and body from LLM result with raw-content fallbacks.
func Present(m Message) (title, body string) {
	var doc struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	_ = json.Unmarshal(m.LLMResult, &doc)
	title, body = doc.Title, doc.Body
	if body == "" {
		body = m.Content
	}
	if title == "" {
		title = m.OriginTitle
	}
	if title == "" {
		title = body
		if len([]rune(title)) > 40 {
			title = string([]rune(title)[:40]) + "…"
		}
	}
	return title, body
}
