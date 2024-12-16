package monitor

import (
	"context"
	"log"
	"net/http"
	"pigdata/datadog/processingServer/internal/webhook"
	baserespone "pigdata/datadog/processingServer/pkg/baseRespone"
	"pigdata/datadog/processingServer/pkg/redis"
	"strconv"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

var (
	configuration = datadog.NewConfiguration()
	apiClient     = datadog.NewAPIClient(configuration)
	monitorApi    = datadogV1.NewMonitorsApi(apiClient)
	eventApi      = datadogV2.NewEventsApi(apiClient)
)

type MonitorSearchResponse struct {
	// The number of found monitors with the listed value.
	Count *int64 `json:"count,omitempty"`
	// The facet value.
	Name interface{} `json:"name,omitempty"`
}

func init() {
	configuration.SetUnstableOperationEnabled("v2.QueryTimeseriesData", true)
}

type MonitorResponse struct {
	Count int64       `json:"count"`
	Name  interface{} `json:"name"`
}

func checkForNewData(ctx context.Context, max int64, min int64, service_name string) (string, bool) {
	// data stored key: max-ts-service value: ts
	maxed, err := redis.GetIntData(ctx, "max-ts-"+service_name)
	if err != nil {
		log.Println("Get Max TS failed", err)
		return "", true
	}
	if max > int64(maxed) {
		return "", true
	}
	mined, err := redis.GetIntData(ctx, "min-ts-"+service_name)
	if err != nil {
		log.Println("Get Min TS failed", err)
		return "", true
	}
	if min < int64(mined) {
		return "", true
	}
	return strconv.Itoa(mined) + "-" + strconv.Itoa(maxed) + "-" + service_name, false
}

func MonitorSearch(ctx context.Context, query string, perpage int64, key string, isRedis bool) ([]byte, *http.Response, error) {
	// data stored key: min-max-service value: ts.status-ts.status
	// check need new (out of range) ? -> no -> return min-max key -> get data from redis with min-max key -> process data turn ts.status to ts and status -> prepare data and return
	// check need new -> yes -> get data from datadog -> set new min-ts max-ts -> process data -> store to redis -> return

	stringSplit := strings.Split(key, "-")
	strmin, strmax, service_name := stringSplit[0], stringSplit[1], stringSplit[2]
	if len(stringSplit) > 3 {
		service_name += stringSplit[3]
	}

	min, _ := strconv.ParseInt(strmin, 10, 64)
	max, _ := strconv.ParseInt(strmax, 10, 64)

	// Check if min max from request is in range of stored min max
	key, needNew := "", true
	if isRedis {
		key, needNew = checkForNewData(ctx, max, min, service_name)
	}

	if needNew {
		data, response, err := monitorApi.SearchMonitors(ctx, *datadogV1.NewSearchMonitorsOptionalParameters().WithQuery(query).WithPerPage(perpage))
		if err != nil {
			log.Println("Datadog response failed:", err)
			return nil, response, err
		}
		if isRedis {
			err = redis.SetData(ctx, max, "max-ts-"+service_name)
			if err != nil {
				log.Println("Set Redis failed", err)
			}
			err = redis.SetData(ctx, min, "min-ts-"+service_name)
			if err != nil {
				log.Println("Set Redis failed", err)
			}
		} else {
			log.Println("Redis connection failed, set data skipped")
		}
		counts := map[string]int64{
			"Alert": 0,
			"Warn":  0,
		}
		// Prepare key to store to redis
		mkey := strmin + "-" + strmax + "-" + service_name
		sdata := ""
		// Process data
		for _, monitor := range data.Monitors {
			// Prepare data to return
			if monitor.GetLastTriggeredTs() >= min && monitor.GetLastTriggeredTs() <= max {
				// Prepare data to store to redis
				ts := strconv.Itoa(int(monitor.GetLastTriggeredTs()))
				sdata += ts + "." + string(monitor.GetStatus()) + "-"

				if monitor.GetStatus() == "Alert" {
					counts["Alert"] = counts["Alert"] + 1
				} else {
					counts["Warn"] = counts["Warn"] + 1
				}
			}
		}

		log.Println("Preparing data store to redis key:", mkey, "value:", sdata)

		if isRedis {
			err = redis.SetData(ctx, sdata, mkey)
			if err != nil {
				log.Println("Store data", sdata, "failed", err)
			}
		} else {
			log.Println("Redis connection failed, set data skipped")
		}

		var dataRes []MonitorResponse
		for key, val := range counts {
			dataRes = append(dataRes, MonitorResponse{
				Count: val,
				Name:  key,
			})
		}

		return baserespone.DatadogBaseRespone(dataRes, response, nil)
	}

	// Get Data from redis
	data := redis.GetStringData(ctx, key)
	if data == "" {
		log.Println("Data is null")
	}
	monitors := strings.Split(data, "-")
	counts := map[string]int64{
		"Alert": 0,
		"Warn":  0,
	}

	for _, monitor := range monitors {
		monitorDetail := strings.Split(monitor, ".")
		ts, _ := strconv.ParseInt(monitorDetail[0], 10, 64)
		if ts >= min && ts <= max {
			if monitorDetail[1] == "Alert" {
				counts["Alert"] = counts["Alert"] + 1
			} else {
				counts["Warn"] = counts["Warn"] + 1
			}
		}
	}

	var dataRes []MonitorResponse
	for key, val := range counts {
		dataRes = append(dataRes, MonitorResponse{
			Count: val,
			Name:  key,
		})
	}

	return baserespone.DatadogBaseRespone(dataRes, nil, nil)
}

func AllMonitorDetails(ctx context.Context, group_state string) ([]byte, *http.Response, error) {
	return baserespone.DatadogBaseRespone(monitorApi.ListMonitors(ctx, *datadogV1.NewListMonitorsOptionalParameters().WithGroupStates(group_state)))
}

func AMonitorDetail(ctx context.Context, monitor_id int64) ([]byte, *http.Response, error) {
	return baserespone.DatadogBaseRespone(monitorApi.GetMonitor(ctx, monitor_id, *datadogV1.NewGetMonitorOptionalParameters()))
}

func GetMonitorFromAlert(ctx context.Context, alertIDs []string, isRedis bool) []MonitorResponse {
	res := []MonitorResponse{{Count: 0, Name: "Alert"}, {Count: 0, Name: "Warn"}, {Count: 0, Name: "No Data"}}
	if !isRedis {
		return res
	}
	for _, alertID := range alertIDs {
		webhookData, err := webhook.GetWebhook(ctx, alertID)
		if err != nil {
			log.Println("Get Webhook at Monitor error:", err, " - return with no data")
			continue
		}
		status := webhookData.AlertType

		if strings.ToLower(status) == "warning" {
			res[1].Count++
			continue
		}

		if strings.ToLower(status) == "error" {
			if strings.ToLower(webhookData.AlertTransition) == "no data" {
				res[2].Count++
			} else {
				res[0].Count++
			}
		}
	}
	return res
}
