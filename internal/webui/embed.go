//go:build !dev

package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:public
var public embed.FS

//go:embed all:admin
var admin embed.FS

func Public() fs.FS { return public }
func Admin() fs.FS { return admin }
