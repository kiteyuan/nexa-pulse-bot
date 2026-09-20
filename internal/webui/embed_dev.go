//go:build dev

package webui

import "io/fs"

func Public() fs.FS { return nil }
func Admin() fs.FS { return nil }
