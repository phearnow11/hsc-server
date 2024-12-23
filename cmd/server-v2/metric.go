package main

import (
	"log"
	"pigdata/datadog/processingServer/internal/metric"
	"pigdata/datadog/processingServer/pkg/mongodb"
	"pigdata/datadog/processingServer/pkg/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
)

func populateMetricData(queryMap map[string]string, timestampBody timstamp) map[string]interface{} {
	metricData := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup

	for metricName, query := range queryMap {
		if metricName == "name" {
			continue
		}

		wg.Add(1)
		go func(metricName, query string) {
			defer wg.Done()
			var result interface{}
			if metricName == "Avail" {
				result = status_data_new[queryMap["name"]]
			} else {
				result = GetMetricData(query, timestampBody.From, timestampBody.To)
			}
			// Lock for multi-thread write mu.Lock()
			mu.Lock()
			metricData[metricName] = result
			mu.Unlock()
		}(metricName, query)
	}

	wg.Wait()
	return metricData
}

func GetStatusData(name string, query string, from int64, to int64) interface{} {
	status_data_msg := "RESPONSE: GET STATUS DATA:"
	if !(query == "test" || query == "") {
		queries := strings.Split(query, "&&&")
		cur_status_data := []interface{}{}
		failed_tagset[name] = make(map[string]interface{})
		for i, query := range queries {
			logic_check := strings.Split(query, "by")
			logic_type := []string{"test_name", "kube_deployment", "service"}
			data := metric.TimeseriesPointQueryData(from, to, query)

			for _, substring := range logic_type {
				if strings.Contains(logic_check[1], substring) {
					switch substring {
					case "test_name":
						cur_status_data = append(cur_status_data, process_status_for_test_name(name, data))
					case "kube_deployment":
						cur_status_data = append(cur_status_data, process_status_for_kube(name, data))
					case "service":
						cur_status_data = append(cur_status_data, process_status_for_service(name, data))
					}
					continue
				}
			}
			log.Println(status_data_msg, "Query:", query, ":", cur_status_data[i], "Dashboard:", name)
		}
		is_nodata := false
		for _, status := range cur_status_data {
			if status == "FAILED" {
				return status
			}
			if status == "No Data" {
				is_nodata = true
			}
		}
		if is_nodata {
			return "No Data"
		}
		return "OK"
	}
	log.Println(status_data_msg, "No data found: Error at mapping configuration")

	return "No Data"
}

func process_status_for_service(name string, data datadogV1.MetricsQueryResponse) interface{} {
	if len(data.GetSeries()) == 0 {
		return "No Data"
	}
	res_status := "OK"

	for _, series := range data.GetSeries() {
		if len(series.GetPointlist()) > 0 {
			if len(series.GetTagSet()) == 0 {
				log.Println("SERVICE METRIC STATUS: No Tagset Found, skipped")
				continue
			}
			cur_tagset_status := DetailData{}
			// last pointlist data
			cur_data := *series.GetPointlist()[len(series.GetPointlist())-1][1]
			if cur_data >= 10 {
				status_data[name] = false
				log.Println("SERVICE METRIC STATUS:", name, false, series.GetTagSet())
				cur_tagset_status = DetailData{Status: "FAILED", Test_name: strings.Split(series.GetTagSet()[0], ":")[1], Success_rate: 0}
				res_status = "FAILED"
			} else {
				cur_tagset_status = DetailData{Status: "OK", Test_name: strings.Split(series.GetTagSet()[0], ":")[1], Success_rate: 0}
			}

			failed_tagset[name][strings.Split(series.GetTagSet()[0], ":")[1]] = cur_tagset_status
		}
	}

	status_data[name] = true
	return res_status
}

