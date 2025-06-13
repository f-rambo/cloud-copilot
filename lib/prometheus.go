package lib

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

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

// QueryNodeResources queries resource usage metrics for a node
func (c *PrometheusClient) QueryNodeResources(ctx context.Context, nodes ...*biz.Node) (biz.MetricsResult, error) {
	nodeExporterDefaultPort := "9100"
	instances := make([]string, 0)
	nodeDevices := make([]string, 0)
	nodeMountpoints := make([]string, 0)
	for _, node := range nodes {
		instances = append(instances, node.Ip+":"+nodeExporterDefaultPort)
		for _, dev := range node.Disks {
			nodeDevices = append(nodeDevices, dev.Device)
			nodeMountpoints = append(nodeMountpoints, dev.Mountpoint)
		}
	}
	metrics := map[string]string{
		biz.CpuKey:     "100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)",
		biz.MemKey:     "avg_over_time(((node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes)[5m:]) * 100",
		biz.DiskKey:    "(node_filesystem_size_bytes{fstype!~\"tmpfs|overlay|squashfs\"} - node_filesystem_free_bytes{fstype!~\"tmpfs|overlay|squashfs\"}) / node_filesystem_size_bytes{fstype!~\"tmpfs|overlay|squashfs\"} * 100",
		biz.NetworkIn:  "sum by(instance) (rate(node_network_receive_bytes_total{device!~\"lo|docker.*|veth.*\"}[5m])/(1024*1024))",  // MB/s
		biz.NetworkOut: "sum by(instance) (rate(node_network_transmit_bytes_total{device!~\"lo|docker.*|veth.*\"}[5m])/(1024*1024))", // MB/s
	}
	results := biz.MetricsResult{}
	timeNow := time.Now()
	for metric, query := range metrics {
		metricPoint := biz.MetricPoint{Timestamp: timeNow}
		result, warnings, err := c.api.Query(ctx, query, timeNow)
		if err != nil {
			return results, errors.Wrapf(err, "error querying %s", metric)
		}
		if len(warnings) > 0 {
			return results, fmt.Errorf("warnings: %v", warnings)
		}
		vec, ok := result.(model.Vector)
		if !ok {
			continue
		}
		if len(vec) == 0 {
			continue
		}
		var sum float64
		var validCount int = 0
		for _, sample := range vec {
			if instance, ok := sample.Metric["instance"]; !ok || !slices.Contains(instances, string(instance)) {
				continue
			}
			if metric == biz.DiskKey {
				if device, ok := sample.Metric["device"]; !ok || !slices.Contains(nodeDevices, string(device)) {
					continue
				}
				if mountpoint, ok := sample.Metric["mountpoint"]; !ok || !slices.Contains(nodeMountpoints, string(mountpoint)) {
					continue
				}
			}
			if float64(sample.Value) != 0 {
				sum += float64(sample.Value)
				validCount++
			}
		}
		if validCount > 0 {
			metricPoint.Value = sum / float64(validCount)
		}
		switch metric {
		case biz.CpuKey:
			results.CPU = append(results.CPU, metricPoint)
		case biz.MemKey:
			results.Memory = append(results.Memory, metricPoint)
		case biz.DiskKey:
			results.Disk = append(results.Disk, metricPoint)
		case biz.NetworkIn:
			results.NetworkIn = append(results.NetworkIn, metricPoint)
		case biz.NetworkOut:
			results.NetworkOut = append(results.NetworkOut, metricPoint)
		case biz.GpuKey:
			results.GPU = append(results.GPU, metricPoint)
		case biz.GpuMemKey:
			results.GPUMem = append(results.GPUMem, metricPoint)
		default:
			continue
		}
	}
	return results, nil
}

