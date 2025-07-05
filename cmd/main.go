package main

import (
	"log"

	"github.com/gauravkr19/jenkins-analytics/internal/api"
	"github.com/gauravkr19/jenkins-analytics/internal/db"
	"github.com/gauravkr19/jenkins-analytics/internal/sync"
	"github.com/gauravkr19/jenkins-analytics/internal/web"
	"github.com/gin-gonic/gin"
)

func main() {
	// Step 1: Connect to PostgreSQL
	dsn := "postgres://jenkins:jenkins@postgresql-jenkins:5432/jenkins?sslmode=disable"
	database, err := db.NewDB(dsn)
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}

	// Step 2: Initial Sync Check
	if err := sync.SyncInitialBuildsIfNeeded(database); err != nil {
		log.Fatalf("Initial sync failed: %v", err)
	}

	// Step 3: Setup Gin routes
	handler := &api.Handler{DB: database}
	r := gin.Default()

	// Load templates via extracted function
	tmpl, err := web.LoadTemplates()
	if err != nil {
		log.Fatalf("Template loading failed: %v", err)
	}
	r.SetHTMLTemplate(tmpl) // Register the final composed template with Gin

	// r.GET("/builds/recent", handler.GetRecentBuilds)
	// r.GET("/builds/:id", handler.GetBuild)

	// Register handler
	r.GET("/builds/filter", handler.FilterBuildsByTime)
	r.GET("/", handler.RenderHome)
	// r.GET("/jenkins/builds/fetch", handler.FetchAndStoreBuilds)

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
	log.Println("Server running at http://localhost:8083")
	if err := r.Run("0.0.0.0:8083"); err != nil {
		log.Fatalf("Gin server failed: %v", err)
	}
}
