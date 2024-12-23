package main

import (
	"encoding/json"
	"fmt"
	"log"
	"pigdata/datadog/processingServer/internal/metric"
	"pigdata/datadog/processingServer/pkg/mongodb"
	"pigdata/datadog/processingServer/pkg/utils"
	"strings"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/gofiber/fiber/v2"
)

type timstamp struct {
	From int64 `json:"from" pg:"from" example:"1725485662"`
	To   int64 `json:"to" pg:"to" example:"1725489262"`
}
type MetricResponse struct {
	Value [][]float64 `json:"value"`
	Units []Unit      `json:"units"`
}

type Webhook struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Unit struct {
	Family     string `json:"family"`
	Name       string `json:"name"`
	Plural     string `json:"plural"`
	Short_name string `json:"short_name"`
}

var (
	status_data          = make(map[string]bool)
	status_data_new      = make(map[string]interface{})
	mockBusiness         = make(map[string]MetricResponse)
	mockService          = make(map[string]map[string]interface{})
	mockMetricResponse   = MetricResponse{Value: [][]float64{{0, 0}}, Units: []Unit{}}
	endpoint_failed_data = make(map[string]map[string]string)
)

func SimpleResponseReturn(ctx *fiber.Ctx, success bool, message string, data interface{}, statusCode int) error {
	ctx.Set("Content-Type", "application/json")
	ctx.Status(statusCode)

	if data == nil {
		return ctx.SendString(message)
	}

	return ctx.Send(data.([]uint8))
}

func MetricHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		res := make(map[string]interface{})
		var timestampBody timstamp
		if err := c.BodyParser(&timestampBody); err != nil {
			log.Println(err)
			return SimpleResponseReturn(c, false, "Check your body format", nil, 422)
		}
		if time.Now().Unix()-timestampBody.To < -300 {
			log.Println("Timestamp more than 5mins to the future")
			return SimpleResponseReturn(c, false, "Timestamp sending to the future", nil, 400)
		}

		if timestampBody.From == 0 || timestampBody.To == 0 {
			log.Println("Missing")
			return SimpleResponseReturn(c, false, "Missing from or to timestamp", nil, 400)
		}
		metric_type := c.Params("metric_type")
		cluster := c.Params("cluster")
		queries, err := utils.Lookup(metric_type, cluster)
		if err != nil {
			message := "query string for cluster " + cluster + " not found"
			log.Println(message)
			return SimpleResponseReturn(c, false, message, nil, 400)
		}
		for _, queryMap := range queries {
			metricData := populateMetricData(queryMap, timestampBody)
			if len(queries) > 1 {
				res[queryMap["name"]] = metricData
			} else {
				res = metricData
			}
		}
		return c.JSON(res)
	}
}

func StatusHandler() fiber.Handler {
	status_msg := "STATUS HANDLER:"
	return func(c *fiber.Ctx) error {
		res := map[string]interface{}{}
		var timestampBody timstamp
		if err := c.BodyParser(&timestampBody); err != nil {
			log.Println(status_msg, err)
			return SimpleResponseReturn(c, false, "Check your body format", nil, 422)
		}
		if time.Now().Unix()-timestampBody.To < -300 {
			log.Println(status_msg, "Timestamp more than 5mins to the future")
			return SimpleResponseReturn(c, false, "Timestamp sending to the future", nil, 400)
		}

		if timestampBody.From == 0 || timestampBody.To == 0 {
			log.Println(status_msg, "Missing timestamp")
			return SimpleResponseReturn(c, false, "Missing from or to timestamp", nil, 400)
		}
		status_type := c.Params("status_type")
		cluster := c.Params("cluster")
		status_names, err := utils.Lookup_status(status_type, cluster)
		if err != nil {
			log.Println(status_msg, "ERROR GETTING STATUS NAME:", err)
		}
		if status_names == nil {
			log.Println(status_msg, "ERROR STATUS NAME NOT FOUND:", status_type, cluster)
			return nil
		}
		if status_type == "business" {
			to := time.Now().In(location)
			from := to.Add(time.Minute * -6)
			return c.JSON(Get_status_process(status_names, from.Unix(), to.Unix(), "", ""))
		}
		for status_name, cluster := range status_names {
			if cluster == "none" {
				res[status_name] = nil
				continue
			}
			if Get_cluster_status(status_name, cluster) {
				res[status_name] = "OK"
			} else {
				res[status_name] = "FAILED"
			}
		}

		return c.JSON(res)
	}
}

