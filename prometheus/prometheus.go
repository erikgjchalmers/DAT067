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

type ResourceUsage struct {
	CpuUsage float64
	MemUsage float64
}

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
func QueryOverTime(query string, api promv1.API, t promv1.Range) (model.Value, promv1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return api.QueryRange(ctx, query, t)
}
func Query(query string, api promv1.API) (model.Value, promv1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return api.Query(ctx, query, time.Now())
}

// TODO: Return the data and let the user choose the time interval as well as the resolution
func GetAvgCpuUsageOverTime() {
	strBuilder := fmt.Sprintf("avg_over_time(sum by (pod) (irate(container_cpu_usage_seconds_total{container != '', container != 'POD', pod != ''}[5m]))[1h:])")
	sTime := time.Now().Add(-24 * time.Hour)

	t := promv1.Range{Start: sTime, End: time.Now(), Step: time.Hour}
	result, _, _ := QueryOverTime(strBuilder, localAPI, t)
	matrix := result.(model.Matrix)
	printMatrix(matrix)

}

// TODO: Return the data and let the user choose the time interval as well as the resolution
func GetAvgMemUsageOverTime() {
	strBuilder := fmt.Sprintf("avg_over_time(sum by (pod) (container_memory_usage_bytes{container != '', container != 'POD', pod != ''})[1h:])")
	sTime := time.Now().Add(-24 * time.Hour)

	t := promv1.Range{Start: sTime, End: time.Now(), Step: time.Hour}
	result, _, _ := QueryOverTime(strBuilder, localAPI, t)
	matrix := result.(model.Matrix)
	printMatrix(matrix)

}

func printMatrix(m model.Matrix) {
	for _, sampleStream := range m {
		fmt.Printf("Metric: %v\n", (*sampleStream).Metric.String())
		for _, samplePair := range (*sampleStream).Values {
			fmt.Printf("\tTime stamp; %v, value; %v\n", samplePair.Timestamp.Time(), samplePair.Value)
		}
	}
}

// Gets available CPU capacity
func GetCPUNodeCapacity(node string) (float64, promv1.Warnings, error) {
	strBuilder := fmt.Sprintf("kube_node_status_capacity{resource='cpu', exported_node='%s'}", node)
	result, warnings, err := Query(strBuilder, localAPI)
	vector := result.(model.Vector)
	sample := vector[0]
	valueField := sample.Value
	value := float64(valueField)
	return value, warnings, err
}

func GetMemoryNodeCapacity(node string) (float64, promv1.Warnings, error) {
	strBuilder := fmt.Sprintf("kube_node_status_capacity{resource='memory', exported_node='%s'}", node)
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
	resourceUsageQuery := fmt.Sprintf("kube_node_status_capacity{resource='%s', exported_node='%s'} - avg_over_time(kube_node_status_allocatable{resource='%s', exported_node='%s'}[1h])", resource, node, resource, node)
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
	usageMap := make(map[string]float64)

	if !ok {
		return nil, nil, fmt.Errorf("Pods memory usage query did not return a Vector.")
	}

	if len(vector) == 0 {
		return nil, nil, fmt.Errorf("Pods Memory usage query returned empty result.")
	}

	for _, sample := range vector {
		labelSet := model.LabelSet(sample.Metric)
		pod := string(labelSet["pod"])
		usageMap[pod] = float64(sample.Value)
	}

	return usageMap, warnings, err
}

func GetPodsResourceUsage(node string) (map[string]ResourceUsage, promv1.Warnings, error) {
	cpuUsages, cpuWarnings, cpuErrors := GetPodsCPUUsage(node)
	memUsages, memWarnings, memErrors := GetPodsMemoryUsage(node)

	var warnings []string
	resources := make(map[string]ResourceUsage)

	if cpuErrors != nil {
		return nil, cpuWarnings, cpuErrors
	}

	if memErrors != nil {
		return nil, memWarnings, memErrors
	}

	if cpuWarnings != nil {
		warnings = append(warnings, cpuWarnings...)
	}

	if memWarnings != nil {
		warnings = append(warnings, memWarnings...)
	}

	for pod, cpuValue := range cpuUsages {
		memValue, ok := memUsages[pod]

		if !ok {
			continue
		}

		resources[pod] = ResourceUsage{
			CpuUsage: cpuValue,
			MemUsage: memValue,
		}
	}

	return resources, warnings, nil
}

