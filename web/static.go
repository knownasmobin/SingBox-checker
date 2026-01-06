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

		data, err := fs.ReadFile(staticFiles, path.Join("static", filePath))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		contentType := "application/octet-stream"
		switch {
		case strings.HasSuffix(filePath, ".css"):
			contentType = "text/css; charset=utf-8"
		case strings.HasSuffix(filePath, ".js"):
			contentType = "application/javascript; charset=utf-8"
		case strings.HasSuffix(filePath, ".woff2"):
			contentType = "font/woff2"
		case strings.HasSuffix(filePath, ".woff"):
			contentType = "font/woff"
		case strings.HasSuffix(filePath, ".svg"):
			contentType = "image/svg+xml"
		case strings.HasSuffix(filePath, ".png"):
			contentType = "image/png"
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		w.Write(data)
	}
}
