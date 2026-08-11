package render

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LoadTemplates parses and returns all HTML views and partials.
func LoadTemplates() *template.Template {
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

	t := template.New("").Funcs(funcMap)
	patterns := []string{
		"templates/layouts/*.html",
		"templates/client/*.html",
		"templates/admin/*.html",
		"templates/partials/*.html",
	}

	for _, p := range patterns {
		files, err := filepath.Glob(p)
		if err != nil {
			continue
		}
		for _, f := range files {
			name := strings.TrimPrefix(f, "templates/")
			name = filepath.ToSlash(name)
			content, err := os.ReadFile(f)
			if err != nil {
				log.Printf("[Render Error] read %s: %v", f, err)
				continue
			}
			if _, err := t.New(name).Parse(string(content)); err != nil {
				log.Printf("[Render Error] parse %s: %v", name, err)
			}
		}
	}
	return t
}
