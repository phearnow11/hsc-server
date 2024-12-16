package metric

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	baserespone "pigdata/datadog/processingServer/pkg/baseRespone"
	"pigdata/datadog/processingServer/pkg/redis"
	"pigdata/datadog/processingServer/pkg/utils"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

var (
	configuration                                                                       = datadog.NewConfiguration()
	apiClient                                                                           = datadog.NewAPIClient(configuration)
	apiV1                                                                               = datadogV1.NewMetricsApi(apiClient)
	apiV2                                                                               = datadogV2.NewMetricsApi(apiClient)
	pointlistFakeData0, pointlistFakeData1                                      float64 = 0, 0.14769268345204828
	unitFakeDataFamily, unitFakeDataName, unitFakeDataPlural, unitFakeDataShort string  = "bytes", "megabyte", "megabytes", "Mb"
	FakeData                                                                            = datadogV1.MetricsQueryResponse{
		Series: []datadogV1.MetricsQueryMetadata{{
			Pointlist: [][]*float64{{&pointlistFakeData0, &pointlistFakeData1}},
			Unit:      []datadogV1.MetricsQueryUnit{{Family: &unitFakeDataFamily, Name: &unitFakeDataName, Plural: &unitFakeDataPlural, ShortName: &unitFakeDataShort}},
		}},
	}
	time_unit_family, time_unit_id, time_unit_name, time_unit_plural, time_unit_scale_factor, time_unit_short_name = "time", 11, "second", "seconds", 1.0, "s"
	Time_unit                                                                                                      = datadogV1.MetricsQueryUnit{
		Family:      &time_unit_family,
		Name:        &time_unit_name,
		Plural:      &time_unit_plural,
		ScaleFactor: &time_unit_scale_factor,
		ShortName:   &time_unit_short_name,
	}
)

func init() {
	configuration.SetUnstableOperationEnabled("v2.QueryTimeseriesData", true)
}

// old code
// =================================================//============================================================//===============================================//
func ListAllMetric(ctx context.Context, from int64, token string) ([]byte, *http.Response, error) {
	return baserespone.DatadogBaseRespone(apiV1.ListActiveMetrics(ctx, from, *datadogV1.NewListActiveMetricsOptionalParameters()))
}

func GetMetricMeta(ctx context.Context, metricName string) ([]byte, *http.Response, error) {
	val, err := redis.GetData(ctx, metricName)
	if err == nil {
		return val, nil, nil
	}
	data, response, err := baserespone.DatadogBaseRespone(apiV1.GetMetricMetadata(ctx, metricName))
	if err != nil {
		return nil, nil, err
	}

	err = redis.SetData(ctx, data, metricName)
	if err != nil {
		log.Println(err)
	}
	return data, response, nil
}

func TimeseriesFormulaQuery(ctx context.Context, from int64, to int64, query string) ([]byte, *http.Response, error) {
	body := datadogV2.TimeseriesFormulaQueryRequest{
		Data: datadogV2.TimeseriesFormulaRequest{
			Attributes: datadogV2.TimeseriesFormulaRequestAttributes{
				Formulas: []datadogV2.QueryFormula{
					{
						Formula: "a",
						Limit: &datadogV2.FormulaLimit{
							Count: datadog.PtrInt32(10),
							Order: datadogV2.QUERYSORTORDER_DESC.Ptr(),
						},
					},
				},
				From: from,
				// Interval: datadog.PtrInt64(5000),
				Queries: []datadogV2.TimeseriesQuery{
					{
						MetricsTimeseriesQuery: &datadogV2.MetricsTimeseriesQuery{
							DataSource: datadogV2.METRICSDATASOURCE_METRICS,
							Query:      query,
							Name:       datadog.PtrString("a"),
						},
					},
				},
				To: to,
			},
			Type: datadogV2.TIMESERIESFORMULAREQUESTTYPE_TIMESERIES_REQUEST,
		},
	}

	return baserespone.DatadogBaseRespone(apiV2.QueryTimeseriesData(ctx, body))
}

