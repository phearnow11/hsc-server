// Create a ci-pipelines monitor returns "OK" response

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	datadog "github.com/DataDog/datadog-api-client-go/api/v1/datadog"
)

func main() {
	body := datadog.Monitor{
		Name:    datadog.PtrString("Example-Create_a_ci_pipelines_monitor_returns_OK_response"),
		Type:    datadog.MONITORTYPE_CI_PIPELINES_ALERT,
		Query:   `ci-pipelines("ci_level:pipeline @git.branch:staging* @ci.status:error").rollup("count").by("@git.branch,@ci.pipeline.name").last("5m") >= 1`,
		Message: datadog.PtrString("some message Notify: @hipchat-channel"),
		Tags: []string{
			"test:examplecreateacipipelinesmonitorreturnsokresponse",
			"env:ci",
		},
		Priority: *datadog.NewNullableInt64(datadog.PtrInt64(3)),
		Options: &datadog.MonitorOptions{
			Thresholds: &datadog.MonitorThresholds{
				Critical: datadog.PtrFloat64(1),
			},
		},
	}
	ctx := datadog.NewDefaultContext(context.Background())
	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)
	resp, r, err := apiClient.MonitorsApi.CreateMonitor(ctx, body)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `MonitorsApi.CreateMonitor`: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}

	responseContent, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Fprintf(os.Stdout, "Response from `MonitorsApi.CreateMonitor`:\n%s\n", responseContent)
}