package proxy

import (
	"bytes"
	"embed"
	"html/template"
	"log"
	"sync"
)

//go:embed templates/error.html
var errorTemplatesFS embed.FS

// errorTmpl is the parsed error page template, loaded once at init.
var (
	errorTmpl     *template.Template
	errorTmplOnce sync.Once
)

func initErrorTemplate() {
	var err error
	errorTmpl, err = template.ParseFS(errorTemplatesFS, "templates/error.html")
	if err != nil {
		log.Fatalf("[proxy] failed to parse error template: %v", err)
	}
}

// errorPageData holds the template variables for the public error page.
type errorPageData struct {
	Title      string
	StatusCode int
	Message    string
	IconSVG    template.HTML // raw SVG, safe to render unescaped
}

// renderErrorPage executes the embedded error page template and returns the HTML string.
func renderErrorPage(data errorPageData) string {
	errorTmplOnce.Do(initErrorTemplate)

	var buf bytes.Buffer
	if err := errorTmpl.Execute(&buf, data); err != nil {
		log.Printf("[proxy] error template render failed: %v", err)
		return fallbackErrorHTML(data)
	}
	return buf.String()
}

// fallbackErrorHTML is a minimal plain-text fallback if the template fails.
func fallbackErrorHTML(data errorPageData) string {
	return "<h1>" + data.Title + "</h1><p>" + data.Message + "</p>"
}
