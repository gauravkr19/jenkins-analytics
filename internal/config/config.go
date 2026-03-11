package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

type RetentionConfig struct {
    MaxRecords     int  // e.g. 280000
    DeleteMultiple int  // e.g. 50
    CleanupEnabled  bool
}
type EnvConfig struct {
	DBUser       string
	DBPass       string
	DBName       string
	DBHost       string
	DBPort       int
	SSLMode      string
	JenkinsURL   string
	JenkinsUser  string
	JenkinsToken string
	JenkinsTLS 	 bool
	DSN          string
}

func LoadEnvConfig() *EnvConfig {
	cfg := &EnvConfig{
		DBUser:       getOrExit("DB_USER"),
		DBPass:       getOrExit("DB_PASS"),
		DBName:       getOrExit("DB_NAME"),
		DBHost:       getOrExit("DB_HOST"),
		JenkinsURL:   getOrExit("JENKINS_URL"),
		JenkinsUser:  getOrExit("JENKINS_USER"),
		JenkinsToken: getOrExit("JENKINS_TOKEN"),
		SSLMode:      getOrDefault("SSL_MODE_DB", "disable"),
		JenkinsTLS:   os.Getenv("JENKINS_TLS_INSECURE") != "true",
		DBPort:       getIntOrDefault("DB_PORT", 5432),		
	}

	cfg.DSN = fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.SSLMode,
	)

	return cfg
}

// DataRetentionConfig reads env vars or falls back to defaults.
func DataRetentionConfig() RetentionConfig {
    maxRecs := getIntOrDefault("MAX_BUILD_RECORDS", 350000)
    multiple := getIntOrDefault("DELETE_MULTIPLE", 50)

    return RetentionConfig{
        MaxRecords:     maxRecs,
        DeleteMultiple: multiple,
        CleanupEnabled: true,
    }
}

func getOrExit(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Missing required environment variable: %s", key)
	}
	return val
}

func getOrDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func getIntOrDefault(envVar string, defaultVal int) int {
	valStr := os.Getenv(envVar)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

