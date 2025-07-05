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
}

type Action struct {
	Causes       []Cause      `json:"causes,omitempty"`
	Parameters   []Param      `json:"parameters,omitempty"`
	RemoteURLs   []string     `json:"remoteUrls,omitempty"`
	LastRevision *GitRevision `json:"lastBuiltRevision,omitempty"`
}

type Param struct {
	Name  string `json:"name"`
	Value string `json:"value"`
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
				log.Printf("Error fetching nested folder: %s", job.URL)
				continue
			}
			builds = append(builds, childBuilds...)
		} else {
			for _, b := range job.Builds {
				b.ProjectName = job.Name
				builds = append(builds, b)
			}
		}
	}

	return builds, nil
}

// puller.go
func FetchAndStoreBuilds(db *db.DB, client *JenkinsClient) (int, int, []int, error) {
	builds, err := client.FetchBuilds()
	if err != nil {
		return 0, 0, nil, fmt.Errorf("fetch builds failed: %w", err)
	}

	saved, failed := 0, 0
	var failedBuilds []int

	for _, b := range builds {
		userID := extractUserID(b.Actions)
		if userID == "unknown@jenkins" {
			log.Printf("User ID not found for build #%d", b.Number)
		}

		gitURL, branch, sha := extractGitInfo(b.Actions)
		params := extractParameters(b.Actions)

		dbModel := &models.Build{
			BuildNumber: b.Number,
			ProjectName: b.ProjectName,
			ProjectPath: extractProjectPathFromURL(b.URL, os.Getenv("JENKINS_URL"), b.Number),
			UserID:      userID,
			Status:      b.Result,
			Timestamp:   time.UnixMilli(b.Timestamp),
			DurationMS:  b.Duration,
			JobURL:      b.URL,
			GitRepo:     gitURL,
			Branch:      branch,
			CommitSHA:   sha,
			DeployEnv:   params["DEPLOY_ENV"],
			TriggerType: extractTriggerType(b.Actions),
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
