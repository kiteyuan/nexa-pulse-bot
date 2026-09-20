package kernel

import "bytes"

// SniffImageExt returns a file extension (with leading dot) when data looks like
// JPEG, PNG, GIF, or WebP. ok is false for empty or unrecognized payloads.
func SniffImageExt(data []byte) (ext string, ok bool) {
	if len(data) < 12 {
		return "", false
	}
	switch {
	case bytes.HasPrefix(data, []byte{0xff, 0xd8, 0xff}):
		return ".jpg", true
	case bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}):
		return ".png", true
	case bytes.HasPrefix(data, []byte("GIF87a")) || bytes.HasPrefix(data, []byte("GIF89a")):
		return ".gif", true
	case bytes.HasPrefix(data, []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return ".webp", true
	default:
		return "", false
	}
}
