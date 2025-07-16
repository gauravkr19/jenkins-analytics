package poller

import (
<<<<<<< Updated upstream
    "log"
    "time"

    "github.com/gauravkr19/jenkins-analytics/internal/db"
    "github.com/gauravkr19/jenkins-analytics/internal/jenkins"
=======
	"log"
	"time"

	"github.com/gauravkr19/jenkins-analytics/internal/db"
	"github.com/gauravkr19/jenkins-analytics/internal/jenkins"
>>>>>>> Stashed changes
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
<<<<<<< Updated upstream
=======

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
>>>>>>> Stashed changes
