package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

func StaticHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filePath := strings.TrimPrefix(r.URL.Path, "/static/")
		if filePath == "" {
			http.NotFound(w, r)
			return
		}

		var data []byte
		var found bool

		loader := GetAssetLoader()
		if loader != nil && loader.IsEnabled() {
			data, found = loader.GetFile(filePath)
		}

		if !found {
			embeddedData, err := fs.ReadFile(staticFiles, path.Join("static", filePath))
			if err != nil {
				if loader != nil && loader.IsEnabled() {
					if customData, customFound := loader.GetFile(filePath); customFound {
						data = customData
						found = true
					}
				}
				if !found {
					http.NotFound(w, r)
					return
				}
			} else {
				data = embeddedData
			}
		}

		contentType := GetContentType(filePath)
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		w.Write(data)
	}
}
