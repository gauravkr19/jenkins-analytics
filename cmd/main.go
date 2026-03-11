package main

import (
	"log"
	"os"
	"time"

	"github.com/gauravkr19/jenkins-analytics/internal/api"
	"github.com/gauravkr19/jenkins-analytics/internal/config"
	"github.com/gauravkr19/jenkins-analytics/internal/db"
	"github.com/gauravkr19/jenkins-analytics/internal/jenkins"
	"github.com/gauravkr19/jenkins-analytics/internal/poller"
	"github.com/gauravkr19/jenkins-analytics/internal/sync"
	"github.com/gauravkr19/jenkins-analytics/internal/web"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load and validate config
	cfg := config.LoadEnvConfig()
	// drConfig := config.DataRetentionConfig()

	// Step 1: Connect to PostgreSQL
	database, err := db.NewDB(cfg.DSN)

	// database.BackfillEnvColumn()
	// Step 2: Initial Build for first run only
	if err := sync.SyncInitialBuildsIfNeeded(database); err != nil {
		log.Fatalf("Initial sync failed: %v", err)
	}

	jenkinsClient := jenkins.NewJenkinsClient(
		os.Getenv("JENKINS_URL"),
		os.Getenv("JENKINS_USER"),
		os.Getenv("JENKINS_TOKEN"),
	)
	// Step 3: Incremental Build to add additional build records
	poller.StartIncrementalPoller(database, jenkinsClient, 30*time.Minute)
	// patches the status of the builds which are empty
	poller.StartStatusPatcher(database, jenkinsClient, 3*time.Hour, 100)
	// Data retention, delete records over config.RetentionConfig.MaxRecords
	poller.DeletionRoutine(database, jenkinsClient, 3*time.Hour, config.DataRetentionConfig())

	// Step 4: Setup Gin routes
	handler := &api.Handler{DB: database}
	r := gin.Default()

	r.Use(gin.Logger())
	r.Use(func(c *gin.Context) {
		log.Printf("[REQ] %s", c.Request.URL.Path)
		c.Next()
	})
	r.Static("/static", "./internal/web/static")

	// Load templates via extracted function
	tmpl, err := web.LoadTemplates()
	if err != nil {
		log.Fatalf("Template loading failed: %v", err)
	}
	r.SetHTMLTemplate(tmpl) // Register the final composed template with Gin

	// Register handler
	// r.GET("/test-folder-view", handler.RenderFolderTest)
	r.GET("/", handler.RenderHome)
	r.GET("/builds/filter", handler.FilterBuildsByTime) // range-based or custom range

	r.GET("/builds/export", handler.ExportBuildsToExcel)
	r.GET("/builds/folder", handler.RenderBuildsByFolder)
	r.GET("/builds/folder/*projectPath", handler.GetPipelineBuilds)

	// r.GET("/builds/folder/:folder/:app/:pipeline", handler.GetPipelineBuilds)

	// r.GET("/builds/recent", handler.GetRecentBuilds)
	// r.GET("/builds/:id", handler.GetBuild)
	// r.GET("/builds/export/daterange", handler.ExportBuildsToExcel) // from-to-based (same handler)

	// r.GET("/builds/export", handler.ExportBuildsToExcel)
	// r.GET("/builds/filter/daterange", handler.FilterBuildsByDateRange)
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
	log.Println("Server running at http://localhost:8091")
	if err := r.Run("0.0.0.0:8091"); err != nil {
		log.Fatalf("Gin server failed: %v", err)
	}
}
