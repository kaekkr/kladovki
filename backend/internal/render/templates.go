package render

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin/render"
)

type CustomRender struct {
	templates map[string]*template.Template
}

// CustomHTML implements gin/render.Render interface
type CustomHTML struct {
	Template *template.Template
	Name     string
	Data     any
}

func (r CustomHTML) Render(w http.ResponseWriter) error {
	r.WriteContentType(w)
	return r.Template.ExecuteTemplate(w, r.Name, r.Data)
}

func (r CustomHTML) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{"text/html; charset=utf-8"}
	}
}

func (r CustomRender) Instance(name string, data any) render.Render {
	tmpl, exists := r.templates[name]
	if !exists {
		log.Printf("[Render Error] template not found: %s", name)
	}
	return CustomHTML{
		Template: tmpl,
		Name:     "layouts/base",
		Data:     data,
	}
}

func LoadTemplates() render.HTMLRender {
	funcMap := template.FuncMap{
		"formatMoney": func(amount int64) string {
			return fmt.Sprintf("%d ₸", amount)
		},
		"formatDate": func(t *time.Time) string {
			if t == nil {
				return "-"
			}
			return t.Format("02.01.2006")
		},
	}

	components, _ := filepath.Glob("templates/components/**/*.html")
	layouts, _ := filepath.Glob("templates/layouts/*.html")
	pages, err := filepath.Glob("templates/pages/**/*.html")
	if err != nil {
		log.Printf("[Render Error] glob pages: %v", err)
	}

	sharedFiles := append(layouts, components...)
	templatesMap := make(map[string]*template.Template)

	for _, page := range pages {
		relPath := filepath.ToSlash(page)
		name := strings.TrimPrefix(relPath, "templates/")
		name = strings.TrimSuffix(name, ".html")

		files := append([]string{page}, sharedFiles...)

		tmpl, err := template.New(name).Funcs(funcMap).ParseFiles(files...)
		if err != nil {
			log.Printf("[Render Error] parsing bundle %s: %v", name, err)
			continue
		}

		templatesMap[name] = tmpl
	}

	return CustomRender{templates: templatesMap}
}
