package api

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gauravkr19/jenkins-analytics/internal/db"
	"github.com/gauravkr19/jenkins-analytics/internal/jenkins"
	"github.com/gauravkr19/jenkins-analytics/models"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	DB *db.DB
}

// POST /builds
func (h *Handler) CreateBuild(c *gin.Context) {
	var build models.Build

	if err := c.ShouldBindJSON(&build); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.DB.InsertBuild(&build); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, build)
}

// GET /builds/:id
func (h *Handler) GetBuild(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid build ID"})
		return
	}

	build, err := h.DB.GetBuildByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, build)
}

func extractUserID(actions []map[string]interface{}) string {
	for _, action := range actions {
		if userID, ok := action["userId"].(string); ok {
			return userID
		}
	}
	return "unknown@jenkins"
}

func (h *Handler) FetchAndStoreBuilds(c *gin.Context) {
	client := jenkins.NewJenkinsClient(
		os.Getenv("JENKINS_URL"),
		os.Getenv("JENKINS_USER"),
		os.Getenv("JENKINS_TOKEN"),
	)

	builds, err := client.FetchBuilds()
	if err != nil {
		log.Printf("Error fetching builds: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch builds from Jenkins",
		})
		return
	}

	saved := 0
	failed := 0
	failedBuilds := []int{}

	for _, b := range builds {
		userID := extractUserID(b.Actions)
		if userID == "" {
			userID = "unknown@jenkins"
		}

		dbModel := &models.Build{
			BuildNumber: b.Number,
			ProjectName: b.ProjectName,
			UserID:      userID,
			Status:      b.Result,
			Result:      b.Result,
			Timestamp:   time.UnixMilli(b.Timestamp),
			DurationMS:  b.Duration,
			Branch:      "main", // Placeholder
			JobURL:      b.URL,
		}

		if err := h.DB.InsertBuild(dbModel); err != nil {
			log.Printf("Failed to insert build #%d: %v", b.Number, err)
			failed++
			failedBuilds = append(failedBuilds, b.Number)
			continue
		}

		head, tail, err := client.FetchConsoleLog(b.URL)
		if err != nil {
			log.Printf("Failed to fetch console log for build #%d: %v", b.Number, err)
			// still count the build as saved, just skip log
			saved++
			continue
		}

		logEntry := &models.BuildLog{
			BuildNumber:    b.Number,
			ProjectName:    b.ProjectName,
			ConsoleLogHead: head,
			ConsoleLogTail: tail,
		}

		if err := h.DB.InsertBuildLog(logEntry); err != nil {
			log.Printf("Failed to insert console log for build #%d: %v", b.Number, err)
		}

		saved++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Fetched and stored builds",
		"total_builds":   len(builds),
		"saved_builds":   saved,
		"failed_builds":  failed,
		"failed_numbers": failedBuilds,
	})
}
