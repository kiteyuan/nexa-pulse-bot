package kernel

import "testing"

func TestThemeSlug(t *testing.T) {
	if got := ThemeSlug("Hello World"); got != "hello-world" {
		t.Fatalf("got %q", got)
	}
	if got := ThemeSlug("科技"); got != "" {
		t.Fatalf("got %q", got)
	}
}
