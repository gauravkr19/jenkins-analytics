package api

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

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

func (h *Handler) FilterBuildsByTime(c *gin.Context) {
    rangeKey := c.Query("range")
    fromStr := c.Query("from")
    toStr := c.Query("to")
	sortBy := c.DefaultQuery("sort_by", "timestamp") // faillback to timestamp if sort_by is missing
	order := c.DefaultQuery("order", "desc")
    searchBy  := strings.ToLower(c.DefaultQuery("search_by", ""))
    searchTerm:= strings.TrimSpace(strings.ToLower(c.DefaultQuery("search_term", "")))	

    var from, to time.Time
    var err error
    var mode string

    switch {
    // —— NEW: search without date/range —— 
    case searchBy != "" && searchTerm != "":
        // full‐history search only when user provided both parts
        from = time.Unix(0, 0)           // epoch start
        to   = time.Now().Add(time.Second) // just beyond now
        mode = "search_only"

    case fromStr != "" && toStr != "":
        // existing custom date‐range
        from, err = time.Parse("2006-01-02", fromStr)
        if err != nil { c.String(http.StatusBadRequest, "Invalid from date"); return }
        to, err = time.Parse("2006-01-02", toStr)
        if err != nil { c.String(http.StatusBadRequest, "Invalid to date"); return }
        to = to.Add(24*time.Hour).Add(-time.Nanosecond)
        mode = "date_range"

    case rangeKey != "":
        // existing named range
        dr, err := GetDateRange(rangeKey)
        if err != nil { c.String(http.StatusBadRequest, "Invalid time range"); return }
        from, to = dr.From, dr.To
        mode = "named_range"

    default:
        c.String(http.StatusBadRequest, "Missing parameters")
        return
    }

    // Fetch **all** builds in this date/range, ignoring pagination
    allBuilds, err := h.DB.GetBuildsByTime(from, to, math.MaxInt32, 0, sortBy, order)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to fetch builds: %v", err)
        return
    }

    // --- New: apply search filter ---
    if searchBy != "" && searchTerm != "" {
        filtered := allBuilds[:0]
        for _, b := range allBuilds {
            var field string
            switch searchBy {
            case "env":
                field = b.Env
            case "project_path":
                field = b.ProjectPath
            case "user_id":
                field = b.UserID
            default:
                continue
            }
            if strings.Contains(strings.ToLower(field), searchTerm) {
                filtered = append(filtered, b)
            }
        }
        allBuilds = filtered
    }

    // --- Recount & paginate on filtered+sorted slice ---
    totalCount := len(allBuilds)
    page, _   := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _  := strconv.Atoi(c.DefaultQuery("limit", "35"))
    if page < 1 { page = 1 }
    totalPages := (totalCount + limit - 1) / limit
    offset     := (page - 1) * limit
    end        := offset + limit
    if end > totalCount { end = totalCount }
    builds     := allBuilds[offset:end]

    // Prepare template data
    data := gin.H{
        "Builds":        builds,
        "CurrentPage":   page,
        "TotalPages":    totalPages,
        "Limit":         limit,
        "CurrentSortBy": sortBy,
        "CurrentOrder":  order,
        "SearchBy":      searchBy,
        "SearchTerm":    searchTerm,
    }
	data["TotalPages"] = totalPages

	log.Printf("Fetching page=%d limit=%d, total builds=%d, totalPages=%d", page, limit, totalCount, totalPages)

	if mode == "date_range" {
		data["FromDate"] = fromStr
		data["ToDate"] = toStr
	} else {
		data["Range"] = rangeKey
	}

    // Choose partial or full
    if c.GetHeader("HX-Request") != "" {
        c.HTML(http.StatusOK, "builds/partial_response", data)
    } else {
        c.HTML(http.StatusOK, "base", data)
    }
}

func (h *Handler) RenderHome(c *gin.Context) {
    c.HTML(http.StatusOK, "base", gin.H{
        "Title": "Home",
        "Home":  true,
		"CurrentPage": 1, // dummy values to pass to base template
		"TotalPages":  1,
		"Limit":       20,		
    })
}