func (c *PrometheusClient) QueryNodeRangeResources(ctx context.Context, timeRange biz.TimeRange, nodes ...*biz.Node) (biz.MetricsResult, error) {
	nodeExporterDefaultPort := "9100"
	instances := make([]string, 0)
	nodeDevices := make([]string, 0)
	nodeMountpoints := make([]string, 0)
	for _, node := range nodes {
		instances = append(instances, node.Ip+":"+nodeExporterDefaultPort)
		for _, dev := range node.Disks {
			nodeDevices = append(nodeDevices, dev.Device)
			nodeMountpoints = append(nodeMountpoints, dev.Mountpoint)
		}
	}
	metrics := map[string]string{
		biz.CpuKey:     "100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[1m])) * 100)",
		biz.MemKey:     "(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100",
		biz.DiskKey:    "(node_filesystem_size_bytes{fstype!~\"tmpfs|overlay|squashfs\"} - node_filesystem_free_bytes{fstype!~\"tmpfs|overlay|squashfs\"}) / node_filesystem_size_bytes{fstype!~\"tmpfs|overlay|squashfs\"} * 100",
		biz.NetworkIn:  "sum(rate(node_network_receive_bytes_total{device!~\"lo|docker.*|veth.*\"}[5m])/(1024*1024)) by (instance)",
		biz.NetworkOut: "sum(rate(node_network_transmit_bytes_total{device!~\"lo|docker.*|veth.*\"}[5m])/(1024*1024)) by (instance)",
	}
	end := time.Now()
	r := v1.Range{
		Start: end.Add(-time.Duration(timeRange.MustParseDuration())),
		End:   end,
		Step:  timeRange.CalculateMetricPointsStep(),
	}
	data := biz.MetricsResult{}
	for metric, query := range metrics {
		result, warnings, err := c.api.QueryRange(ctx, query, r)
		if err != nil {
			return data, errors.Wrapf(err, "error querying %s", metric)
		}
		if len(warnings) > 0 {
			return data, fmt.Errorf("warnings: %v", warnings)
		}
		matrix, ok := result.(model.Matrix)
		if !ok || len(matrix) == 0 {
			continue
		}
		timeValueMap := make(map[time.Time][]float64)
		for _, stream := range matrix {
			instance, ok := stream.Metric["instance"]
			if !ok || !slices.Contains(instances, string(instance)) {
				continue
			}
			if metric == biz.DiskKey {
				if device, ok := stream.Metric["device"]; !ok || !slices.Contains(nodeDevices, string(device)) {
					continue
				}
				if mountpoint, ok := stream.Metric["mountpoint"]; !ok || !slices.Contains(nodeMountpoints, string(mountpoint)) {
					continue
				}
			}
			for _, point := range stream.Values {
				timestamp := point.Timestamp.Time()
				timeValueMap[timestamp] = append(timeValueMap[timestamp], float64(point.Value))
			}
		}

		points := make(biz.MetricPoints, 0)
		for timestamp, values := range timeValueMap {
			if len(values) > 0 {
				var sum float64
				var validCount int
				for _, value := range values {
					if value != 0 {
						sum += value
						validCount++
					}
				}
				if validCount > 0 {
					points = append(points, biz.MetricPoint{
						Timestamp: timestamp,
						Value:     sum / float64(validCount),
					})
				}
			}
		}
		switch metric {
		case biz.CpuKey:
			data.CPU = points
		case biz.MemKey:
			data.Memory = points
		case biz.DiskKey:
			data.Disk = points
		case biz.NetworkIn:
			data.NetworkIn = points
		case biz.NetworkOut:
			data.NetworkOut = points
		case biz.GpuKey:
			data.GPU = points
		case biz.GpuMemKey:
			data.GPUMem = points
		default:
			continue
		}
	}
	return data, nil
}

