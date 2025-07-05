// File: web/templates/template.go
package web

import (
	"html/template"
	"path/filepath"
)

// FuncMap can be extended globally
var FuncMap = template.FuncMap{
	"div": func(a, b int64) int64 {
		if b == 0 {
			return 0
		}
		return a / b
	},
}

// LoadTemplates loads all .tmpl files including nested folders like builds/
func LoadTemplates() (*template.Template, error) {
	tmpl := template.New("").Funcs(FuncMap)

	// Load base templates (including sidebar, home, base)
	baseTemplates, err := filepath.Glob("web/templates/*.tmpl")
	if err != nil {
		return nil, err
	}
	if len(baseTemplates) > 0 {
		tmpl, err = tmpl.ParseFiles(baseTemplates...)
		if err != nil {
			return nil, err
		}
	}

	// Load nested folders like web/templates/builds/
	nestedTemplates, err := filepath.Glob("web/templates/*/*.tmpl")
	if err != nil {
		return nil, err
	}
	if len(nestedTemplates) > 0 {
		_, err = tmpl.ParseFiles(nestedTemplates...)
		if err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}
