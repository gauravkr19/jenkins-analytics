// File: web/templates/template.go
package web

import (
<<<<<<< Updated upstream
=======
	"fmt"
>>>>>>> Stashed changes
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
<<<<<<< Updated upstream
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
=======
	"add": func(a, b int) int {
		return a + b
	},
	"sub": func(a, b int) int {
		return a - b
	},
	// seq(start, end) returns a slice [start, start+1, …, end]
	"seq": func(start, end int) []int {
		if end < start {
			return []int{}
		}
		out := make([]int, end-start+1)
		for i := range out {
			out[i] = start + i
		}
		return out
	},
}


func LoadTemplates() (*template.Template, error) {
	tmpl := template.New("").Funcs(FuncMap)

	allTemplates, err := filepath.Glob("web/templates/**/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to find templates: %w", err)
	}

	if len(allTemplates) == 0 {
		return nil, fmt.Errorf("no templates found")
	}

	_, err = tmpl.ParseFiles(allTemplates...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	for _, t := range tmpl.Templates() {
		fmt.Println("[TEMPLATE LOADED]:", t.Name())
>>>>>>> Stashed changes
	}

	return tmpl, nil
}
