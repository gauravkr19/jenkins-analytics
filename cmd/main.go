package main

import (
	"log"
	"text/template"

	"github.com/gauravkr19/jenkins-analytics/internal/api"
	"github.com/gauravkr19/jenkins-analytics/internal/db"
	"github.com/gin-gonic/gin"
)

func main() {
	// Step 1: Connect to PostgreSQL
	dsn := "postgres://jenkins:jenkins@localhost:5432/jenkins?sslmode=disable"
	database, err := db.NewDB(dsn)
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}

	// Step 2: Initialize HTTP handlers
	handler := &api.Handler{DB: database}

	// Step 3: Setup Gin routes
	r := gin.Default()
	r.LoadHTMLGlob("templates/**/*.tmpl")
	r.GET("/builds/recent", handler.GetRecentBuilds)
	// r.GET("/builds/:id", handler.GetBuild)
	// r.GET("/jenkins/builds/fetch", handler.FetchAndStoreBuilds)

	r.SetFuncMap(template.FuncMap{
		"div": func(a, b int64) int64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
	})

	// | --------------------- | --------------------------------------- |
	// | Get recent builds     | `GET /builds/recent`                    |
	// | Get single build      | `GET /builds/:id`                       |
	// | Filter builds         | `GET /builds?status=FAILED&project=abc` |
	// | Trigger Jenkins fetch | `POST /jenkins/builds/fetch`            |

	// | ------------------------- | ---------------------------- | ----------------------- |
	// | Sidebar → "Recent Builds" | `GET /builds?limit=50`       | `GetBuilds()`           |
	// | Tab → "Failed Builds"     | `GET /builds?status=FAILED`  | `GetBuilds()`           |
	// | Click on a Build          | `GET /builds/:id`            | `GetBuild()`            |
	// | "Refresh Builds" button   | `POST /jenkins/builds/fetch` | `FetchAndStoreBuilds()` |
	// | Search builds by project  | `GET /builds?project=infra`  | `GetBuilds()`           |

	// Step 4: Start HTTP server
	log.Println("Server running at http://localhost:8080")
	if err := r.Run(":8082"); err != nil {
		log.Fatalf("Gin server failed: %v", err)
	}
}
