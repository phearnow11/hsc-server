package main

import (
	"log"
	"pigdata/datadog/processingServer/internal/metric"
	"pigdata/datadog/processingServer/pkg/mongodb"
	"pigdata/datadog/processingServer/pkg/utils"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var (
	err_msg_prefix = "SCRIPT FILE:"
	location, _    = time.LoadLocation("Asia/Bangkok") // Commonly used for GMT+7
)

func init() {
	godotenv.Load(".env")
}

func main() {
	cur_time := time.Now()
	cur_date := time.Date(cur_time.Year(), cur_time.Month(), cur_time.Day(), cur_time.Hour(), cur_time.Minute(), cur_time.Second(), 0, location)

	to := cur_date.Unix()
	from := time.Now().Add(-3 * time.Minute).Unix()
	var wg sync.WaitGroup

	queries, _ := get_metric_queries()
	for _, query := range queries {
		wg.Add(1)
		go func(query string) {
			defer wg.Done()
			tried := 0
			for tried < 3 {
				if query == "test" {
					log.Println(err_msg_prefix, "Skip test query")
					break
				}
				data := metric.TimeseriesPointQueryData(from, to, query)

				if data.Query != nil {
					if len(data.GetSeries()) > 0 {
						pointlist := data.GetSeries()[0].GetPointlist()
						if len(pointlist) > 0 {
							data_point_data := []interface{}{}
							for _, datapoint := range pointlist {
								bson_data, err := mongodb.StructToBsonM(mongodb.Datapoint{Query: *data.Query, Timestamp: int64(*datapoint[0]), Data: *datapoint[1]})
								if err != nil {
									log.Println(err_msg_prefix, "Convert to BSON error:", err)
								}
								data_point_data = append(data_point_data, bson_data)
							}
							err := mongodb.Bulkwrite_upsert(data_point_data, mongodb.DATAPOINTS_COLLECTION)
							if err != nil {
								log.Println(err_msg_prefix, err)
							}
							break
						}
					}
					break
				}
				tried += 1
				log.Println(err_msg_prefix, "Error Get Datadog API, Query:", query, "Retried:", tried)
			}
		}(query)
		wg.Wait()
	}
}

func get_metric_queries() ([]string, map[string]string) {
	res := []string{}
	unit := map[string]string{}
	for _, querymap := range utils.Metrics.BusinessMetric {
		for _, query := range querymap {
			query_spilt := strings.Split(query, "$$")
			query = query_spilt[0]
			if len(query_spilt) > 1 {
				unit[query] = query_spilt[1]
			}
			res = append(res, query)
		}
	}
	for _, querymap := range utils.Metrics.ServiceMetric {
		for _, subquerymap := range querymap {
			for _, query := range subquerymap {
				query_spilt := strings.Split(query, "$$")
				query = query_spilt[0]
				if len(query_spilt) > 1 {
					unit[query] = query_spilt[1]
				}
				res = append(res, query)
			}
		}
	}

	return res, unit
}
