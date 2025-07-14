package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gauravkr19/jenkins-analytics/internal/db"
	"github.com/gauravkr19/jenkins-analytics/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
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

<<<<<<< Updated upstream
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
=======
func (h *Handler) FilterBuildsByTime(c *gin.Context) {
	rangeKey := c.Query("range")
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var from, to time.Time
	var err error
	var mode string

	switch {
	case fromStr != "" && toStr != "":
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid from date")
			return
		}
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid to date")
			return
		}
		to = to.Add(24 * time.Hour)
		mode = "date_range"
	case rangeKey != "":
		from, to, err = ParseTimeRange(rangeKey)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid time range")
			return
		}
		mode = "named_range"
	default:
		c.String(http.StatusBadRequest, "Missing parameters")
		return
	}

	// Pagination params
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "35")
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	// Fetch builds and total count
	totalCount, err := h.DB.CountBuildsByTime(from, to)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to count builds")
		return
	}
	totalPages := (totalCount + limit - 1) / limit

	builds, err := h.DB.GetBuildsByTime(from, to, limit, offset)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to fetch builds")
		return
	}

	data := gin.H{
		"Builds":      builds,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Limit":       limit,
	}
	data["TotalPages"] = totalPages

	log.Printf("Fetching page=%d limit=%d, total builds=%d, totalPages=%d", page, limit, totalCount, totalPages)

	if mode == "date_range" {
		data["FromDate"] = fromStr
		data["ToDate"] = toStr
	} else {
		data["Range"] = rangeKey
	}

	// c.HTML(http.StatusOK, "builds/partial_response.tmpl", data)
	hx := c.GetHeader("HX-Request")
	isHX := strings.ToLower(hx) == "true" || hx != ""
	var tmplName string
	if isHX {
		tmplName = "builds/partial_response"
	} else {
		tmplName = "builds/index"
	}
	c.HTML(http.StatusOK, tmplName, data)
}

func (h *Handler) RenderHome(c *gin.Context) {
	c.HTML(http.StatusOK, "base", gin.H{
		"Title": "Home",
	})
}

func (h *Handler) ExportBuildsToExcel(c *gin.Context) {
	rangeKey := c.Query("range")
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var from, to time.Time
	var err error
	var label string

	switch {
	case fromStr != "" && toStr != "":
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid from date")
			return
		}
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid to date")
			return
		}
		to = to.Add(24 * time.Hour)
		label = fmt.Sprintf("%s_to_%s", fromStr, toStr)

	case rangeKey != "":
		from, to, err = ParseTimeRange(rangeKey)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid range")
			return
		}
		label = rangeKey

	default:
		c.String(http.StatusBadRequest, "Missing filter parameters")
		return
	}

	// set limit to a higher number to export the complete range
	builds, err := h.DB.GetBuildsByTime(from, to, 999999, 0)
	if err != nil {
>>>>>>> Stashed changes
		c.String(http.StatusInternalServerError, "Failed to fetch builds")
		return
	}

<<<<<<< Updated upstream
	// c.HTML(http.StatusOK, "index.tmpl", data)
	c.HTML(http.StatusOK, "builds/table.tmpl", gin.H{
		"Builds": builds,
	})
}

func (h *Handler) RenderHome(c *gin.Context) {
	c.HTML(http.StatusOK, "base", gin.H{
		"Title": "Home",
	})
=======
	// Create the Excel file
	f := excelize.NewFile()
	sheet := "Builds"
	f.NewSheet(sheet)

	// Header
	f.SetSheetRow(sheet, "A1", &[]interface{}{"#", "Project", "Status", "User", "Time", "Duration", "JobURL", "Trigger"})

	// Data rows
	for i, b := range builds {
		row := []interface{}{
			b.BuildNumber,
			b.ProjectName,
			b.Status,
			b.UserID,
			b.Timestamp.Format("2006-01-02 15:04"),
			b.DurationMS,
			b.JobURL,
			b.TriggerType,
		}
		cell := fmt.Sprintf("A%d", i+2)
		f.SetSheetRow(sheet, cell, &row)
	}

	// Generate timestamped filename
	now := time.Now().Format("2006-01-02_1504")
	filename := fmt.Sprintf("builds_%s_%s.xlsx", label, now)

	// Proper headers
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Expires", "0")

	// Write file to response
	if err := f.Write(c.Writer); err != nil {
		log.Printf("Error writing Excel: %v", err)
		c.String(http.StatusInternalServerError, "Failed to write Excel file")
	}
}

// GET /builds/folder/:folder/:app/:pipeline, pipeline_builds.tmpl - shows table
func (h *Handler) GetPipelineBuilds(c *gin.Context) {
	rawPath := c.Param("projectPath")
	fullPath := strings.TrimPrefix(rawPath, "/")

	fmt.Printf("DEBUG fullPathRaw: %+v\n", fullPath)
	builds, err := h.DB.GetBuildsByProjectPath(fullPath)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: %v", err)
		return
	}
	data := gin.H{
		"ProjectPath": fullPath,
		"Builds":      builds,
	}
	// fmt.Printf("DEBUG PIPELINE: %+v\n", data)
	if c.GetHeader("HX-Request") == "true" {
		// HTMX: only send the table fragment
		c.HTML(http.StatusOK, "pipeline_partial.tmpl", data)
	} else {
		// Full‐page load: wrap in base layout
		c.HTML(http.StatusOK, "base", data)
	}
}

// GET /builds/folder - now uses recursive tree
func (h *Handler) RenderBuildsByFolder(c *gin.Context) {
	tree, err := h.DB.GetBuildTree()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: %v", err)
		return
	}

	data := gin.H{"BuildTree": tree}
	// fmt.Printf("DEBUG FOLDER-VIEW: %+v\n", data)

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "folder_partial.tmpl", data)
	} else {
		c.HTML(http.StatusOK, "base", data)
	}
>>>>>>> Stashed changes
}
