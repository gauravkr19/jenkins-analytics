package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gauravkr19/jenkins-analytics/internal/db"
	"github.com/gauravkr19/jenkins-analytics/internal/jenkins"
	"github.com/gauravkr19/jenkins-analytics/models"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	DB *db.DB
}

func (h *Handler) GetRecentBuilds(c *gin.Context) {
	builds, err := h.DB.FetchRecentBuilds(10)
	if err != nil {
		log.Printf("Failed to fetch recent builds: %v", err)
		c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{"error": "Failed to fetch recent builds"})
		return
	}

	c.HTML(http.StatusOK, "builds/recent.tmpl", gin.H{
		"builds": builds,
	})
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

func extractUserID(actions []jenkins.Action) string {
	for _, action := range actions {
		for _, cause := range action.Causes {
			if cause.UserID != "" {
				return cause.UserID
			}
		}
	}
	return "unknown@jenkins"
}

func extractProjectPathFromURL(jobURL, baseURL string, buildNumber int) string {
	// Remove base URL prefix
	trimmed := strings.TrimPrefix(jobURL, baseURL)
	// Remove trailing slash and build number
	trimmed = strings.TrimSuffix(trimmed, fmt.Sprintf("/%d/", buildNumber))
	// Remove leading /job/
	trimmed = strings.TrimPrefix(trimmed, "/job/")
	// Convert /job/ segments to /
	projectPath := strings.ReplaceAll(trimmed, "/job/", "/")
	return projectPath
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch builds from Jenkins"})
		return
	}

	saved, failed := 0, 0
	var failedBuilds []int

	for _, b := range builds {
		userID := extractUserID(b.Actions)
		if userID == "unknown@jenkins" {
			log.Printf("User ID not found for build #%d: %+v", b.Number, b.Actions)
		}

		dbModel := &models.Build{
			BuildNumber: b.Number,
			ProjectName: b.ProjectName,
			ProjectPath: extractProjectPathFromURL(b.URL, os.Getenv("JENKINS_URL"), b.Number),
			UserID:      userID,
			Status:      b.Result,
			Timestamp:   time.UnixMilli(b.Timestamp),
			DurationMS:  b.Duration,
			Branch:      "main",
			JobURL:      b.URL,
		}

		if err := h.DB.InsertBuild(dbModel); err != nil {
			log.Printf("Failed to insert build #%d: %v", b.Number, err)
			failed++
			failedBuilds = append(failedBuilds, b.Number)
			continue
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