func WebhookHandler() fiber.Handler {
	status_msg := "WEBHOOK HANDLER:"
	return func(c *fiber.Ctx) error {
		var body WebhookData
		c.BodyParser(&body)
		split_webhook := strings.Split(body.Raw_msg, "\n\n")
		var msg_status Raw_msg_status
		if len(split_webhook) > 1 {
			err := json.Unmarshal([]byte(split_webhook[1]), &msg_status)
			if err != nil {
				log.Println(status_msg, "Unmarshal error:", err)
			}
		}

		StatusProcess(msg_status, body.Alert_type, body.Alert_transition)

		return c.JSON(body)
	}
}

// webhook for only special case Order-Update
func WebhookHandlerOrderUpdate() fiber.Handler {
	status_msg := "WEBHOOK HANDLER FOR ORDER UPDATE:"
	return func(c *fiber.Ctx) error {
		var body WebhookData
		c.BodyParser(&body)
		split_webhook := strings.Split(body.Raw_msg, "\n\n")
		var pod_ready_data Raw_msg_pod_ready
		var error_rate_data Raw_msg_error_rate
		if len(split_webhook) > 1 {
			err := json.Unmarshal([]byte(split_webhook[1]), &pod_ready_data)
			if err != nil {
				log.Println("Unmarshal error:", err)
			}
			err = json.Unmarshal([]byte(split_webhook[1]), &error_rate_data)
			if err != nil {
				log.Println("Unmarshal error:", err)
			}
		}
		if pod_ready_data.Data.Podready == 0 || error_rate_data.Data.ErrorRate >= 10 {
			log.Println(status_msg, mongodb.Add_status_report("Order-Update", false))
		} else {
			log.Println(status_msg, mongodb.Add_status_report("Order-Update", true))
		}

		return c.JSON(body)
	}
}

func ServiceMetricDetailHandler() fiber.Handler {
	service_metric_detail_msg := "SERVICE METRIC DETAIL HANLDER:"
	return func(c *fiber.Ctx) error {
		service_name := c.Params("name")
		cluster := c.Params("cluster")
		cluster = strings.ToUpper(cluster)
		q := utils.Metrics.ServiceMetric[cluster][service_name]["Avail"]
		cur_date := time.Now()
		from := time.Date(cur_date.Year(), cur_date.Month(), cur_date.Day(), 8, 0, 0, 0, location)
		to := time.Date(cur_date.Year(), cur_date.Month(), cur_date.Day(), 15, 0, 0, 0, location)
		res := ServiceMetricDetailDataProcess(q, service_name, from.Unix(), to.Unix())
		fmt.Println(service_metric_detail_msg)
		return c.JSON(res)
	}
}

func GetDurationWithConvert(base_duration int) string {
	if base_duration > 999 {
		res := 0.0
		res, _, short_name := utils.Data_convertion(utils.TIME_FAMILY, 1000, float64(base_duration), 1)
		return fmt.Sprintf("%.2f%s", res, short_name)
	}
	return fmt.Sprintf("%dms", base_duration)
}

func ChartBaseline() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(base_line_data)
	}
}

func BaseLineCal() {
	baseline_msg := "GET BASE LINE:"
	to := time.Now().In(location)
	from := to.AddDate(0, -1, 0)
	charts_name := []string{"Logins-per-Day", "Orders-per-Day"}
	for _, chart := range charts_name {
		data := metric.TimeseriesPointQueryData(from.Unix(), to.Unix(), utils.Metrics.Baseline[chart])
		if len(data.GetSeries()) == 0 {
			log.Println(baseline_msg, "Error: Data is literally empty")
			break
		}
		if len(data.GetSeries()[0].GetPointlist()) == 0 {
			log.Println(baseline_msg, "No Pointlist Data found")
			break
		}
		datas_point := data.GetSeries()[0].GetPointlist()
		sum := 0.0
		for _, data_point := range datas_point {
			if datas_point[1] == nil {
				continue
			}
			sum += *data_point[1]
		}
		final_baseline := sum / float64(len(datas_point)) * 1.5
		base_line_data[chart] = final_baseline
	}
}

func log_datadog_api() fiber.Handler {
	return func(c *fiber.Ctx) error {
		now := time.Now()

		return c.SendString("Log triggered at " + now.String())
	}
}

func to_string_datadog(mqr datadogV1.MetricsQueryResponse) string {
	jsonData, err := json.MarshalIndent(mqr, "", "  ")
	if err != nil {
		return fmt.Sprintf("MetricsQueryResponse: error converting to JSON: %v", err)
	}
	return string(jsonData)
}

func trigger_log(messages []interface{}) {
	log.Println("/*******************************************************TRIGGERED_LOG*****************************************/")
	for _, message := range messages {
		log.Println("", message, "")
	}
	log.Println("/**************************************************", time.Now().In(location).String(), "**********************************************/")
}