func process_status_for_kube(name string, data datadogV1.MetricsQueryResponse) interface{} {
	if len(data.GetSeries()) == 0 {
		return "No Data"
	}
	res_status := "OK"

	for _, series := range data.GetSeries() {
		if len(series.GetPointlist()) > 0 {
			if len(series.GetTagSet()) == 0 {
				log.Println("SERVICE METRIC STATUS: No Tagset Found, skipped")
				continue
			}
			cur_tagset_status := DetailData{}
			// last pointlist data
			cur_data := *series.GetPointlist()[len(series.GetPointlist())-1][1]

			if cur_data == 0 {
				status_data[name] = false
				log.Println("SERVICE METRIC STATUS:", name, false, series.GetTagSet())
				cur_tagset_status = DetailData{Status: "FAILED", Test_name: strings.Split(series.GetTagSet()[0], ":")[1], Success_rate: 0}
				res_status = "FAILED"
			} else {
				cur_tagset_status = DetailData{Status: "OK", Test_name: strings.Split(series.GetTagSet()[0], ":")[1], Success_rate: 0}
			}

			failed_tagset[name][strings.Split(series.GetTagSet()[0], ":")[1]] = cur_tagset_status
		}
	}
	status_data[name] = true
	return res_status
}

func process_status_for_test_name(name string, data datadogV1.MetricsQueryResponse) interface{} {
	if len(data.GetSeries()) == 0 {
		return "No Data"
	}
	res_status := "OK"
	for _, series := range data.GetSeries() {
		if len(series.GetPointlist()) > 0 {
			if len(series.GetTagSet()) == 0 {
				log.Println("SERVICE METRIC STATUS: No Tagset Found, skipped")
				continue
			}
			// cal success_rate
			var success_rate interface{}

			cur_tagset_status := DetailData{}
			// last pointlist data
			cur_data := *series.GetPointlist()[len(series.GetPointlist())-1][1]
			metric_status_detail_last_update[name] = int64(*series.GetPointlist()[len(series.GetPointlist())-1][0])
			if cur_data != 1 {
				status_data[name] = false
				log.Println("SERVICE METRIC STATUS:", name, false, series.GetTagSet(), cur_data)
				cur_tagset_status = DetailData{Status: "FAILED", Test_name: strings.Split(series.GetTagSet()[0], ":")[1], Success_rate: success_rate}
				res_status = "FAILED"
			} else {
				cur_tagset_status = DetailData{Status: "OK", Test_name: strings.Split(series.GetTagSet()[0], ":")[1], Success_rate: success_rate}
			}

			failed_tagset[name][strings.Split(series.GetTagSet()[0], ":")[1]] = cur_tagset_status
		}
	}
	status_data[name] = true
	return res_status
}

func GetMetricData(query string, from int64, to int64) MetricResponse {
	query_split := strings.Split(query, "$$")
	query = query_split[0]
	// Can hard code default value for custom_unit and aggregation here
	custom_unit, aggregation := "", ""
	if len(query_split) > 1 {
		custom_unit = query_split[1]
	}
	if len(query_split) > 2 {
		aggregation = query_split[2]
	}
	if len(query_split) > 3 {
		fixed_time := query_split[3]
		time_range := strings.Split(fixed_time, "-")
		if len(time_range) < 2 {
			log.Println("GET METRIC DATA WARNING: FIXED TIME RANGE CONFIGURATION ERROR: WRONG CONFIGURATION FOUND LESS THAN 2", time_range)
		}
		final_time_range := []int64{}
		now := time.Now()

		for _, cur_time := range time_range {
			detail_time := strings.Split(cur_time, ".")
			cur_time_unit := []int{0, 0, 0, 0}
			for i, detail_time_unit := range detail_time {
				detail_time_unit_value, err := strconv.Atoi(detail_time_unit)
				if err != nil {
					log.Println("GET METRIC DATA WARNING: FIXED TIME RANGE CONFIGURATION ERROR: WRONG CONFIGURATION FOR DETAIL TIME:", err)
					break
				}
				cur_time_unit[i] = detail_time_unit_value
			}
			if len(cur_time_unit) < 2 {
				log.Println("GET METRIC DATA WARNING: FIXED TIME RANGE CONFIGURATION ERROR: WRONG CONFIGURATION FOR DETAIL TIME:", cur_time_unit)
				break
			}
			cur_date := time.Date(now.Year(), now.Month(), now.Day(), cur_time_unit[0], cur_time_unit[1], cur_time_unit[2], cur_time_unit[3], location)
			final_time_range = append(final_time_range, cur_date.Unix())
		}
		if len(final_time_range) >= 2 {
			from = final_time_range[0]
			to = final_time_range[1]
		} else {
			log.Println("GET METRIC DATA WARNING: CUSTOM FIXED TIME CONFIGURATION GET ERROR: NO FINAL TIME RANGE FOUND")
		}
	}
	if query == "test" {
		return mockMetricResponse
	}
	metric_data := metric.TimeseriesPointQueryData(from, to, query)
	return UnitProcess(custom_unit, ProcessMetricData(metric_data, aggregation))
}

