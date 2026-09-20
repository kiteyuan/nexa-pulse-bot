package kernel

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPresentFallbacks(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]string{"title": "T", "body": "B"})
	title, body := Present(Message{LLMResult: raw, Content: "raw", OriginTitle: "orig"})
	if title != "T" || body != "B" {
		t.Fatalf("%q %q", title, body)
	}

	title, body = Present(Message{Content: "only body", OriginTitle: "orig"})
	if title != "orig" || body != "only body" {
		t.Fatalf("%q %q", title, body)
	}

	long := strings.Repeat("字", 45)
	title, body = Present(Message{Content: long})
	if body != long {
		t.Fatal(body)
	}
	runes := []rune(title)
	if len(runes) != 41 || runes[len(runes)-1] != '…' {
		t.Fatalf("title=%q len=%d", title, len(runes))
	}
}
