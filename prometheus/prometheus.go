package prometheus

import (
	"context"
	"fmt"
	"time"

	//Might not be needed after proper error handling
	"os"

	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var localAPI promv1.API

func ImportantFunction() int {
	return 3
}

func CreateAPI(address string) promv1.API {
	client, err := api.NewClient(api.Config{
		Address: address,
	})

	if err != nil {
		fmt.Printf("An error occured when creating the client: %v\n", err)
		os.Exit(1)
	}
	localAPI = promv1.NewAPI(client)
	return localAPI
}

func Query(query string, api promv1.API) (model.Value, promv1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return api.Query(ctx, query, time.Now())
}

// Gets available CPU capacity
func GetCPUNodeCapacity(node string) (float64, promv1.Warnings, error) {
	strBuilder := fmt.Sprintf("kube_node_status_capacity{resource='cpu', node='%s'}", node)
	result, warnings, err := Query(strBuilder, localAPI)
	vector := result.(model.Vector)
	sample := vector[0]
	valueField := sample.Value
	value := float64(valueField)
	return value, warnings, err
}

func GetMemoryNodeCapacity(node string) (float64, promv1.Warnings, error) {
	strBuilder := fmt.Sprintf("kube_node_status_capacity{resource='memory', node='%s'}", node)
	result, warnings, err := Query(strBuilder, localAPI)
	vector := result.(model.Vector)
	sample := vector[0]
	valueField := sample.Value
	value := float64(valueField)
	return value, warnings, err
}

func GetCPUNodeUsage(node string) (float64, promv1.Warnings, error) {
	return getNodeResourceUsageQuery("cpu", node)
}

func GetMemoryNodeUsage(node string) (float64, promv1.Warnings, error) {
	return getNodeResourceUsageQuery("memory", node)
}

func getNodeResourceUsageQuery(resource string, node string) (float64, promv1.Warnings, error) {
	resourceUsageQuery := fmt.Sprintf("kube_node_status_capacity{resource='%s', node='%s'} - avg_over_time(kube_node_status_allocatable{resource='%s', node='%s'}[1h])", resource, node, resource, node)
	result, warnings, err := Query(resourceUsageQuery, localAPI)
	vector := result.(model.Vector)
	sample := vector[0]
	valueField := sample.Value
	value := float64(valueField)
	return value, warnings, err
}

/*
 * Author: Erik Wahlberger
 * Retrieves a map of pod-CPU usage key-value pairs. CPU usage is given in the amount of CPU cores being used by each respective pod
 */
func GetPodsCPUUsage(node string) (map[string]float64, promv1.Warnings, error) {
	resourceUsageQuery := fmt.Sprintf("sum(irate(container_cpu_usage_seconds_total{container!='POD', container!='', pod!='', instance='%s'}[5m])) by (pod)", node)
	result, warnings, err := Query(resourceUsageQuery, localAPI)
	vector, ok := result.(model.Vector)

	if !ok {
		return nil, nil, fmt.Errorf("Pods CPU usage query did not return a Vector.")
	}

	if len(vector) == 0 {
		return nil, nil, fmt.Errorf("Pods CPU usage query returned empty result.")
	}

	usageMap := make(map[string]float64)

	for _, sample := range vector {
		labelSet := model.LabelSet(sample.Metric)
		pod := string(labelSet["pod"])

		fmt.Printf("Pod %s is running on node %s and is currently using %.6f cores of CPU.\n", pod, node, float64(sample.Value))
		usageMap[pod] = float64(sample.Value)
	}

	return usageMap, warnings, err
}

/*
 * Author: Erik Wahlberger
 * Retrieves a map of pod-RAM usage key-value pairs. RAM usage is given in bytes being used by each respective pod
 */
func GetPodsMemoryUsage(node string) (map[string]float64, promv1.Warnings, error) {
	resourceUsageQuery := fmt.Sprintf("sum(container_memory_usage_bytes{container!='POD', container !='', pod != '', instance='%s'}) by (pod)", node)
	result, warnings, err := Query(resourceUsageQuery, localAPI)
	vector, ok := result.(model.Vector)

	if !ok {
		return nil, nil, fmt.Errorf("Pods memory usage query did not return a Vector.")
	}

	if len(vector) == 0 {
		return nil, nil, fmt.Errorf("Pods Memory usage query returned empty result.")
	}

	usageMap := make(map[string]float64)

	for _, sample := range vector {
		labelSet := model.LabelSet(sample.Metric)
		pod := string(labelSet["pod"])

		fmt.Printf("Pod %s is running on node %s and is currently using %.1f bytes of RAM.\n", pod, node, float64(sample.Value))
		usageMap[pod] = float64(sample.Value)
	}

	return usageMap, warnings, err
}

type prometheusInterface interface {
	GetCPUNodeCapacity(node string) (float64, promv1.Warnings, error)
}
