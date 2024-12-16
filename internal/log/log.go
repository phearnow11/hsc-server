package ddlog

import (
	"context"
	"net/http"
	baserespone "pigdata/datadog/processingServer/pkg/baseRespone"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

var (
	configuration = datadog.NewConfiguration()
	apiClient     = datadog.NewAPIClient(configuration)
	api           = datadogV2.NewLogsApi(apiClient)
)

func SearchLog(ctx context.Context) ([]byte, *http.Response, error) {
	body := datadogV2.LogsListRequest{
		Filter: &datadogV2.LogsQueryFilter{
			Query: datadog.PtrString("datadog-agent"),
			Indexes: []string{
				"main",
			},
			From: datadog.PtrString("2024-08-27T11:48:36+01:00"),
			To:   datadog.PtrString("2024-08-28T12:48:36+01:00"),
		},
		Sort: datadogV2.LOGSSORT_TIMESTAMP_ASCENDING.Ptr(),
		Page: &datadogV2.LogsListRequestPage{
			Limit: datadog.PtrInt32(5),
		},
	}

	return baserespone.DatadogBaseRespone(api.ListLogs(ctx, *datadogV2.NewListLogsOptionalParameters().WithBody(body)))
}
