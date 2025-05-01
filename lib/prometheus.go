package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// client, err := NewPrometheusClient("http://prometheus:9090")

// PrometheusClient represents a client for interacting with Prometheus
type PrometheusClient struct {
	api     v1.API
	baseURL string
}

// NewPrometheusClient creates a new Prometheus client
func NewPrometheusClient(baseURL string) (*PrometheusClient, error) {
	client, err := api.NewClient(api.Config{
		Address: baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	return &PrometheusClient{
		api:     v1.NewAPI(client),
		baseURL: baseURL,
	}, nil
}

// QueryByLabel queries metrics by label selectors
func (c *PrometheusClient) QueryByLabel(query string, labels map[string]string) (model.Value, error) {
	labelSelectors := ""
	for k, v := range labels {
		labelSelectors += fmt.Sprintf("%s=\"%s\",", k, v)
	}
	if len(labelSelectors) > 0 {
		labelSelectors = labelSelectors[:len(labelSelectors)-1]
		query = fmt.Sprintf("%s{%s}", query, labelSelectors)
	}

	result, _, err := c.api.Query(context.Background(), query, time.Now())
	return result, err
}

// QueryServiceMonitor queries metrics for a specific ServiceMonitor
func (c *PrometheusClient) QueryServiceMonitor(serviceName, namespace string) (model.Value, error) {
	query := fmt.Sprintf("up{service=\"%s\", namespace=\"%s\"}", serviceName, namespace)
	return c.QueryByLabel(query, nil)
}

// CalculateQPS calculates the queries per second for a service
func (c *PrometheusClient) CalculateQPS(serviceName, namespace string, duration time.Duration) (float64, error) {
	query := fmt.Sprintf("rate(http_requests_total{service=\"%s\", namespace=\"%s\"}[%s])",
		serviceName, namespace, duration.String())

	result, err := c.QueryByLabel(query, nil)
	if err != nil {
		return 0, err
	}

	if vec, ok := result.(model.Vector); ok && len(vec) > 0 {
		return float64(vec[0].Value), nil
	}
	return 0, fmt.Errorf("no QPS data found")
}

// QueryPodResources queries resource usage metrics for a pod
func (c *PrometheusClient) QueryPodResources(podName, namespace string) (map[string]float64, error) {
	metrics := map[string]string{
		"cpu":    "container_cpu_usage_seconds_total{pod=\"%s\", namespace=\"%s\"}",
		"memory": "container_memory_usage_bytes{pod=\"%s\", namespace=\"%s\"}",
		"disk":   "container_fs_usage_bytes{pod=\"%s\", namespace=\"%s\"}",
		"gpu":    "container_gpu_utilization{pod=\"%s\", namespace=\"%s\"}",
	}

	results := make(map[string]float64)
	for metric, query := range metrics {
		formattedQuery := fmt.Sprintf(query, podName, namespace)
		result, err := c.QueryByLabel(formattedQuery, nil)
		if err != nil {
			return nil, fmt.Errorf("error querying %s: %v", metric, err)
		}

		if vec, ok := result.(model.Vector); ok && len(vec) > 0 {
			results[metric] = float64(vec[0].Value)
		}
	}

	return results, nil
}

// QueryNodeResources queries resource usage metrics for a node
func (c *PrometheusClient) QueryNodeResources(nodeName string) (map[string]float64, error) {
	metrics := map[string]string{
		"cpu":    "node_cpu_seconds_total{instance=\"%s\"}",
		"memory": "node_memory_MemTotal_bytes{instance=\"%s\"}",
		"disk":   "node_filesystem_size_bytes{instance=\"%s\"}",
		"gpu":    "node_gpu_utilization{instance=\"%s\"}",
	}

	results := make(map[string]float64)
	for metric, query := range metrics {
		formattedQuery := fmt.Sprintf(query, nodeName)
		result, err := c.QueryByLabel(formattedQuery, nil)
		if err != nil {
			return nil, fmt.Errorf("error querying %s: %v", metric, err)
		}

		if vec, ok := result.(model.Vector); ok && len(vec) > 0 {
			results[metric] = float64(vec[0].Value)
		}
	}

	return results, nil
}
