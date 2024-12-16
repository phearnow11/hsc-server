package span

import (
	"context"
	"net/http"
	baserespone "pigdata/datadog/processingServer/pkg/baseRespone"
	"pigdata/datadog/processingServer/pkg/redis"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

var (
	apiSpanMetric *datadogV2.SpansMetricsApi
	apiSpan       *datadogV2.SpansApi
)

func init() {
	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)
	apiSpanMetric = datadogV2.NewSpansMetricsApi(apiClient)

	apiSpan = datadogV2.NewSpansApi(apiClient)
}

func GetAllSpanMetric(ctx context.Context, key string) ([]byte, *http.Response, error) {
	// return baserespone.DatadogBaseRespone(apiSpanMetric.ListSpansMetrics(ctx))
	val, err := redis.GetData(ctx, key)
	if err == nil {
		return val, nil, nil
	}

	data, response, err := baserespone.DatadogBaseRespone(apiSpanMetric.ListSpansMetrics(ctx))
	if err != nil {
		return nil, nil, err
	}

	err = redis.SetData(ctx, data, key)
	if err != nil {
		return nil, nil, err
	}

	return data, response, err
}

func GetASpanMetric(ctx context.Context, SpansMetricDataID string) ([]byte, *http.Response, error) {
	val, err := redis.GetData(ctx, SpansMetricDataID)
	if err == nil {
		return val, nil, nil
	}
	data, response, err := baserespone.DatadogBaseRespone(apiSpanMetric.GetSpansMetric(ctx, SpansMetricDataID))
	if err != nil {
		return nil, nil, err
	}

	err = redis.SetData(ctx, data, SpansMetricDataID)
	if err != nil {
		return nil, nil, err
	}
	return data, response, err
}

func GetAListSpans(ctx context.Context) ([]byte, *http.Response, error) {
	return baserespone.DatadogBaseRespone(apiSpan.ListSpansGet(ctx, *datadogV2.NewListSpansGetOptionalParameters()))
}