func (c *PrometheusClient) QueryPodResources(ctx context.Context, service *biz.Service) (biz.MetricsResult, error) {
	if service == nil || len(service.Pods) == 0 {
		return biz.MetricsResult{}, nil
	}

	podNames := make([]string, 0)
	for _, pod := range service.Pods {
		podNames = append(podNames, pod.Name)
	}

	metrics := map[string]string{
		biz.CpuKey:     fmt.Sprintf("sum(rate(container_cpu_usage_seconds_total{pod=~\"%s\"}[5m])) by (pod) * 100", strings.Join(podNames, "|")),
		biz.MemKey:     fmt.Sprintf("sum(container_memory_working_set_bytes{pod=~\"%s\"}) by (pod) / sum(container_spec_memory_limit_bytes{pod=~\"%s\"}) by (pod) * 100", strings.Join(podNames, "|"), strings.Join(podNames, "|")),
		biz.NetworkIn:  fmt.Sprintf("sum(rate(container_network_receive_bytes_total{pod=~\"%s\"}[5m])) by (pod)/(1024*1024)", strings.Join(podNames, "|")),
		biz.NetworkOut: fmt.Sprintf("sum(rate(container_network_transmit_bytes_total{pod=~\"%s\"}[5m])) by (pod)/(1024*1024)", strings.Join(podNames, "|")),
	}

	results := biz.MetricsResult{}
	timeNow := time.Now()

	for metric, query := range metrics {
		metricPoint := biz.MetricPoint{Timestamp: timeNow}
		result, warnings, err := c.api.Query(ctx, query, timeNow)
		if err != nil {
			return results, errors.Wrapf(err, "error querying %s", metric)
		}
		if len(warnings) > 0 {
			return results, fmt.Errorf("warnings: %v", warnings)
		}

		vec, ok := result.(model.Vector)
		if !ok || len(vec) == 0 {
			continue
		}

		var sum float64
		var validCount int = 0
		for _, sample := range vec {
			if pod, ok := sample.Metric["pod"]; !ok || !slices.Contains(podNames, string(pod)) {
				continue
			}
			if float64(sample.Value) != 0 {
				sum += float64(sample.Value)
				validCount++
			}
		}

		if validCount > 0 {
			metricPoint.Value = sum / float64(validCount)
		}

		switch metric {
		case biz.CpuKey:
			results.CPU = append(results.CPU, metricPoint)
		case biz.MemKey:
			results.Memory = append(results.Memory, metricPoint)
		case biz.NetworkIn:
			results.NetworkIn = append(results.NetworkIn, metricPoint)
		case biz.NetworkOut:
			results.NetworkOut = append(results.NetworkOut, metricPoint)
		}
	}

	return results, nil
}

func (c *PrometheusClient) QueryPodRangeResources(ctx context.Context, timeRange biz.TimeRange, service *biz.Service) (biz.MetricsResult, error) {
	if service == nil || len(service.Pods) == 0 {
		return biz.MetricsResult{}, nil
	}

	podNames := make([]string, 0)
	for _, pod := range service.Pods {
		podNames = append(podNames, pod.Name)
	}

	metrics := map[string]string{
		biz.CpuKey:     fmt.Sprintf("sum(rate(container_cpu_usage_seconds_total{pod=~\"%s\"}[1m])) by (pod) * 100", strings.Join(podNames, "|")),
		biz.MemKey:     fmt.Sprintf("sum(container_memory_working_set_bytes{pod=~\"%s\"}) by (pod) / sum(container_spec_memory_limit_bytes{pod=~\"%s\"}) by (pod) * 100", strings.Join(podNames, "|"), strings.Join(podNames, "|")),
		biz.NetworkIn:  fmt.Sprintf("sum(rate(container_network_receive_bytes_total{pod=~\"%s\"}[5m])) by (pod)/(1024*1024)", strings.Join(podNames, "|")),
		biz.NetworkOut: fmt.Sprintf("sum(rate(container_network_transmit_bytes_total{pod=~\"%s\"}[5m])) by (pod)/(1024*1024)", strings.Join(podNames, "|")),
	}

	end := time.Now()
	r := v1.Range{
		Start: end.Add(-time.Duration(timeRange.MustParseDuration())),
		End:   end,
		Step:  timeRange.CalculateMetricPointsStep(),
	}

	data := biz.MetricsResult{}
	for metric, query := range metrics {
		result, warnings, err := c.api.QueryRange(ctx, query, r)
		if err != nil {
			return data, errors.Wrapf(err, "error querying %s", metric)
		}
		if len(warnings) > 0 {
			return data, fmt.Errorf("warnings: %v", warnings)
		}

		matrix, ok := result.(model.Matrix)
		if !ok || len(matrix) == 0 {
			continue
		}

		timeValueMap := make(map[time.Time][]float64)
		for _, stream := range matrix {
			pod, ok := stream.Metric["pod"]
			if !ok || !slices.Contains(podNames, string(pod)) {
				continue
			}

			for _, point := range stream.Values {
				timestamp := point.Timestamp.Time()
				timeValueMap[timestamp] = append(timeValueMap[timestamp], float64(point.Value))
			}
		}

		points := make([]biz.MetricPoint, 0)
		for timestamp, values := range timeValueMap {
			if len(values) > 0 {
				var sum float64
				var validCount int
				for _, value := range values {
					if value != 0 {
						sum += value
						validCount++
					}
				}
				if validCount > 0 {
					points = append(points, biz.MetricPoint{
						Timestamp: timestamp,
						Value:     sum / float64(validCount),
					})
				}
			}
		}

		switch metric {
		case biz.CpuKey:
			data.CPU = points
		case biz.MemKey:
			data.Memory = points
		case biz.NetworkIn:
			data.NetworkIn = points
		case biz.NetworkOut:
			data.NetworkOut = points
		}
	}

	return data, nil
}

