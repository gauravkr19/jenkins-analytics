// File: web/templates/template.go
package web

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
)

// FuncMap can be extended globally
var FuncMap = template.FuncMap{
	"div": func(a, b int64) int64 {
		if b == 0 {
			return 0
		}
		return a / b
	},
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
	"mul": func(a, b int) int { return a * b },
	"upper": func(s string) string { return strings.ToUpper(s) },

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
	"slice": func(s string, start, end int) string {
		if len(s) < start || len(s) < end {
			return s
		}
		return s[start:end]
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

	// for _, t := range tmpl.Templates() {
	// 	fmt.Println("[TEMPLATE LOADED]:", t.Name())
	// }

	return tmpl, nil
}
