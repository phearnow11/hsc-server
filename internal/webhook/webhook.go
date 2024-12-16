package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"pigdata/datadog/processingServer/internal/metric"
	"pigdata/datadog/processingServer/pkg/redis"
	"pigdata/datadog/processingServer/pkg/utils"
	"regexp"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
)

var (
	NO_DATA        string = "no data"
	OK             string = "ok"
	ALERT          string = "alert"
	WARNING        string = "warning"
	WebhookMockAPI        = WebhookReturn{
		MetricData:      metric.FakeData,
		Status:          OK,
		EvaluationValue: 1.0,
		EvaluationUnit:  "Mb",
	}
)

type WebhookReturn struct {
	MetricData      datadogV1.MetricsQueryResponse
	Status          string
	EvaluationValue float64
	EvaluationUnit  string
	Window          string
}

type WebHookBody struct {
	Id              string    `json:"id"`
	EventType       string    `json:"event_type"`
	Date            string    `json:"date"`
	Body            string    `json:"body"`
	RawMsg          string    `json:"raw_msg"`
	Alert_id        string    `json:"alert_id"`
	AlertMetric     string    `json:"alert_metric"`
	AlertQuery      string    `json:"alert_query"`
	AlertTransition string    `json:"alert_transition"`
	AlertType       string    `json:"alert_type"`
	AlertData       AlertData `json:"alert_data"`
}

type RawBody struct {
	Alert_data AlertData `json:"data"`
}

type AlertData struct {
	Warning          float64  `json:"warning"`
	Alert            float64  `json:"alert"`
	Value            float64  `json:"value"`
	First_triggered  int64    `json:"first_triggered"`
	Last_triggered   int64    `json:"last_triggered"`
	Trigger_duration int64    `json:"trigger_duration"`
	Instance         []string `json:"instance"`
}

func SaveWebhook(ctx context.Context, data interface{}, key string, isRedis bool) error {
	if !isRedis {
		return fmt.Errorf("Redis is disconnected")
	}
	return redis.SetData(ctx, data, key)
}

func GetWebhook(ctx context.Context, alertID string) (WebHookBody, error) {
	res := WebHookBody{}
	val, err := redis.GetData(ctx, alertID)
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(val, &res)
	if err != nil {
		return res, err
	}
	return res, nil
}

func AttachWebhookDataToMetric(ctx context.Context, metricData datadogV1.MetricsQueryResponse, alertID string, isRedis bool, customUnit string) (WebhookReturn, error) {
	res := WebhookReturn{
		MetricData: metricData,
		Status:     OK,
	}
	unit := customUnit
	if !isRedis {
		return res, fmt.Errorf("Redis is disconnected")
	}
	if unit != "" {
		log.Println("CUSTOM UNIT FOUND, CONVERTING")
		if unit == "(none)" {
			metricData.GetSeries()[0].SetUnit([]datadogV1.MetricsQueryUnit{})
		} else {
			unit_split := strings.Split(unit, "/")
			short_split := strings.Split(unit_split[0], "-")
			plural := short_split[0] + "s"
			old_unit := datadogV1.MetricsQueryUnit{}
			if len(metricData.GetSeries()[0].GetUnit()) > 0 {
				old_unit = metricData.GetSeries()[0].GetUnit()[0]
			}
			new_units := []datadogV1.MetricsQueryUnit{}
			new_units = append(new_units, datadogV1.MetricsQueryUnit{
				Family:      old_unit.Family,
				Name:        &short_split[0],
				Plural:      &plural,
				ScaleFactor: old_unit.ScaleFactor,
			})
			if len(short_split) > 1 {
				new_units[0].ShortName = &short_split[1]
				unit = short_split[1]
			}
			if len(unit_split) > 1 {
				new_units = append(new_units, metric.Time_unit)
				unit += "/s"
			}

			metricData.GetSeries()[0].SetUnit(new_units)
		}
		log.Println("CONVERTING SUCCESSFULLY")
	}

	// fix empty unit
	if len(metricData.GetSeries()) > 0 {
		metricData = metric.Empty_unit_bug_temp_solved(metricData)
	}

	webhookData, err := GetWebhook(ctx, alertID)
	if err != nil || webhookData.Id == "" {
		return res, err
	}
	log.Println("Webhook DATA found, proceed to attach: OLD UNIT:", unit)

	window := ""

	if unit == "" {
		alert_query := strings.Split(webhookData.AlertQuery, ":")
		if len(alert_query) > 0 {
			halfrightUnit := alert_query[0]
			if len(metricData.GetSeries()) > 0 {
				unit_name := ""
				for _, _unit := range metricData.GetSeries()[0].GetUnit() {
					if unit_name == "" {
						unit_name = _unit.GetName()
					}
					if _unit.GetShortName() == "" {
						unit += _unit.GetPlural() + "/"
						continue
					}
					unit += _unit.GetShortName() + "/"
				}
				// Temp resolve for dup empty unit bug (short name side) -- risk: low
				for i := len(unit) - 1; i >= 0; i-- {
					if unit[len(unit)-1] == '/' {
						unit = unit[:len(unit)-1]
					} else {
						break
					}
				}
				data, name, short := convert_data_unit(unit_name, webhookData.AlertData.Value)
				if name != "" {
					webhookData.AlertData.Value = data
					unit = short
				}

				window = half_right_unit_convertion(halfrightUnit)
			}
		}
	} else {
	}

	if unit == "(none)" {
		unit = ""
	}

	log.Println("Unit attach SUCCESSFULLY: FINAL UNIT:", unit)

	status := webhookData.AlertType
	log.Println(status)
	if strings.ToLower(status) == "success" {
		status = OK
		window = ""
		webhookData.AlertData.Value = 0
		unit = ""
	}

	if strings.ToLower(status) == "error" {
		if strings.ToLower(webhookData.AlertTransition) == "no data" {
			status = NO_DATA
		} else {
			status = ALERT
		}
	}

	res = WebhookReturn{
		MetricData:      metricData,
		Status:          status,
		EvaluationValue: webhookData.AlertData.Value,
		EvaluationUnit:  unit,
		Window:          window,
	}

	return res, nil
}

