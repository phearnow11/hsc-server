package utils

import (
	"fmt"
	"strings"
)

type QueryMetric struct {
	BusinessMetric                        map[string]map[string]string            `yaml:"business-metric"`
	ServiceMetric                         map[string]map[string]map[string]string `yaml:"service-metric"`
	ClusterStatus                         map[string]map[string]string            `yaml:"cluster-status"`
	BusinessStatus                        map[string]map[string]string            `yaml:"business-status"`
	SyntheticTest                         map[string][]string                     `yaml:"service-endpoint"`
	Baseline                              map[string]string                       `yaml:"baseline"`
	ServiceMetricDetailChart              map[string]string                       `yaml:"detail-chart"`
	ServiceMetricDetailOvrSuccessRate     map[string]string                       `yaml:"overall-success-rate"`
	ErrorDetailServiceMetricDetail        map[string]string                       `yaml:"error-detail-service-detail"`
	ServiceMetricDetailSuccessRate        map[string]string                       `yaml:"test-name-success-rate"`
	ServiceMetricDetailServiceSuccessRate map[string]string                       `yaml:"service-success-rate"`
}

var (
	Metrics      QueryMetric
	queries_file = "queryV2.yaml"
)

func init() {
	fmt.Println("Loading query yaml file:", getQuery(queries_file))
}

func Reset_query() error {
	return getQuery(queries_file)
}

func getQuery(file string) error {
	return ReadYAMLFile(file, &Metrics)
}

func writeQuery(file string) error {
	return WriteYAMLFile(file, &Metrics)
}

func Lookup_status(status_type string, cluster string) (map[string]string, error) {
	if status_type == "cluster" {
		if cluster == "all" {
			res := map[string]string{}
			for _, status_name := range Metrics.ClusterStatus {
				for status_name, service_cluster := range status_name {
					res[status_name] = service_cluster
				}
			}
			return res, nil
		}
		status_name, exist := Metrics.ClusterStatus[cluster]
		if exist {
			return status_name, nil
		}
	}
	if status_type == "business" {
		status_name, exist := Metrics.BusinessStatus[cluster]
		if exist {
			return status_name, nil
		}
	}
	return nil, nil
}

func Lookup(metric_type string, cluster string) ([]map[string]string, error) {
	if metric_type == "business" {
		metric_map, exist := Metrics.BusinessMetric[strings.ToUpper(cluster)]
		if exist {
			return []map[string]string{metric_map}, nil
		}
		return nil, fmt.Errorf("Business metric query not found")
	} else if metric_type == "service" {
		metric_map, exist := Metrics.ServiceMetric[strings.ToUpper(cluster)]
		if !exist {
			return nil, fmt.Errorf("Service metric query not found")
		}
		res := []map[string]string{}
		for dashboard_name, sub_metric := range metric_map {
			sub_metric["name"] = dashboard_name
			res = append(res, sub_metric)
		}
		return res, nil
	}
	return nil, fmt.Errorf("Metric type not found")
}