func TimeseriesPointQuery(ctx context.Context, from int64, to int64, query string, key string) ([]byte, *http.Response, error) {
	key = query + key
	if !redis.IsConnected() {
		return baserespone.DatadogBaseRespone(apiV1.QueryMetrics(ctx, from, to, query))
	}
	val, err := redis.GetData(ctx, key)
	if err == nil {
		return val, nil, nil
	}
	log.Println("Failed to get Redis data:", err)
	// data, response, err := baserespone.DatadogBaseRespone(apiV1.QueryMetrics(ctx, from, to, query))
	data, response, err := apiV1.QueryMetrics(ctx, from, to, query)
	if err != nil {
		log.Println("Datadog response failed:", err)
		return nil, response, err
	}
	if len(data.Series) != 1 {
		return baserespone.DatadogBaseRespone(data, nil, nil)
	}

	real_data := data.Series[0].GetPointlist()
	if len(data.Series[0].GetUnit()) < 1 {
		return baserespone.DatadogBaseRespone(data, nil, nil)
	}
	unit := data.Series[0].GetUnit()[0]

	convertion_unit := 0
	base_unit_name := unit.GetName()
	base_unit_index := -1
	unit_range := []string{}

	if unit.GetFamily() == "bytes" {

		convertion_unit = 1024
		unit_range = []string{"byte", "kilobyte", "megabyte", "gigabyte", "terabyte", ""}
	} else if unit.GetFamily() == "time" {

		convertion_unit = 60
		if base_unit_name == "millisecond" {
			convertion_unit = 1000
		}
		unit_range = []string{"millisecond", "second", "minute", "hour", ""}

	} else {
		data, _, _ := baserespone.DatadogBaseRespone(data, response, nil)
		err = redis.SetData(ctx, data, key)
		if err != nil {
			log.Println("Failed to set Redis data:", err)
		}
		return data, nil, nil

	}

	base_unit_index = utils.Contain(unit_range, base_unit_name)

	sum, nums := 0.0, 0
	for i := 0; i < len(real_data); i++ {
		sum += *real_data[i][1]
		nums++
	}

	if sum/float64(nums) > 100 {
		log.Println("Condition met, ready to convert data")
		final_data, unit_name, unit_short := utils.Data_convert_up(unit_range, float64(convertion_unit), sum/float64(nums), base_unit_index)
		ts := 0.0
		data.GetSeries()[0].SetPointlist([][]*float64{{&ts, &final_data}})
		data.GetSeries()[0].GetUnit()[0].SetName(unit_name)
		data.GetSeries()[0].GetUnit()[0].SetPlural(unit_name + "s")
		data.GetSeries()[0].GetUnit()[0].SetShortName(unit_short)
	}

	resdata, _, _ := baserespone.DatadogBaseRespone(data, response, nil)
	err = redis.SetData(ctx, resdata, key)
	if err != nil {
		log.Println("Failed to set Redis data:", err)
	}
	return resdata, nil, nil
	// return data, response, nil

	// return baserespone.DatadogBaseRespone(apiV1.QueryMetrics(ctx, from, to, query))
}

func processDataWithConvert(data datadogV1.MetricsQueryResponse) (datadogV1.MetricsQueryResponse, error) {
	if len(data.Series) != 1 {
		return data, fmt.Errorf("Data Series is null, return with no convertion")
	}

	real_data := data.Series[0].GetPointlist()
	if len(data.Series[0].GetUnit()) < 1 {
		return data, fmt.Errorf("Data Unit is null, return with no convertion")
	}
	unit := data.Series[0].GetUnit()[0]

	convertion_unit := 0
	base_unit_name := unit.GetName()
	base_unit_index := -1
	unit_range := []string{}

	if unit.GetFamily() == "bytes" {

		convertion_unit = 1024
		unit_range = []string{"byte", "kilobyte", "megabyte", "gigabyte", "terabyte", ""}
	} else if unit.GetFamily() == "time" {

		convertion_unit = 60
		if base_unit_name == "millisecond" {
			convertion_unit = 1000
		}
		unit_range = []string{"millisecond", "second", "minute", "hour", ""}

	} else {
		return data, fmt.Errorf("Unit is not convertable, return with no convertion unit:", unit.GetName())
	}

	base_unit_index = utils.Contain(unit_range, base_unit_name)

	sum, nums := 0.0, 0
	for i := 0; i < len(real_data); i++ {
		sum += *real_data[i][1]
		nums++
	}

	base_data := sum / float64(nums)

	if base_unit_name == "nanosecond" {
		nn_data, unit_name, unit_short := utils.Data_convert_up([]string{"nanosecond", "millisecond", "second", "minute", "hour", ""}, float64(1000000), base_data, 0)
		base_data = nn_data
		base_unit_index = utils.Contain(unit_range, unit_name)
		ts := 0.0
		data.GetSeries()[0].SetPointlist([][]*float64{{&ts, &base_data}})
		data.GetSeries()[0].GetUnit()[0].SetName(unit_name)
		data.GetSeries()[0].GetUnit()[0].SetPlural(unit_name + "s")
		data.GetSeries()[0].GetUnit()[0].SetShortName(unit_short + "s")
	}

	if base_unit_index == -1 {
		log.Println("Unit not found -", data.GetQuery(), "- exit without convertion.")
		return data, nil
	}
	if base_data < 1 && base_unit_index >= 0 {
		log.Println("Condition met, ready to convert data")
		base_unit_index = utils.Contain([]string{"microsecond", "nanosecond", "millisecond", "second", "minute", "hour", ""}, base_unit_name)
		final_data, unit_name, unit_short := utils.Data_convert_down([]string{"microsecond", "nanosecond", "millisecond", "second", "minute", "hour", ""}, float64(convertion_unit), base_data, base_unit_index)
		ts := 0.0
		data.GetSeries()[0].SetPointlist([][]*float64{{&ts, &final_data}})
		data.GetSeries()[0].GetUnit()[0].SetName(unit_name)
		data.GetSeries()[0].GetUnit()[0].SetPlural(unit_name + "s")
		if unit_name == "millisecond" {
			unit_short += "s"
		}
		if unit_name == "nanosecond" {
			unit_short += "s"
		}
		if unit_name == "microsecond" {
			unit_short = "µs"
		}
		// Unknown dup empty unit bug risk: none
		data.GetSeries()[0].GetUnit()[0].SetShortName(unit_short)
		data.GetSeries()[0].SetUnit([]datadogV1.MetricsQueryUnit{data.GetSeries()[0].GetUnit()[0]})
	}
	if base_data > 100 && base_unit_index >= 0 {
		log.Println("Condition met, ready to convert data")
		final_data, unit_name, unit_short := utils.Data_convert_up(unit_range, float64(convertion_unit), base_data, base_unit_index)
		ts := 0.0
		data.GetSeries()[0].SetPointlist([][]*float64{{&ts, &final_data}})
		data.GetSeries()[0].GetUnit()[0].SetName(unit_name)
		data.GetSeries()[0].GetUnit()[0].SetPlural(unit_name + "s")
		data.GetSeries()[0].GetUnit()[0].SetShortName(unit_short)
	}

	return data, nil
}

