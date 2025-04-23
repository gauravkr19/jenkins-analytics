package jenkins

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	Number      int    `json:"number"`
	Result      string `json:"result"`
	Duration    int64  `json:"duration"`
	Timestamp   int64  `json:"timestamp"`
	URL         string `json:"url"`
	BuiltOn     string `json:"builtOn"`
	ProjectName string `json:"project_name"`
}

func NewJenkinsClient(baseURL, username, token string) *JenkinsClient {
	return &JenkinsClient{
		BaseURL:  baseURL,
		Username: username,
		APIToken: token,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (jc *JenkinsClient) FetchBuilds() ([]Build, error) {
	apiURL := fmt.Sprintf("%s/api/json?tree=jobs[name,url,builds[number,result,duration,timestamp,url,builtOn]]", jc.BaseURL)

	resp, err := jc.Client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Jenkins: %w", err)
	}
	defer resp.Body.Close()

	var jResp JenkinsResponse
	if err := json.NewDecoder(resp.Body).Decode(&jResp); err != nil {
		return nil, fmt.Errorf("failed to decode Jenkins response: %w", err)
	}

	var builds []Build
	for _, job := range jResp.Jobs {
		for _, b := range job.Builds {
			builds = append(builds, b)
		}
	}
	return builds, nil
}
