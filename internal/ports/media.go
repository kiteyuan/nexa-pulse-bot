package ports

import "context"

// ImageUploader uploads raw image bytes and returns a public https URL.
type ImageUploader interface {
	Upload(ctx context.Context, name string, data []byte) (string, error)
}