/*
 *Returns map with replicaset as key and deployment it belong to as value
 *
 */
func getReplicasetToDeployment() (map[string]string, promv1.Warnings, error) {
	//creat query that gets all pods in cluster

	result, warnings, err := Query("kube_replicaset_owner{owner_kind='Deployment'}", localAPI)
	vector, ok := result.(model.Vector)

	if !ok {
		return nil, nil, fmt.Errorf("Pods CPU usage query did not return a Vector.")
	}

	if len(vector) == 0 {
		return nil, nil, fmt.Errorf("Pods CPU usage query returned empty result.")
	}
	resultMap := make(map[string]string)

	for _, sample := range vector {
		labelSet := model.LabelSet(sample.Metric)
		replicaSet := string(labelSet["replicaset"])
		deployment := string(labelSet["owner_name"])

		//fmt.Printf("Pod %s is running on node %s and is currently using %.6f cores of CPU.\n", pod, node, float64(sample.Value))
		resultMap[replicaSet] = deployment
	}

	return resultMap, warnings, err

}

/*
*Returns a map with pod as key and which replicaset it belongs to as value
*
 */
func getPodsToReplicaset() (map[string]string, promv1.Warnings, error) {
	//creat query that gets all pods in cluster

	result, warnings, err := Query("kube_pod_owner{owner_kind='ReplicaSet'}", localAPI)

	vector, ok := result.(model.Vector)

	if !ok {
		return nil, nil, fmt.Errorf("Pods CPU usage query did not return a Vector.")
	}

	if len(vector) == 0 {
		return nil, nil, fmt.Errorf("Pods CPU usage query returned empty result.")
	}
	resultMap := make(map[string]string)

	for _, sample := range vector {
		labelSet := model.LabelSet(sample.Metric)
		pod := string(labelSet["pod"])
		replicaset := string(labelSet["owner_name"])

		//fmt.Printf("Pod %s is running on node %s and is currently using %.6f cores of CPU.\n", pod, node, float64(sample.Value))
		resultMap[pod] = replicaset
	}

	return resultMap, warnings, err

}

/*
*Returns map with keys as pod and value as deployment
*Gives out which deployment each pod belongs to
 */
func GetPodsToDeployment() map[string]string {
	repTodep := make(map[string]string)
	podsToRep := make(map[string]string)
	resultMap := make(map[string]string)
	podsToRep, warnings, err := getPodsToReplicaset()
	if warnings != nil {
		println("kube_pod_owner warning")
	}
	if err != nil {
		println("kube_pod_owner error")
	}
	repTodep, warnings, err = getReplicasetToDeployment()
	if warnings != nil {
		println("kube_pod_owner warning")
	}
	if err != nil {
		println("kube_pod_owner error")
	}
	// loop to match every pod with a deploymet with help of the replicaset the pod belongs to
	for key, element := range podsToRep {

		var deployment, ok = repTodep[element]
		if !ok {
			fmt.Print("PANIC!")
		}
		resultMap[key] = deployment

	}
	return resultMap

}

/*
*Returns a string slice with all pods in a specific node. Take in node as argument
*
 */
func GetPodsOfNode(node string) ([]string, promv1.Warnings, error) {
	strBuilder := fmt.Sprintf("kube_pod_info{exported_node='%s'}", node)
	result, warnings, err := Query(strBuilder, localAPI)

	vector, ok := result.(model.Vector)

	if !ok {
		return nil, nil, fmt.Errorf("Pods CPU usage query did not return a Vector.")
	}

	if len(vector) == 0 {
		return nil, nil, fmt.Errorf("Pods CPU usage query returned empty result.")
	}
	//resultMap := make(map[string]string)
	a := []string{}

	for _, sample := range vector {
		labelSet := model.LabelSet(sample.Metric)
		//node := string(labelSet["node"])
		pod := string(labelSet["pod"])

		//fmt.Printf("Pod %s is running on node %s and is currently using %.6f cores of CPU.\n", pod, node, float64(sample.Value))
		a = append(a, pod)
	}
	return a, warnings, err
}

type prometheusInterface interface {
	GetCPUNodeCapacity(node string) (float64, promv1.Warnings, error)
}
