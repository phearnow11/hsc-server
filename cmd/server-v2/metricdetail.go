package main

import (
	"log"
	"math"
	"pigdata/datadog/processingServer/internal/metric"
	"pigdata/datadog/processingServer/pkg/utils"
	"strings"
	"time"
)

type ServiceMetricDetail struct {
	Test_num     int           `json:"tests_attempt"`
	Passed       int           `json:"passed"`
	Failed       int           `json:"failed"`
	Success      int           `json:"success"`
	Last_updated int64         `json:"last_updated"`
	List_data    []*DetailData `json:"list_data"`
}

type DetailData struct {
	Status       string      `json:"status"`
	Test_name    string      `json:"test"`
	Success_rate interface{} `json:"success"`
	Charts       interface{} `json:"charts"`
	Error_detail ErrorDetail `json:"error_detail"`
}

type ChartData struct {
	Timestamp int64   `json:"timestamp"`
	Data      float64 `json:"data"`
}

type ErrorDetail struct {
	Error_type    interface{} `json:"error_type"`
	Error_message interface{} `json:"error_message"`
}

var (
	failed_tagset                              = make(map[string]interface{})
	metric_status_detail_last_update           = make(map[string]int64)
	metric_status_detail_error_msg             = make(map[string]ErrorDetail)
	metric_status_detail_success_rate_testname = make(map[string]interface{})
)

func ServiceMetricDetailDataProcess(query, service_name string, from, to int64) interface{} {
	service_metric_detail_msg := "SERVICE METRIC DETAIL PROCESS:"

	queries := strings.Split(query, "&&&")
	failed, passed, _ := 0, 0, 0
	detail_datas := []*DetailData{}
	exist_testname := map[string]DetailData{}
	detail_datas_map := make(map[string]*DetailData)
	detail_status := make(map[string]interface{})

	for _, q := range queries {
		q_check := strings.Split(q, "by")
		if len(q_check) < 2 {
			log.Println(service_metric_detail_msg, "Error at configuration, wrong query format -- skipped")
			continue
		}
		//if !strings.Contains(q_check[1], "test_name") {
		//	log.Println(service_metric_detail_msg, "Metric detail support only for test_name query -- skipped")
		//	continue
		//}
		data := metric.TimeseriesPointQueryData(from, to, q)

		for _, series := range data.GetSeries() {
			cur_detail_data := DetailData{}
			if len(series.GetTagSet()) == 0 {
				continue
			}
			test_name := strings.Split(series.GetTagSet()[0], ":")[1]
			cur_detail_data, ok := failed_tagset[test_name].(DetailData)
			if !ok {
				cur_detail_data = DetailData{Status: "No Data", Test_name: test_name, Success_rate: nil}
			}

			deployment_detail, ok := exist_testname[cur_detail_data.Test_name]
			if ok {
				cur_detail_data = MergeMetricDetail(deployment_detail, cur_detail_data)
				test_name = cur_detail_data.Test_name
			} else {
				if !strings.Contains(cur_detail_data.Test_name, "-kafka") {
					exist_testname[cur_detail_data.Test_name+"-kafka"] = cur_detail_data
				}
			}

			if cur_detail_data.Status == "OK" {
				detail_status[test_name] = "OK"
			} else if cur_detail_data.Status == "FAILED" {
				detail_status[test_name] = "FAILED"
				cur_detail_data.Error_detail = metric_status_detail_error_msg[test_name]
			}

			// SuccessRateChartCalNoQuery(series.GetPointlist(), &cur_detail_data)
			detail_pt, ok := detail_datas_map[test_name]
			detail_datas_map[test_name] = &cur_detail_data
			if ok {
				*detail_pt = cur_detail_data
			} else {
				detail_datas = append(detail_datas, detail_datas_map[test_name])
			}

		}
		SuccessRateChartCalWQuery(detail_datas_map, from, to, service_name)
	}

	for _, status := range detail_status {
		if status == "OK" {
			passed++
		} else if status == "FAILED" {
			failed++
		}
	}
	cur_detail := ServiceMetricDetail{Test_num: len(detail_datas), Last_updated: metric_status_detail_last_update[service_name], List_data: detail_datas, Passed: passed, Failed: failed, Success: OverallSuccessRate(service_name)}

	return cur_detail
}

func SuccessRateChartCalNoQuery(data [][]*float64, detail_data *DetailData) {
	// Call success rate for a tagset
	success_rate := 0.0
	cnt := 0
	for i := len(data) - 1; i >= 0; i-- {
		if data[i][1] == nil {
			continue
		}
		cnt++
		success_rate += *data[i][1]
	}
	detail_data.Success_rate = math.Round(success_rate / float64(cnt) * 100)
	// cal chart
	chart_vals := []ChartData{}
	chart_cnt := 0
	chart_data_sum := 0.0
	for i, cur_data := range data {
		if cur_data[1] == nil {
			continue
		}
		chart_cnt++
		chart_data_sum += *cur_data[1]
		cur_ts := *cur_data[0]
		// Avg for each 5 datapoints
		if chart_cnt == 5 || i+1 == len(data) {
			cur_chart_data := ChartData{Timestamp: int64(cur_ts), Data: math.Round(chart_data_sum / float64(chart_cnt) * 100)}
			chart_vals = append(chart_vals, cur_chart_data)
			chart_cnt = 0
			chart_data_sum = 0
		}

	}
	detail_data.Charts = chart_vals
}

