package jenkins

import (
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestFetchBuilds(t *testing.T) {
	client := NewJenkinsClient("http://jenkins.local")
	httpmock.ActivateNonDefault(client.Client)
	defer httpmock.DeactivateAndReset()

	mockResponse := `{
		"jobs": [
			{
				"name": "test-job",
				"url": "http://jenkins.local/job/test-job/",
				"builds": [
					{
						"number": 42,
						"result": "SUCCESS",
						"duration": 12345,
						"timestamp": 1680000000000,
						"url": "http://jenkins.local/job/test-job/42/",
						"builtOn": "agent-1"
					}
				]
			}
		]
	}`

	httpmock.RegisterResponder("GET",
		"http://jenkins.local/api/json?tree=jobs[name,url,builds[number,result,duration,timestamp,url,builtOn]]",
		httpmock.NewStringResponder(200, mockResponse),
	)

	builds, err := client.FetchBuilds()
	if err != nil {
		t.Fatalf("FetchBuilds failed: %v", err)
	}

	if len(builds) != 1 || builds[0].Number != 42 {
		t.Errorf("unexpected builds: %+v", builds)
	}
}