func ProcessMetricMongoData(data []mongodb.Datapoint, custom_unit string, aggeration string) MetricResponse {
	res := MetricResponse{}

	if aggeration == "" {
		log.Println("ProcessMetricMongoData: Error: Aggregation not found")
		return res
	}

	if len(data) == 0 {
		log.Println("ProcessMetricMongoData: Warning: No data points provided")
		return res
	}

	aggeration = strings.ToLower(aggeration)

	switch aggeration {
	case "all":
		for _, dataPoint := range data {
			res.Value = append(res.Value, []float64{float64(dataPoint.Timestamp), dataPoint.Data})
		}

	case "last":
		res.Value = append(res.Value, []float64{0.0, data[len(data)-1].Data})

	case "sum":
		var sum float64
		for _, dataPoint := range data {
			sum += dataPoint.Data
		}
		res.Value = append(res.Value, []float64{0.0, sum})

	case "avg":
		var sum float64
		for _, dataPoint := range data {
			sum += dataPoint.Data
		}
		average := sum / float64(len(data))
		res.Value = append(res.Value, []float64{0.0, average})

	default:
		log.Printf("ProcessMetricMongoData: Warning: Unrecognized aggregation '%s'\n", aggeration)
	}

	return res
}

func UnitProcess(custom_unit string, data MetricResponse) MetricResponse {
	if custom_unit == "" {
		return data
	}
	if len(data.Value) == 0 || len(data.Value) > 1 || len(data.Units) == 0 {
		return data
	}
	if custom_unit == "none" {
		data.Units = nil
		return data
	}
	if custom_unit == "convert" {
		unit_family, rate, unit_index := utils.Get_supported_unit_family_and_convertion_rate(data.Units[0].Family, data.Units[0].Name)
		converted_data, unit_name, unit_short := utils.Data_convertion(unit_family, rate, data.Value[0][1], unit_index)
		data.Value[0][1] = converted_data
		data.Units = []Unit{{Name: unit_name, Short_name: unit_short}}
		return data
	}
	data.Units = []Unit{{Short_name: custom_unit}}

	return data
}

func ProcessMetricData(data datadogV1.MetricsQueryResponse, aggeration string) MetricResponse {
	res := MetricResponse{Units: []Unit{}}

	if len(data.GetSeries()) == 0 {
		log.Println("Process Metric Data: Error: data series is null")
		return res
	}
	if len(data.GetSeries()[0].GetPointlist()) == 0 {
		log.Println("Process Metric Data: Error: data pointlist is null")
	}
	data_pointlist := data.GetSeries()[0].GetPointlist()
	if aggeration == "" {
		log.Println("Process Metric Data: Error: Aggeration not found")
		return res
	}
	for _, unit := range data.GetSeries()[0].GetUnit() {
		cur_unit := Unit{}
		cur_unit.Name = unit.GetName()
		cur_unit.Family = unit.GetFamily()
		cur_unit.Short_name = unit.GetShortName()
		res.Units = append(res.Units, cur_unit)
	}
	if strings.ToLower(aggeration) == "all" {
		for _, row := range data_pointlist {
			if len(row) > 1 {
				res.Value = append(res.Value, []float64{*row[0], *row[1]})
			}
		}
		return res
	}
	if strings.ToLower(aggeration) == "last" {
		res.Value = append(res.Value, []float64{0.0, *data_pointlist[len(data_pointlist)-1][1]})
		return res
	}
	var sum float64 = 0.0
	for _, data := range data_pointlist {
		sum += *data[1]
	}
	res.Value = append(res.Value, []float64{0.0, sum})
	if strings.ToLower(aggeration) == "sum" {
		return res
	}
	if strings.ToLower(aggeration) == "avg" {
		if len(res.Value) > 0 {
			res.Value = [][]float64{{0.0, res.Value[0][1] / float64(len(data_pointlist))}}
		}
		return res
	}

	return res
}