// ExportBuildsToExcel handles exporting builds to an Excel file.
func (h *Handler) ExportBuildsToExcel(c *gin.Context) {
    rangeKey := c.Query("range")
    fromStr := c.Query("from")
    toStr := c.Query("to")
    project := c.Query("project")
	sortBy := c.DefaultQuery("sort_by", "timestamp") // faillback to timestamp if sort_by is missing
	order := c.DefaultQuery("order", "desc")	

    searchBy   := strings.ToLower(c.DefaultQuery("search_by", ""))
    searchTerm := strings.TrimSpace(strings.ToLower(c.DefaultQuery("search_term", "")))

	// Resolve date range or full‑history for search-only:
    var from, to time.Time
    var err error
    var label string
    var builds []models.Build

    // Determine filter mode
    switch {
    case searchBy != "" && searchTerm != "":
        from = time.Unix(0, 0)
        to   = time.Now()
		builds, err = h.DB.GetBuildsByTime(from, to, math.MaxInt32, 0, sortBy, order)

    case fromStr != "" && toStr != "":
        // Manual date range
        from, err = time.Parse("2006-01-02", fromStr)
        if err != nil {
            c.String(http.StatusBadRequest, "Invalid 'from' date")
            return
        }
        to, err = time.Parse("2006-01-02", toStr)
        if err != nil {
            c.String(http.StatusBadRequest, "Invalid 'to' date")
            return
        }		
        // Include entire 'to' day
        to = to.AddDate(0, 0, 1).Add(-time.Nanosecond)
        label = fmt.Sprintf("%s_to_%s", fromStr, toStr)
        builds, err = h.DB.GetBuildsByTime(from, to, math.MaxInt32, 0, sortBy, order)

    case rangeKey != "":
        // Named range
        dr, err := GetDateRange(rangeKey)
        if err != nil {
            c.String(http.StatusBadRequest, "Invalid range key")
            return
        }
        from, to = dr.From, dr.To
        label = rangeKey
        builds, err = h.DB.GetBuildsByTime(from, to, math.MaxInt32, 0, sortBy, order)

    case project != "":
        label = strings.ReplaceAll(project, "/", "_")

        builds, err = h.DB.GetBuildsByProjectPath(project)
        if err != nil {
            c.String(http.StatusInternalServerError, "Failed to fetch builds")
            return
        }		

	default:
		c.String(http.StatusBadRequest, "Missing filter parameters")
		return
	}



	// Create the Excel file
	f := excelize.NewFile()

	// Create & activate your single “Builds” sheet
	sheet := "Builds"
	f.NewSheet(sheet)
	active, err := f.GetSheetIndex(sheet); if err != nil {
		log.Printf("Error creating active sheet in Excel: %v", err)
	}
	f.SetActiveSheet(active)
	// Delete the default, empty “Sheet1”
	f.DeleteSheet("Sheet1")	

	// Header
	f.SetSheetRow(sheet, "A1", &[]interface{}{"#", "Build#", "Env", "Project", "Status", "User", "Time", "DurationMS", "JobURL", "Trigger", "GitRepoURL", "GitBranch", "CommitID"})

	// Data rows
	for i, b := range builds {
		row := []interface{}{
			i + 1,
			b.BuildNumber,
			b.Env,
			b.ProjectName,
			b.Status,
			b.UserID,
			b.Timestamp.Format("2006-01-02 15:04"),
			b.DurationMS,
			b.JobURL,
			b.TriggerType,
			b.GitRepo,
			b.Branch,
			b.CommitSHA,			
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

// GET "/builds/folder/*projectPath", pipeline_partial.tmpl - shows table
func (h *Handler) GetPipelineBuilds(c *gin.Context) {
    rawPath := c.Param("projectPath")
    fullPath := strings.TrimPrefix(rawPath, "/")

    page, _  := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
    if page < 1 { page = 1 }
    offset := (page - 1) * limit

    // fetch all, then sort in Go
    allBuilds, err := h.DB.GetBuildsByProjectPath(fullPath)
    if err != nil {
        c.String(http.StatusInternalServerError, "Error: %v", err)
        return
    }

    // Apply sorting
    sortBy := c.DefaultQuery("sort_by", "timestamp")
    order  := c.DefaultQuery("order", "desc")

    // perform in‐memory sort for folder view
    sort.SliceStable(allBuilds, func(i, j int) bool {
        a, b := allBuilds[i], allBuilds[j]
        asc := (order == "asc")

        switch sortBy {
        case "env":
            if asc { return a.Env < b.Env } else { return a.Env > b.Env }
        case "status":
            if asc { return a.Status < b.Status } else { return a.Status > b.Status }
        case "user_id":
            if asc { return a.UserID < b.UserID } else { return a.UserID > b.UserID }
        case "duration_ms":
            if asc { return a.DurationMS < b.DurationMS } else { return a.DurationMS > b.DurationMS }
        case "timestamp":
            if asc { return a.Timestamp.Before(b.Timestamp) } else { return a.Timestamp.After(b.Timestamp) }
        default:
            return a.Timestamp.After(b.Timestamp) // fallback
        }
    })

	// Paginate after sort+search
    total := len(allBuilds)
    totalPages := (total + limit - 1) / limit
    end := offset + limit
    if end > total { end = total }
    paged := allBuilds[offset:end]

    data := gin.H{
        "ProjectPath":   fullPath,
        "Builds":        paged,
        "CurrentPage":   page,
        "TotalPages":    totalPages,
        "Limit":         limit,
        "CurrentSortBy": sortBy,
        "CurrentOrder":  order,	
    }

    if c.GetHeader("HX-Request") == "true" {
        c.HTML(http.StatusOK, "pipeline_partial", data)
    } else {
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

        if c.GetHeader("HX-Request") == "true" {
                c.HTML(http.StatusOK, "folder_partial.tmpl", data)
        } else {
                c.HTML(http.StatusOK, "base", data)
        }
}