/*
{__name__="hubble_http_requests_total", container="cilium-agent", destination="coreapi", destination_ip="10.244.2.116", destination_namespace="tenant-jobs", destination_workload="coreapi", endpoint="hubble-metrics", instance="172.18.0.4:9965", job="hubble-metrics", method="GET", namespace="kube-system", node="kind-worker", pod="cilium-kc5bd", protocol="HTTP/1.1", reporter="server", service="hubble-metrics", source="jobposting", source_ip="10.244.2.191", source_namespace="tenant-jobs", source_workload="jobposting", status="404", traffic_direction="ingress"}
{__name__="hubble_http_request_duration_seconds_sum", container="cilium-agent", destination="coreapi", destination_ip="10.244.2.116", destination_namespace="tenant-jobs", destination_workload="coreapi", endpoint="hubble-metrics", instance="172.18.0.4:9965", job="hubble-metrics", method="GET", namespace="kube-system", node="kind-worker", pod="cilium-kc5bd", reporter="server", service="hubble-metrics", source="jobposting", source_ip="10.244.2.191", source_namespace="tenant-jobs", source_workload="jobposting", traffic_direction="ingress"}
{__name__="hubble_http_request_duration_seconds_count", container="cilium-agent", destination="coreapi", destination_ip="10.244.2.116", destination_namespace="tenant-jobs", destination_workload="coreapi", endpoint="hubble-metrics", instance="172.18.0.4:9965", job="hubble-metrics", method="GET", namespace="kube-system", node="kind-worker", pod="cilium-kc5bd", reporter="server", service="hubble-metrics", source="jobposting", source_ip="10.244.2.191", source_namespace="tenant-jobs", source_workload="jobposting", traffic_direction="ingress"}
{__name__="hubble_http_request_duration_seconds_bucket", container="cilium-agent", destination="coreapi", destination_ip="10.244.2.116", destination_namespace="tenant-jobs", destination_workload="coreapi", endpoint="hubble-metrics", instance="172.18.0.4:9965", job="hubble-metrics", le="+Inf", method="GET", namespace="kube-system", node="kind-worker", pod="cilium-kc5bd", reporter="server", service="hubble-metrics", source="jobposting", source_ip="10.244.2.191", source_namespace="tenant-jobs", source_workload="jobposting", traffic_direction="ingress"}
type MetricPoints []MetricPoint
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}
*/
// CalculateQPS calculates the queries per second for a service
func (c *PrometheusClient) CalculateQPS(ctx context.Context, tr biz.TimeRange, service *biz.Service) (biz.MetricPoints, error) {
	workspace := biz.GetWorkspace(ctx)
	query := fmt.Sprintf(`sum(rate(hubble_http_requests_total{destination="%s",destination_namespace="%s",destination_workload="%s"}[1m])) by (destination)`,
		service.Name, workspace.Name, service.Name)

	end := time.Now()
	r := v1.Range{
		Start: end.Add(-time.Duration(tr.MustParseDuration())),
		End:   end,
		Step:  tr.CalculateMetricPointsStep(),
	}

	result, warnings, err := c.api.QueryRange(ctx, query, r)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying QPS")
	}
	if len(warnings) > 0 {
		return nil, fmt.Errorf("warnings: %v", warnings)
	}

	matrix, ok := result.(model.Matrix)
	if !ok || len(matrix) == 0 {
		return biz.MetricPoints{}, nil
	}

	points := make(biz.MetricPoints, 0)
	for _, stream := range matrix {
		for _, point := range stream.Values {
			points = append(points, biz.MetricPoint{
				Timestamp: point.Timestamp.Time(),
				Value:     float64(point.Value),
			})
		}
	}

	return points, nil
}

