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
	Actions     []Action `json:"actions,omitempty"`
}
type Action struct {
	Causes []Cause `json:"causes,omitempty"`
}
type Cause struct {
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
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
	apiURL := fmt.Sprintf("%s/api/json?tree=jobs[name,url,_class,builds[number,result,duration,timestamp,url,builtOn]]", strings.TrimSuffix(folderURL, "/"))

	req, _ := http.NewRequest("GET", apiURL, nil)
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
