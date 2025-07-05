package api

import (
	"log"
	"net/http"
	"os"
	"strconv"

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

func (h *Handler) FetchAndStoreBuilds(c *gin.Context) {
	client := jenkins.NewJenkinsClient(
		os.Getenv("JENKINS_URL"),
		os.Getenv("JENKINS_USER"),
		os.Getenv("JENKINS_TOKEN"),
	)

	saved, failed, failedBuilds, err := jenkins.FetchAndStoreBuilds(h.DB, client)
	if err != nil {
		log.Printf("Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Fetched and stored builds",
		"saved_builds":   saved,
		"failed_builds":  failed,
		"failed_numbers": failedBuilds,
	})
}

// Handler already has DB *db.DB
func (h *Handler) FilterBuildsByTime(c *gin.Context) {
	rangeKey := c.Query("range")
	from, to, err := ParseTimeRange(rangeKey)
	log.Printf("Received time range param: %s", rangeKey)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid time range: %v", err)
		return
	}

	builds, err := h.DB.GetBuildsByTime(from, to)
	if err != nil {
		log.Printf("DB error: %v", err)
		c.String(http.StatusInternalServerError, "Failed to fetch builds")
		return
	}

	// c.HTML(http.StatusOK, "index.tmpl", data)
	c.HTML(http.StatusOK, "builds/table.tmpl", gin.H{
		"Builds": builds,
	})
}

func (h *Handler) RenderHome(c *gin.Context) {
	c.HTML(http.StatusOK, "base", gin.H{
		"Title": "Home",
	})
}