func (c *PrometheusClient) CalculateSuccessRate(ctx context.Context, tr biz.TimeRange, service *biz.Service) (float64, error) {
	workspace := biz.GetWorkspace(ctx)
	// 构建查询语句，分别统计总请求数和成功请求数（状态码小于500的请求）
	totalQuery := fmt.Sprintf(`sum(rate(hubble_http_requests_total{destination="%s",destination_namespace="%s",destination_workload="%s"}[1m])) by (destination)`,
		service.Name, workspace.Name, service.Name)
	successQuery := fmt.Sprintf(`sum(rate(hubble_http_requests_total{destination="%s",destination_namespace="%s",destination_workload="%s",status=~"[1-4].*"}[1m])) by (destination)`,
		service.Name, workspace.Name, service.Name)

	end := time.Now()
	r := v1.Range{
		Start: end.Add(-time.Duration(tr.MustParseDuration())),
		End:   end,
		Step:  tr.CalculateMetricPointsStep(),
	}

	// 查询总请求数
	totalResult, warnings, err := c.api.QueryRange(ctx, totalQuery, r)
	if err != nil {
		return 0, errors.Wrapf(err, "error querying total requests")
	}
	if len(warnings) > 0 {
		return 0, fmt.Errorf("warnings in total query: %v", warnings)
	}

	// 查询成功请求数
	successResult, warnings, err := c.api.QueryRange(ctx, successQuery, r)
	if err != nil {
		return 0, errors.Wrapf(err, "error querying success requests")
	}
	if len(warnings) > 0 {
		return 0, fmt.Errorf("warnings in success query: %v", warnings)
	}

	// 转换结果为矩阵格式
	totalMatrix, ok := totalResult.(model.Matrix)
	if !ok || len(totalMatrix) == 0 {
		return 0, nil
	}

	successMatrix, ok := successResult.(model.Matrix)
	if !ok || len(successMatrix) == 0 {
		return 0, nil
	}

	// 计算整体的成功率
	var totalSum float64
	var successSum float64

	// 累加所有时间点的值
	for _, stream := range totalMatrix {
		for _, point := range stream.Values {
			totalSum += float64(point.Value)
		}
	}

	for _, stream := range successMatrix {
		for _, point := range stream.Values {
			successSum += float64(point.Value)
		}
	}

	// 如果总请求数为0，返回100%的成功率
	if totalSum == 0 {
		return 100, nil
	}

	// 计算整体成功率（百分比）
	successRate := (successSum / totalSum) * 100

	return successRate, nil
}

func (c *PrometheusClient) QueryServerInfo(ctx context.Context) (map[string]any, error) {
	runtimeInfo, err := c.api.Runtimeinfo(ctx)
	if err != nil {
		return nil, err
	}

	serverInfo := map[string]any{
		"runtime_info": runtimeInfo,
		"base_url":     c.baseURL,
	}

	return serverInfo, nil
}

func (c *PrometheusClient) QueryAlerts(ctx context.Context, status ...v1.AlertState) ([]v1.Alert, error) {
	alerts, err := c.api.Alerts(ctx)
	if err != nil {
		return nil, err
	}
	if len(status) == 0 {
		return alerts.Alerts, nil
	}

	var filteredAlerts []v1.Alert
	for _, alert := range alerts.Alerts {
		if slices.Contains(status, alert.State) {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}
	return filteredAlerts, nil
}

func (c *PrometheusClient) QueryTargets(ctx context.Context) (v1.TargetsResult, error) {
	return c.api.Targets(ctx)
}
