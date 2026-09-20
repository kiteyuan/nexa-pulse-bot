package process

import (
	"strings"
)

func Passes(text string, minLength int, keywords []string, hasMedia bool) (bool, string) {
	body := strings.TrimSpace(text)
	if body == "" && !hasMedia {
		return false, "内容过短"
	}
	if body != "" && !hasMedia && len([]rune(body)) < minLength {
		return false, "内容过短"
	}
	lower := strings.ToLower(body)
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw != "" && strings.Contains(lower, strings.ToLower(kw)) {
			return false, "命中屏蔽词: " + kw
		}
	}
	return true, ""
}

func FallbackTitle(content string) string {
	line := strings.TrimSpace(content)
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = strings.Join(strings.Fields(line), " ")
	if line == "" {
		return "资讯速递"
	}
	r := []rune(line)
	if len(r) > 40 {
		return string(r[:40]) + "…"
	}
	return line
}
