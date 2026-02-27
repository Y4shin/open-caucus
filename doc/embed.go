package doc

import (
	"embed"
	"io/fs"
)

//go:embed content assets
var embedded embed.FS

func ContentFS() fs.FS {
	sub, err := fs.Sub(embedded, "content")
	if err != nil {
		panic("embedded docs content subtree missing")
	}
	return sub
}

func AssetsFS() fs.FS {
	sub, err := fs.Sub(embedded, "assets")
	if err != nil {
		panic("embedded docs assets subtree missing")
	}
	return sub
}