func processDataRange(data datadogV1.MetricsQueryResponse, from int64, to int64) datadogV1.MetricsQueryResponse {
	fromf, tof := float64(from)*1000, float64(to)*1000
	if len(data.GetSeries()) < 1 {
		return data
	}
	if len(data.GetSeries()[0].GetPointlist()) < 1 {
		return data
	}
	processed_pointlist := [][]*float64{}

	for _, pointlist := range data.GetSeries()[0].GetPointlist() {
		ts := pointlist[0]
		if *ts >= fromf && *ts <= tof {
			processed_pointlist = append(processed_pointlist, []*float64{ts, pointlist[1]})
		}

	}
	data.GetSeries()[0].SetPointlist(processed_pointlist)
	return data
}

func GetTimeSeriesPointInRange(ctx context.Context, key string, from int64, to int64, query string, isRedis bool) (datadogV1.MetricsQueryResponse, error) {
	var dataFD datadogV1.MetricsQueryResponse
	if !isRedis {
		log.Println("Redis failed, attempt to get data without redis")
		dataFD, _, error := apiV1.QueryMetrics(ctx, from, to, query)
		if error != nil {
			return datadogV1.MetricsQueryResponse{}, error
		}
		resData, err := processDataWithConvert(processDataRange(dataFD, from, to))
		if err != nil {
			log.Println("Error converting data:", err)
		}
		return resData, nil
	}
	data, err := redis.CheckRangeGetData(ctx, key)
	json.Unmarshal(data, &dataFD)
	if err == nil {
		resData, err := processDataWithConvert(processDataRange(dataFD, from, to))
		if err != nil {
			log.Println("Error converting data:", err)
		}
		return resData, nil
	}
	log.Println("Get data from redis failed:", err, ", ready to get new data")
	ndata, res, err := apiV1.QueryMetrics(ctx, from, to, query)
	if err != nil {
		log.Println("Data dog failed with response:", res)
		return datadogV1.MetricsQueryResponse{}, fmt.Errorf("Data dog failed: %v", err)
	}
	stdata, _, _ := baserespone.DatadogBaseRespone(ndata, nil, nil)
	err = redis.CheckRangeSetData(ctx, key, stdata)
	if err != nil {
		log.Println("Set data to redis failed:", err)
	}
	converted_data, err := processDataWithConvert(processDataRange(ndata, from, to))
	if err != nil {
		log.Println("Error converting data:", err)
	}
	return converted_data, nil
}

func Empty_unit_bug_temp_solved(metricData datadogV1.MetricsQueryResponse) datadogV1.MetricsQueryResponse {
	series_updated := []datadogV1.MetricsQueryMetadata{}
	for _, series := range metricData.GetSeries() {

		units := []datadogV1.MetricsQueryUnit{}
		for _, unit := range series.GetUnit() {
			if unit.Name != nil {
				units = append(units, unit)
			}
		}
		series.SetUnit(units)
		series_updated = append(series_updated, series)
	}
	metricData.SetSeries(series_updated)
	return metricData
}

// =================================================//============================================================//===============================================//

// new code
func TimeseriesPointQueryData(from int64, to int64, query string) datadogV1.MetricsQueryResponse {
	ctx := datadog.NewDefaultContext(context.Background())
	data, _, err := apiV1.QueryMetrics(ctx, from, to, query)
	if err != nil {
		log.Println("Metric: Error call Datadog API:", err)
		return datadogV1.MetricsQueryResponse{}
	}
	return data
}
