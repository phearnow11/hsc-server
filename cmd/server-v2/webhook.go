package main

import (
	"log"
	"pigdata/datadog/processingServer/pkg/utils"
	"strings"
)


type TestDetails struct {
	Status   bool        `json:"status"`
	Method   string      `json:"method"`
	Endpoint string      `json:"endpoint"`
	Duration string      `json:"duration"`
	Detail   interface{} `json:"detail"`
}

type WebhookData struct {
	Alert_type       string `json:"alert_type"`
	Alert_transition string `json:"alert_transition"`
	Raw_msg          string `json:"raw_msg"`
}

var pod_ready, error_rate = 0, 0

type Raw_msg_pod_ready struct {
	Data struct {
		Podready float64  `json:"pod-ready"`
		Instance []string `json:"instance"`
	} `json:"data"`
}

type Raw_msg_error_rate struct {
	Data struct {
		ErrorRate float64  `json:"error-rate"`
		Instance  []string `json:"instance"`
	} `json:"data"`
}

type Raw_msg_status struct {
	Kind string `json:"kind"`
	Data struct {
		Collection string  `json:"collection"`
		Endpoint   string  `json:"endpoint"`
		Success    float64 `json:"success"`
	} `json:"data"`
}

// Add endpoint to endpoint_failed_data if endpoint is failed
func StatusProcess(data Raw_msg_status, alert_type string, alert_transition string) {
	webhook_status_msg := "WEBHOOK STATUS PROCESS:"
	config_endpoints := utils.Metrics.SyntheticTest[data.Data.Collection]

	cur_endpoint_failed_data, exist := endpoint_failed_data[data.Data.Collection]
	if !exist {
		endpoint_failed_data[data.Data.Collection] = make(map[string]string)
		cur_endpoint_failed_data = endpoint_failed_data[data.Data.Collection]
	}

	// If specific endpoint is specified in config file, then only check those endpoint
	if len(config_endpoints) > 0 {
		if utils.Contain(config_endpoints, data.Data.Endpoint) >= 0 {
			EndpointProcess(data, cur_endpoint_failed_data, alert_type, alert_transition)
			return
		} else {
			log.Println(webhook_status_msg, "Endpoint is not specified in query file", data.Data.Endpoint+", skipped")
			return
		}
	}
	// If there is no endpoint specified
	EndpointProcess(data, cur_endpoint_failed_data, alert_type, alert_transition)
	return
}

func EndpointProcess(data Raw_msg_status, cur_endpoint_failed_data map[string]string, alert_type string, alert_transition string) {
	endpoint_failed_msg := "Process Endpoint:"

	// If current webhook data is avail
	if "success" == strings.ToLower(alert_type) {
		_, exist := cur_endpoint_failed_data[data.Data.Endpoint]
		if exist {
			delete(endpoint_failed_data[data.Data.Collection], data.Data.Endpoint)
			log.Println(endpoint_failed_msg, "Remove failed endpoint", data.Data.Endpoint, "successfully, current failed:", endpoint_failed_data[data.Data.Collection])
		}
		return
	}

	// If current is failed
	endpoint_failed_data[data.Data.Collection][data.Data.Endpoint] = strings.ToLower(alert_transition)
	log.Println(endpoint_failed_msg, "Add failed endpoint", data.Data.Endpoint, "successfully, current failed:", endpoint_failed_data[data.Data.Collection])
	return
}
