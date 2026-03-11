package jenkins

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gauravkr19/jenkins-analytics/internal/db"
	"github.com/gauravkr19/jenkins-analytics/models"
)

type JenkinsClient struct {
	BaseURL  string
	Username string
	APIToken string
	Client   *http.Client
}

type JenkinsResponse struct {
	Jobs []Job `json:"jobs"`
}

type Job struct {
	Name   string  `json:"name"`
	URL    string  `json:"url"`
	Builds []Build `json:"builds"`
}

type Build struct {
	Number      int      `json:"number"`
	Result      string   `json:"result"`
	Duration    int64    `json:"duration"`
	Timestamp   int64    `json:"timestamp"`
	URL         string   `json:"url"`
	ProjectName string   `json:"project_name"`
	Branch      string   `json:"branch"`
	GitRepo     string   `json:"giturl"`
	CommitSHA   string   `json:"gitcommit"`
	Actions     []Action `json:"actions,omitempty"`
	Env         string   `json:"env,omitempty"`  
}

type Action struct {
	Causes       []Cause      `json:"causes,omitempty"`
	Parameters   []Param      `json:"parameters,omitempty"`
	RemoteURLs   []string     `json:"remoteUrls,omitempty"`
	LastRevision *GitRevision `json:"lastBuiltRevision,omitempty"`
}

type Param struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"` // can accept bool, int, string, etc.
}

type GitRevision struct {
	SHA1   string      `json:"SHA1"`
	Branch []GitBranch `json:"branch"`
}

type GitBranch struct {
	Name string `json:"name"`
}
type Cause struct {
	UserID           string `json:"userId"`
	UserName         string `json:"userName"`
	ShortDescription string `json:"shortDescription"` // TriggerType
}

