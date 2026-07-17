//go:build embed_web

package web

import (
	"embed"
	"io/fs"
	"net/http"

	"mcsd/api"
)

//go:embed all:dist
var webFiles embed.FS

type spaHandler struct {
	fileSystem http.FileSystem
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, err := h.fileSystem.Open(r.URL.Path)
	if err == nil {
		stat, serr := f.Stat()
		f.Close()
		if serr == nil && !stat.IsDir() {
			http.FileServer(h.fileSystem).ServeHTTP(w, r)
			return
		}
	}
	r2 := *r
	r2.URL.Path = "/"
	http.FileServer(h.fileSystem).ServeHTTP(w, &r2)
}

func init() {
	sub, err := fs.Sub(webFiles, "dist")
	if err != nil {
		panic(err)
	}
	api.SetStaticFiles(spaHandler{http.FS(sub)})
}
