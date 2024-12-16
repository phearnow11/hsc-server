package main

import (
	"log"
	"pigdata/datadog/processingServer/internal/metric"
	"pigdata/datadog/processingServer/pkg/mongodb"
	"pigdata/datadog/processingServer/pkg/utils"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
)

func Get_status_process(status_names map[string]string, from, to int64, lquery string, lname string) interface{} {
	process_get_status_msg := "GET STATUS PROCESS:"
	OK_status := 1.0
	res := make(map[string]interface{}, len(status_names)-1)
	query := status_names["query"]
	var data datadogV1.MetricsQueryResponse
	if lquery == "" {
		data = metric.TimeseriesPointQueryData(from, to, query)
	} else {
		data = metric.TimeseriesPointQueryData(from, to, lquery)
	}
	status_data := make(map[string]interface{}, len(data.GetSeries()))
	for _, series := range data.GetSeries() {
		tag_set_split := []string{}
		if lquery == "" {
			if len(series.GetTagSet()) == 0 {
				log.Println(process_get_status_msg, "Tag set is null:", status_names)
				continue
			}
			tag_set_split = strings.Split(series.GetTagSet()[0], ":")
		} else {
			tag_set_split = append(tag_set_split, "query")
			tag_set_split = append(tag_set_split, lname)
		}
		status := "OK"
		if len(series.GetPointlist()) == 0 {
			status = "No Data"
		} else {
			data := series.GetPointlist()[len(series.GetPointlist())-1]
			if data[1] == nil {
				log.Println(process_get_status_msg, "Business Status last data point is null, Name:", tag_set_split[1])
				status = "No Data"
				continue
			}
			if *data[1] != OK_status {
				log.Println(process_get_status_msg, "Data at", *data[0], "is", *data[1], "status: NOT OK, Name:", tag_set_split[1])
				status = "Not OK"
			}
		}
		go func(data_pointlist [][]*float64) {
			save_data := []interface{}{}
			for _, data := range data_pointlist {
				if data[1] == nil {
					continue
				}
				save_data = append(save_data, mongodb.TagsetDatapoint{Tagset: tag_set_split[1], Timestamp: int64(*data[0]), Data: *data[1]})
			}
			err := mongodb.Bulkwrite_upsert(save_data, mongodb.STATUS_DATA_COLLECTION)
			if err != nil {
				log.Println(process_get_status_msg, "Error saving status data:", err)
			}
		}(series.GetPointlist())
		status_data[tag_set_split[1]] = status
	}

	if lquery != "" {
		if status_data[lname] == nil {
			return "No Data"
		}
		return status_data[lname]
	}
	for key, tag_set := range status_names {
		if key == "query" {
			continue
		}
		tag_set_split := strings.Split(tag_set, "&&")
		var _status interface{}
		for _, _tag_set := range tag_set_split {
			if strings.Count(_tag_set, ":") > 1 {
				_status = Get_status_process(status_names, from, to, _tag_set, key)
				break
			}

			status, exist := status_data[_tag_set]
			if !exist {
				_status = "No Data"
				break
			}
			if status == "Not OK" {
				_status = "Not OK"
				break
			}
			_status = "OK"
		}
		res[key] = _status
	}

	return res
}

func Get_cluster_status(status_name, cluster string) bool {
	get_status_msg := "RESPONSE GET STATUS:"
	check_dashboard := utils.Metrics.ServiceMetric[cluster]
	for name := range check_dashboard {
		data := status_data_new[name]
		log.Println(get_status_msg, "Status Name:", status_name, "Cluster:", cluster, "Data:", data)
		if data == nil {
			return false
		}
		if data == "FAILED" {
			return false
		}
	}
	return true
}
