package poller

import (
	"log"
	"time"

	"github.com/gauravkr19/jenkins-analytics/internal/config"
	"github.com/gauravkr19/jenkins-analytics/internal/db"
	"github.com/gauravkr19/jenkins-analytics/internal/jenkins"
)

func StartIncrementalPoller(database *db.DB, client *jenkins.JenkinsClient, interval time.Duration) {
    go func() {
        for {
            log.Println("[Poller] Checking for new Jenkins builds...")
            saved, failed, failedIDs, err := jenkins.FetchAndStoreBuilds(database, client, true)
            if err != nil {
                log.Printf("[Poller] Error fetching builds: %v", err)
            } else {
                log.Printf("[Poller] Poll result: saved=%d, failed=%d, failedIDs=%v", saved, failed, failedIDs)
            }

            time.Sleep(interval)
        }
    }()
}

func StartStatusPatcher(database *db.DB, client *jenkins.JenkinsClient, patchInterval time.Duration, patchLimit int) {
    go func() {
        for {
            log.Println("[Patcher] Scanning for builds with missing status...")
            err := jenkins.PatchMissingStatuses(database, client, patchLimit)
            if err != nil {
                log.Printf("[Patcher] Error patching statuses: %v", err)
            }

            time.Sleep(patchInterval)
        }
    }()
}

func DeletionRoutine(database *db.DB, client *jenkins.JenkinsClient, interval time.Duration, cfg config.RetentionConfig) {
    go func() {
        for {
            if !cfg.CleanupEnabled {
                time.Sleep(interval)
                continue
            }

            // 1) Check current record count to determine cleanup
            var total int
            total, err := database.CountBuilds()
            if err != nil {
                log.Printf("[Retention] count check error: %v", err)
                time.Sleep(interval)
                continue
            }

            if total > cfg.MaxRecords {
                log.Printf("[Retention] total records=%d, exceeding limit=%d. Performing cleanup.", total, cfg.MaxRecords)
                if err := database.CleanupToMax(cfg.MaxRecords, cfg.DeleteMultiple); err != nil {
                    log.Printf("[Retention] cleanup error: %v", err)
                }
            } else {
                log.Printf("[Retention] total records=%d, under limit=%d. Skipping cleanup.", total, cfg.MaxRecords)
            }

            time.Sleep(interval)
        }
    }()
}