func SuccessRateChartCalWQuery(detailData_map map[string]*DetailData, from, to int64, service_name string) {
	success_rate_chart_cal_w_query_msg := "SUCCESS RATE AND CHART CALCULATION WITH QUERY:"
	chart_query, exist := utils.Metrics.ServiceMetricDetailChart[service_name]
	if !exist {
		log.Println(success_rate_chart_cal_w_query_msg, "Error: No chart query found with service name:", service_name)
		return
	}

	chart_data := metric.TimeseriesPointQueryData(from, to, chart_query)
	for _, series := range chart_data.GetSeries() {
		if len(series.GetTagSet()) == 0 {
			log.Println(success_rate_chart_cal_w_query_msg, "Data with no tagset, skipped")
			continue
		}
		test_name := strings.Split(series.GetTagSet()[0], ":")[1]
		if detailData_map[test_name] == nil {
			log.Println(success_rate_chart_cal_w_query_msg, "No test_name found in detail data map, skipped")
			continue
		}
		for _, cur_datapoint := range series.GetPointlist() {
			if cur_datapoint[1] == nil {
				continue
			}
			*cur_datapoint[1] = math.Round(*cur_datapoint[1] * float64(100))

		}
		success_rate, exist := metric_status_detail_success_rate_testname[test_name]
		if !exist {
			success_rate = nil
		}
		detailData_map[test_name].Success_rate = success_rate
		detailData_map[test_name].Charts = series.GetPointlist()
	}
}

func OverallSuccessRate(service_name string) int {
	overall_success_rate_calculate_msg := "OVERALL SUCCESS RATE CALCULATION:"
	to := time.Now().In(location)
	from := to.Add(time.Minute * -35)

	query, exist := utils.Metrics.ServiceMetricDetailOvrSuccessRate[service_name]
	if !exist {
		log.Println(overall_success_rate_calculate_msg, "Error: No service name found:", service_name)
		return 0
	}
	res := 0
	data := metric.TimeseriesPointQueryData(from.Unix(), to.Unix(), query)
	if len(data.GetSeries()) == 0 {
		log.Println(overall_success_rate_calculate_msg, "Error: No Series data:", service_name)
		return res
	}
	if len(data.GetSeries()[0].GetPointlist()) == 0 {
		log.Println(overall_success_rate_calculate_msg, "Error: No Data Point:", service_name)
		return res
	}
	// Get last data point
	res = int(*data.GetSeries()[0].GetPointlist()[len(data.GetSeries()[0].GetPointlist())-1][1] * 100)

	return res
}

func ServiceMetricErrorDetail(service_name, query string, from, to int64) {
	data := metric.TimeseriesPointQueryData(from, to, query)
	mapped_res := make(map[string]string)
	for _, series := range data.GetSeries() {
		for _, tag_set := range series.GetTagSet() {
			tag_set_split := strings.Split(tag_set, ":")
			mapped_res[tag_set_split[0]] = tag_set_split[1]
		}
		for range mapped_res {
			cur_error_detail := ErrorDetail{Error_type: mapped_res["error_type"], Error_message: mapped_res["error_message"]}
			metric_status_detail_error_msg[mapped_res["test_name"]] = cur_error_detail
		}
	}
}

func ServiceMetricDetailSuccessRateTestName(service_name, query string, from, to int64) {
	success_rate_test_name_msg := "SERVICE METRIC SUCCESS RATE EACH TEST NAME:"
	data := metric.TimeseriesPointQueryData(from, to, query)
	// jsonData, _ := json.MarshalIndent(data, "", "  ")
	// fmt.Println(string(jsonData))
	for _, series := range data.GetSeries() {
		if len(series.GetTagSet()) == 0 {
			log.Println(success_rate_test_name_msg, "Error: No tag set, skipped a series data with query:", query)
			continue
		}
		test_name := strings.Split(series.GetTagSet()[0], ":")[1]
		if len(series.GetPointlist()) == 0 {
			log.Println(success_rate_test_name_msg, "Warn: No Datapoint data, skipped test name:", test_name)
			continue
		}
		if series.GetPointlist()[len(series.GetPointlist())-1][1] == nil {
			log.Println(success_rate_test_name_msg, "Error: Last data point is null, skipped test name:", test_name)
			continue
		}
		data_point := *series.GetPointlist()[len(series.GetPointlist())-1][1]
		metric_status_detail_success_rate_testname[test_name] = data_point * 100
	}
}

func MergeMetricDetail(data1, data2 DetailData) DetailData {
	res := DetailData{Test_name: data1.Test_name}
	final_status := ""

	if data1.Status == "FAILED" || data2.Status == "FAILED" {
		final_status = "FAILED"
	} else if data1.Status == "No Data" || data2.Status == "No Data" {
		final_status = "No Data"
	} else {
		final_status = "OK"
	}
	res.Status = final_status

	return res
}
