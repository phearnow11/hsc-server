package event

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
	eventApi      = datadogV2.NewEventsApi(apiClient)
)

func ListEvents(ctx context.Context, from string, to string, query string, sort string, pageCursor string, pageLimit int32) ([]byte, *http.Response, error) {
	return baserespone.DatadogBaseRespone(eventApi.ListEvents(ctx, *datadogV2.NewListEventsOptionalParameters().WithFilterQuery(query).WithFilterFrom(from).WithFilterTo(to)))
}

func SearchEvents(ctx context.Context, from string, to string, query string) ([]byte, *http.Response, error) {
	body := datadogV2.EventsListRequest{
		Filter: &datadogV2.EventsQueryFilter{
			Query: datadog.PtrString(query),
			From:  datadog.PtrString(from),
			To:    datadog.PtrString(to),
		},
	}
	return baserespone.DatadogBaseRespone(eventApi.SearchEvents(ctx, *datadogV2.NewSearchEventsOptionalParameters().WithBody(body)))
}
