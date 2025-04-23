package main

import (
	"log"

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
	r.POST("/builds", handler.CreateBuild)
	r.GET("/builds/:id", handler.GetBuild)
	r.GET("/jenkins/builds/fetch", handler.FetchAndStoreBuilds)

	// Step 4: Start HTTP server
	log.Println("Server running at http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Gin server failed: %v", err)
	}
}
