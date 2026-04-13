package gopherquest

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var staticEmbedded embed.FS

// StaticRoot 为 static/ 子目录，与 HTTP 路径 /static/ + StripPrefix 对齐，否则会出现 404 且 MIME 为 text/plain。
var StaticRoot fs.FS

func init() {
	sub, err := fs.Sub(staticEmbedded, "static")
	if err != nil {
		panic("gopherquest: embed static: " + err.Error())
	}
	StaticRoot = sub
}
