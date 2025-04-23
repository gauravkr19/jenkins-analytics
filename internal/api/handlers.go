package api

import (
	"log"
	"net/http"
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

func (h *Handler) FetchAndStoreBuilds(c *gin.Context) {
	client := jenkins.NewJenkinsClient("https://ci.jenkins.io", "", "") // You can later use env vars

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
		dbModel := &models.Build{
			BuildNumber:    b.Number,
			ProjectName:    b.ProjectName,
			UserID:         "unknown@jenkins",
			Status:         b.Result,
			Result:         b.Result,
			Timestamp:      time.UnixMilli(b.Timestamp), // Corrected
			DurationMS:     b.Duration,
			Branch:         "main",     // Placeholder
			CommitID:       "abcd1234", // Placeholder
			JobURL:         b.URL,
			ConsoleLogHead: "",
			ConsoleLogTail: "",
			ErrorMessage:   "",
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
