package sync

import (
	"fmt"
	"log"
	"os"

	"github.com/gauravkr19/jenkins-analytics/internal/db"
	"github.com/gauravkr19/jenkins-analytics/internal/jenkins"
)

// SyncInitialBuildsIfNeeded ensures DB has the initial Jenkins builds which runs once
func SyncInitialBuildsIfNeeded(database *db.DB) error {

	syncDone, err := database.IsInitialSyncDone()
	if err != nil {
		return err
	}

	if syncDone {
		log.Println("Initial sync already completed. Skipping...")
		return nil
	}

	log.Println("Initial sync not found. Fetching builds from Jenkins...")
	client := jenkins.NewJenkinsClient(
		os.Getenv("JENKINS_URL"),
		os.Getenv("JENKINS_USER"),
		os.Getenv("JENKINS_TOKEN"),
	)

	if client.BaseURL == "" || client.Username == "" || client.APIToken == "" {
		log.Fatalf("Jenkins client not properly configured. Check environment variables.")
	}

	saved, failed, _, err := jenkins.FetchAndStoreBuilds(database, client)
	if err != nil {
		log.Printf("Initial sync failed: %v", err)
		return err
	}

	if saved == 0 {
		log.Printf("Initial sync skipped marking as complete: no builds inserted.")
		return fmt.Errorf("zero builds inserted during initial sync")
	}

	log.Printf("Initial sync complete: saved=%d, failed=%d\n", saved, failed)

	if err := database.MarkInitialSyncDone(); err != nil {
		return fmt.Errorf("failed to mark initial sync done: %w", err)
	}

	return nil
}