func NewJenkinsClient(baseURL, username, token string) *JenkinsClient {
	timeout := 10 * time.Second

	transport := http.DefaultTransport

	// TLS config
	caCertPath := os.Getenv("JENKINS_CACERT")
	insecure := os.Getenv("JENKINS_TLS_INSECURE") == "true"

	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err == nil {
			caCertPool := x509.NewCertPool()
			if caCertPool.AppendCertsFromPEM(caCert) {
				transport = &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs: caCertPool,
					},
				}
			}
		}
	} else if insecure {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return &JenkinsClient{
		BaseURL:  baseURL,
		Username: username,
		APIToken: token,
		Client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

// call seq: external -> FetchAndStoreBuilds -> FetchBuilds -> fetchBuildsRecursive
func (jc *JenkinsClient) FetchBuilds() ([]Build, error) {
	return jc.fetchBuildsRecursive(jc.BaseURL)
}

func (jc *JenkinsClient) fetchBuildsRecursive(folderURL string) ([]Build, error) {

	// apiURL := fmt.Sprintf("%s/api/json?tree=jobs[name,url,_class,builds[number,result,duration,timestamp,url,builtOn,actions[causes[userId,userName],parameters[name,value],lastBuiltRevision[branch[name],SHA1],remoteUrls],changeSet[items[msg,author[fullName],commitId],kind]]]", strings.TrimSuffix(folderURL, "/"))
	apiURL := fmt.Sprintf("%s/api/json?tree=jobs[name,url,_class,builds[number,result,duration,timestamp,url,builtOn,actions[causes[userId,userName,shortDescription],parameters[name,value],lastBuiltRevision[branch[name],SHA1],remoteUrls],changeSet[items[msg,author[fullName],commitId],kind]]]", strings.TrimSuffix(folderURL, "/"))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", apiURL, err)
	}
	req.SetBasicAuth(jc.Username, jc.APIToken)

	resp, err := jc.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s: %w", apiURL, err)
	}
	defer resp.Body.Close()

	var data struct {
		Jobs []struct {
			Class  string  `json:"_class"`
			Name   string  `json:"name"`
			URL    string  `json:"url"`
			Builds []Build `json:"builds"`
		} `json:"jobs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error decoding Jenkins response from %s: %w\nResponse: %s", apiURL, err, string(body))
	}

	var builds []Build
	for _, job := range data.Jobs {

		
		if strings.Contains(job.Class, "Folder") {
			childBuilds, err := jc.fetchBuildsRecursive(job.URL)
			if err != nil {
				log.Printf("Error fetching nested folder: %s", job.URL, err)
				continue
			}
			builds = append(builds, childBuilds...)
		} else {
			for _, b := range job.Builds {
				b.ProjectName = job.Name

				// Common parameter keys people use; add your exact one here
				// Exact matches (fast + preferred)
				if env, ok := b.GetParamString(
					"ENV", "Environment", "environment",
					"TARGET_ENV", "DEPLOY_ENV", "DEPLOYING_ENVIRONMENT",
				); ok {
					b.Env = normalizeEnv(env)
				} else {
					// 2) Fallback heuristic: scan all params and guess
					if env, ok := b.GuessEnvFromParams(); ok {
						b.Env = normalizeEnv(env)
					}
				}
				builds = append(builds, b)
			}
		}
	}

	return builds, nil
}

func normalizeEnv(s string) string {
    return strings.ToLower(strings.TrimSpace(s))
}
// GuessEnvFromParams looks for env-ish param names beyond the allowlist.
func (b *Build) GuessEnvFromParams() (string, bool) {
    bestScore := -1
    bestVal := ""

    for _, a := range b.Actions {
        for _, p := range a.Parameters {
            name := strings.ToLower(strings.TrimSpace(p.Name))
            if name == "" {
                continue
            }

            // Skip obvious non-env params (avoid false positives)
            if strings.Contains(name, "git") ||
                strings.Contains(name, "branch") ||
                strings.Contains(name, "commit") ||
                strings.Contains(name, "sha") ||
                strings.Contains(name, "version") ||
                strings.Contains(name, "tag") ||
                strings.Contains(name, "repo") {
                continue
            }

            // Score likely env keys
            score := 0
            switch {
            case strings.Contains(name, "deploying_environment"):
                score = 60
            case strings.Contains(name, "deploy_env"):
                score = 55
            case strings.Contains(name, "target_env"):
                score = 50
            case strings.Contains(name, "environment"):
                score = 45
            case name == "env" || strings.HasSuffix(name, "_env"):
                score = 40
            case strings.Contains(name, "env"):
                score = 30
            case name == "stage" || strings.Contains(name, "stage"):
                score = 25
            }

            if score == 0 {
                continue
            }

            // Convert value to string
            val := fmt.Sprintf("%v", p.Value)
            if strings.TrimSpace(val) == "" {
                continue
            }

            if score > bestScore {
                bestScore = score
                bestVal = val
            }
        }
    }

    if bestScore >= 0 {
        return bestVal, true
    }
    return "", false
}

// Fetches build data from Jenkins and writes to DB
func FetchAndStoreBuilds(db *db.DB, client *JenkinsClient, incremental bool) (int, int, []int, error) {
	builds, err := client.FetchBuilds()
	if err != nil {
		return 0, 0, nil, fmt.Errorf("fetch builds failed: %w", err)
	}

	saved, failed := 0, 0
	var failedBuilds []int

	for _, b := range builds {
		if incremental {
			lastSeen, err := db.GetLastSeenBuildNumber(b.ProjectName)
			if err != nil {
				log.Printf("Error getting last seen for project %s: %v", b.ProjectName, err)
				continue
			}
			if b.Number <= lastSeen {
				continue // Already stored
			}
		}

		userID := extractUserID(b.Actions)

		gitURL, branch, sha := extractGitInfo(b.Actions)
		params := extractParameters(b.Actions)
		projectPath := extractProjectPathFromURL(b.URL, os.Getenv("JENKINS_URL"), b.Number)

		deployEnv := b.Env 			// if env is "", then use params to capture it
		if strings.TrimSpace(deployEnv) == "" {
			deployEnv = extractAny(params, "DEPLOY_ENV", "DEPLOYING_ENVIRONMENT", "ENV", "TARGET_ENV")
		}
		deployEnv = strings.ToLower(strings.TrimSpace(deployEnv))
		igrmNo := strings.TrimSpace(params["IGRM_NO"])		

		dbModel := &models.Build{
			BuildNumber: b.Number,
			ProjectName: b.ProjectName,
			ProjectPath: projectPath,
			UserID:      userID,
			Status:      b.Result,
			Timestamp:   time.UnixMilli(b.Timestamp),
			DurationMS:  b.Duration,
			JobURL:      b.URL,
			GitRepo:     gitURL,
			Branch:      branch,
			CommitSHA:   sha,						
			DeployEnv:   deployEnv,						// parameter env 
			TriggerType: extractTriggerType(b.Actions), // ShortDescription - Started by user
			Env: 		 extractEnv(projectPath),		// env from folder path
			IGRMNo:    	 igrmNo,
		}

		if err := db.InsertBuild(dbModel); err != nil {
			log.Printf("Insert failed for build #%d: %v", b.Number, err)
			failedBuilds = append(failedBuilds, b.Number)
			failed++
			continue
		}
		saved++
	}

	return saved, failed, failedBuilds, nil
}

func extractAny(params map[string]string, keys ...string) string {
    for _, k := range keys {
        if v, ok := params[k]; ok && strings.TrimSpace(v) != "" {
            return v
        }
    }
    return ""
}

// Fetches paramaterized values like ENV
func (b *Build) GetParamString(paramNames ...string) (string, bool) {
    // Normalize names for matching
    want := make(map[string]struct{}, len(paramNames))
    for _, n := range paramNames {
        want[strings.ToLower(strings.TrimSpace(n))] = struct{}{}
    }

    for _, a := range b.Actions {
        for _, p := range a.Parameters {
            if _, ok := want[strings.ToLower(strings.TrimSpace(p.Name))]; !ok {
                continue
            }
            switch v := p.Value.(type) {
            case string:
                if v != "" {
                    return v, true
                }
            case fmt.Stringer:
                s := v.String()
                if s != "" {
                    return s, true
                }
            case float64: // Jenkins numbers decode as float64 into interface{}
                return strconv.FormatFloat(v, 'f', -1, 64), true
            case bool:
                if v {
                    return "true", true
                }
                return "false", true
            default:
                // last resort: stringify JSON-ish values
                by, err := json.Marshal(v)
                if err == nil && len(by) > 0 {
                    return string(by), true
                }
            }
        }
    }
    return "", false
}

// extractEnv derives env from project path
func extractEnv(path string) string {
    parts := strings.Split(path, "/")
    if len(parts) == 0 {
        return "UNKNOWN"
    }

    switch strings.ToUpper(parts[0]) {
    case "DEV":
        return "DEV"
    case "NONPROD", "NON_PROD":
        return "NON_PROD"
    case "PROD_AND_DR", "PROD-DR":
        return "PROD_AND_DR"
    }
    return "UNKNOWN"
}

func extractUserID(actions []Action) string {
	for _, action := range actions {
		for _, cause := range action.Causes {
			if cause.UserID != "" {
				return cause.UserID
			}
			if cause.UserName != "" {
				return cause.UserName + "@jenkins"
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

func extractGitInfo(actions []Action) (url, branch, sha string) {
	for _, action := range actions {
		if len(action.RemoteURLs) > 0 && url == "" {
			url = action.RemoteURLs[0]
		}
		if action.LastRevision != nil {
			if sha == "" {
				sha = action.LastRevision.SHA1
			}
			if len(action.LastRevision.Branch) > 0 && branch == "" {
				branch = strings.TrimPrefix(action.LastRevision.Branch[0].Name, "origin/")
			}
		}
	}
	return
}

func extractTriggerType(actions []Action) string {
	for _, action := range actions {
		for _, cause := range action.Causes {
			if cause.ShortDescription != "" {
				return cause.ShortDescription
			}
		}
	}
	return "unknown"
}

func extractParameters(actions []Action) map[string]string {
	params := make(map[string]string)
	for _, action := range actions {
		for _, p := range action.Parameters {
			params[p.Name] = fmt.Sprintf("%v", p.Value)
		}
	}
	return params
}

func PatchMissingStatuses(db *db.DB, client *JenkinsClient, patchLimit int) error {
    builds, err := db.GetRecentBuildsMissingStatus(patchLimit)
    if err != nil {
        return err
    }

    for _, b := range builds {
        // Construct the full Jenkins API URL from the JobURL
        buildURL := strings.TrimSuffix(b.JobURL, "/") + "/api/json"

        build, err := client.FetchBuildByURL(buildURL)
		if err != nil {
			// Only log if the build is from today
			if time.Since(b.Timestamp).Hours() < 24 {
				log.Printf("Failed to fetch build by URL %s (likely deleted or inaccessible): %v", buildURL, err)
			}
			continue
		}

        if build.Result == "" {
            log.Printf("Build at %s is still running or has no result, skipping", b.JobURL)
            continue
        }

        err = db.UpdateBuildStatus(b.ID, build.Result)
        if err != nil {
            log.Printf("Failed to patch status for build ID %d: %v", b.ID, err)
        } 
    }

    return nil
}

// to patch missing status
func (jc *JenkinsClient) FetchBuildByURL(apiURL string) (*Build, error) {
    req, err := http.NewRequest("GET", apiURL, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request for %s: %w", apiURL, err)
    }

    if jc.Username != "" && jc.APIToken != "" {
        req.SetBasicAuth(jc.Username, jc.APIToken)
    }

    resp, err := jc.Client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error fetching %s: %w", apiURL, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Jenkins returned status %d for %s", resp.StatusCode, apiURL)
    }

    var b Build
    if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
        return nil, err
    }

    return &b, nil
}
