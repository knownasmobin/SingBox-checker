package web

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"time"
)

//go:embed templates/*.html
var content embed.FS

var indexTmpl *template.Template

func init() {
	var err error
	funcMap := template.FuncMap{
		"formatLatency": func(d time.Duration) string {
			if d == 0 {
				return "n/a"
			}
			return fmt.Sprintf("%dms", d.Milliseconds())
		},
	}

	indexTmpl, err = template.New("index.html").Funcs(funcMap).ParseFS(content, "templates/*.html")
	if err != nil {
		panic(err)
	}
}

type PageData struct {
	Version                    string
	Host                       string
	Port                       string
	CheckInterval              int
	IPCheckUrl                 string
	SimulateLatency            bool
	CheckMethod                string
	StatusCheckUrl             string
	DownloadUrl                string
	Timeout                    int
	SubscriptionUpdate         bool
	SubscriptionUpdateInterval int
	StartPort                  int
	Instance                   string
	PushUrl                    string
	Endpoints                  []EndpointInfo
	ShowServerDetails          bool
	IsPublic                   bool
	SubscriptionName           string
}

func RenderIndex(w io.Writer, data PageData) error {
	return indexTmpl.Execute(w, data)
}
