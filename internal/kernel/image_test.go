package kernel

import "testing"

func TestSniffImageExt(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		data []byte
		ext  string
		ok   bool
	}{
		{"jpeg", []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0, 0, 0, 0, 0}, ".jpg", true},
		{"png", append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 4)...), ".png", true},
		{"gif", []byte("GIF89axxxxxx"), ".gif", true},
		{"webp", []byte("RIFF....WEBP"), ".webp", true},
		{"short", []byte{0xff, 0xd8}, "", false},
		{"html", []byte("<!DOCTYPE html>"), "", false},
	}
	for _, tc := range cases {
		ext, ok := SniffImageExt(tc.data)
		if ok != tc.ok || ext != tc.ext {
			t.Fatalf("%s: got %q %v want %q %v", tc.name, ext, ok, tc.ext, tc.ok)
		}
	}
}