var half_right_format = `(\w+)\(([\w\d_]+)\)`

func half_right_unit_convertion(unit string) string {
	// format max(...)
	re := regexp.MustCompile(half_right_format)
	if !re.MatchString(unit) {
		log.Println("ERROR WEBHOOK ATTACH: Wrong unit format:", unit, "return without converting")
		return unit
	}
	sub_matches := re.FindStringSubmatch(unit)
	return "(" + sub_matches[len(sub_matches)-1] + ")"
}

var (
	BYTE_UNIT_FAMILY      = []string{"byte", "kilobyte", "megabyte", "gigabyte", "terabyte", ""}
	TIME_UNIT_FAMILY      = []string{"microsecond", "nanosecond", "millisecond", "second", "minute", "hour", ""}
	SUPPORTED_UNIT_FAMILY = [][]string{BYTE_UNIT_FAMILY, TIME_UNIT_FAMILY}
)

func convert_data_unit(unit_name string, data float64) (float64, string, string) {
	unit_index, convertion_rate, direction, unit_family := convert_condition(unit_name, data)
	if unit_index < 0 {
		return data, "", ""
	}
	if direction {
		return utils.Data_convert_up(unit_family, convertion_rate, data, unit_index)
	} else {
		return utils.Data_convert_down(unit_family, convertion_rate, data, unit_index)
	}
}

func convert_condition(unit_name string, data float64) (int, float64, bool, []string) {
	direction := false
	if data < 1 {
		direction = false
	} else if data >= 1000 {
		direction = true
	} else {
		return -1, 0, direction, []string{}
	}

	unit_index := utils.Contain(BYTE_UNIT_FAMILY, unit_name)
	if unit_index >= 0 {
		return unit_index, 1024, direction, BYTE_UNIT_FAMILY
	}
	unit_index = utils.Contain(TIME_UNIT_FAMILY, unit_name)
	if unit_index >= 0 {
		convertion_rate := 0.0
		if direction {
			if unit_index >= 3 {
				convertion_rate = 60
			} else {
				convertion_rate = 1000
			}
		} else {
			if unit_index >= 4 {
				convertion_rate = 60
			} else {
				convertion_rate = 1000
			}
		}
		return unit_index, convertion_rate, direction, TIME_UNIT_FAMILY
	}
	return -1, 0, direction, []string{}
}
