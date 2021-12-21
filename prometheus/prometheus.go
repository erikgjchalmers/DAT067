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

type ResourceUsageSample struct {
	Time           time.Time
	ResourceUsages map[string]ResourceUsage
}

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

/*
 * Authors: Jessica Barai, Erik Wahlberger
 * Performs a Prometheus query over the specified time range t
 */
func QueryOverTime(query string, api promv1.API, t promv1.Range) (model.Value, promv1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return api.QueryRange(ctx, query, t)
}

func Query(query string, api promv1.API, t time.Time) (model.Value, promv1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return api.Query(ctx, query, t)
}

/*
 * Authors: Jessica Barai, Erik Wahlberger
 * Calculates the average CPU usage (in cores) over the specified resolution duration, and returns the average values between startTime and endTime
 */
func GetAvgPodCpuUsageOverTime(node string, startTime time.Time, endTime time.Time, resolution time.Duration) (model.Matrix, promv1.Warnings, error) {
	strBuilder := fmt.Sprintf("avg_over_time(sum by (pod) (irate(container_cpu_usage_seconds_total{instance = '%s', container != '', container != 'POD', pod != ''}[5m]))[%s:])", node, resolution)

	t := promv1.Range{Start: startTime, End: endTime, Step: resolution}
	result, warnings, err := QueryOverTime(strBuilder, localAPI, t)

	if err != nil {
		return nil, warnings, err
	}

	matrix := result.(model.Matrix)
	/*
		returnMap := make(map[string][]model.SamplePair)

		for _, sampleStream := range matrix {
			labelSet := model.LabelSet(sampleStream.Metric)
			pod := string(labelSet["pod"])
			returnMap[pod] = sampleStream.Values
		}


		return returnMap, warnings, nil
	*/

	return matrix, warnings, nil
}

/*
 * Authors: Jessica Barai, Erik Wahlberger
 * Calculates the average RAM usage (in bytes) over the specified resolution duration, and returns the average values between startTime and endTime
 */
func GetAvgPodMemUsageOverTime(node string, startTime time.Time, endTime time.Time, resolution time.Duration) (model.Matrix, promv1.Warnings, error) {
	strBuilder := fmt.Sprintf("avg_over_time(sum by (pod) (container_memory_usage_bytes{instance = '%s', container != '', container != 'POD', pod != ''})[%s:])", node, resolution)

	t := promv1.Range{Start: startTime, End: endTime, Step: resolution}
	result, warnings, err := QueryOverTime(strBuilder, localAPI, t)

	if err != nil {
		return nil, warnings, err
	}

	matrix := result.(model.Matrix)

	/*
		returnMap := make(map[string][]model.SamplePair)

		for _, sampleStream := range matrix {
			labelSet := model.LabelSet(sampleStream.Metric)
			pod := string(labelSet["pod"])
			returnMap[pod] = sampleStream.Values
		}

		return returnMap, warnings, nil
	*/

	return matrix, warnings, nil
}

func matrixToVectorMap(matrix model.Matrix) (map[time.Time]model.Vector, error) {
	vectorMap := make(map[time.Time]model.Vector)

	for _, sampleStream := range matrix {
		//fmt.Println(sampleStream.Metric)

		for _, samplePair := range sampleStream.Values {
			vector, ok := vectorMap[samplePair.Timestamp.Time()]

			if !ok {
				vector = []*model.Sample{}
			}

			sample := &model.Sample{
				Metric:    sampleStream.Metric,
				Timestamp: samplePair.Timestamp,
				Value:     samplePair.Value,
			}

			vector = append(vector, sample)
			vectorMap[samplePair.Timestamp.Time()] = vector
		}
	}

	return vectorMap, nil
}

func GetAvgPodResourceUsageOverTime(node string, startTime time.Time, endTime time.Time, resolution time.Duration) ([]ResourceUsageSample, promv1.Warnings, error) {
	//TODO: Is startTime before endTime?
	duration := endTime.Sub(startTime)

	if duration >= resolution {
		endTime = endTime.Add(-resolution)
	}

	cpuUsages, cpuWarnings, cpuErrors := GetAvgPodCpuUsageOverTime(node, startTime, endTime, resolution)
	memUsages, memWarnings, memErrors := GetAvgPodMemUsageOverTime(node, startTime, endTime, resolution)

	var warnings []string

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

	//fmt.Println("Cpu query:")
	cpuUsageVectorMap, err := matrixToVectorMap(cpuUsages)

	if err != nil {
		return nil, warnings, err
	}

	//fmt.Println("Mem query:")
	memUsageVectorMap, err := matrixToVectorMap(memUsages)

	if err != nil {
		return nil, warnings, err
	}

	podsResourceUsages := []ResourceUsageSample{}

	for time, cpuVector := range cpuUsageVectorMap {
		memVector, ok := memUsageVectorMap[time]

		if !ok {
			fmt.Printf("Warning: Cannot find memory usages for the time stamp %s\n", time)
			continue
		}

		resourceUsages, err := getCombinedResourceUsage(cpuVector, memVector)

		if err != nil {
			return nil, warnings, err
		}

		podsInstantaneousUsage := ResourceUsageSample{
			Time:           time,
			ResourceUsages: resourceUsages,
		}

		podsResourceUsages = append(podsResourceUsages, podsInstantaneousUsage)
	}

	return podsResourceUsages, warnings, nil
}

// Gets available CPU capacity
func GetCPUNodeCapacity(node string) (float64, promv1.Warnings, error) {
	strBuilder := fmt.Sprintf("kube_node_status_capacity{resource='cpu', exported_node='%s'}", node)
	result, warnings, err := Query(strBuilder, localAPI, time.Now())
	vector := result.(model.Vector)
	sample := vector[0]
	valueField := sample.Value
	value := float64(valueField)
	return value, warnings, err
}

func GetMemoryNodeCapacity(node string) (float64, promv1.Warnings, error) {
	strBuilder := fmt.Sprintf("kube_node_status_capacity{resource='memory', exported_node='%s'}", node)
	result, warnings, err := Query(strBuilder, localAPI, time.Now())
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
	result, warnings, err := Query(resourceUsageQuery, localAPI, time.Now())
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
func GetPodsCPUUsage(node string, startTime time.Time, endTime time.Time) (model.Vector, promv1.Warnings, error) {
	//TODO: Check that endTime is after startTime, and that startTime is before time.Now()
	duration := endTime.Sub(startTime)

	resourceUsageQuery := fmt.Sprintf("avg_over_time(sum by (pod) (irate(container_cpu_usage_seconds_total{instance = '%s', container != '', container != 'POD', pod != ''}[5m]))[%s:])", node, duration)
	result, warnings, err := Query(resourceUsageQuery, localAPI, endTime)

	if warnings != nil {
		fmt.Println("Warnings when querying pod CPU usage: ", warnings)
	}

	if err != nil {
		fmt.Println("Error when querying pod CPU usage:", err)
		return nil, warnings, err
	}

	vector, ok := result.(model.Vector)

	if !ok {
		return nil, nil, fmt.Errorf("Pods CPU usage query did not return a Vector.")
	}

	if len(vector) == 0 {
		return nil, nil, fmt.Errorf("Pods CPU usage query returned empty result.")
	}

	return vector, warnings, err
}

/*
 * Author: Erik Wahlberger
 * Retrieves a map of pod-RAM usage key-value pairs. RAM usage is given in bytes being used by each respective pod
 */
func GetPodsMemoryUsage(node string, startTime time.Time, endTime time.Time) (model.Vector, promv1.Warnings, error) {
	//TODO: Check that endTime is after startTime, and that startTime is before time.Now()
	duration := endTime.Sub(startTime)

	resourceUsageQuery := fmt.Sprintf("avg_over_time(sum by (pod) (container_memory_usage_bytes{instance='%s', container != '', container != 'POD', pod != ''})[%s:])", node, duration)
	result, warnings, err := Query(resourceUsageQuery, localAPI, endTime)

	if err != nil {
		return nil, warnings, err
	}

	vector, ok := result.(model.Vector)

	if !ok {
		return nil, nil, fmt.Errorf("Pods memory usage query did not return a Vector.")
	}

	if len(vector) == 0 {
		return nil, nil, fmt.Errorf("Pods Memory usage query returned empty result.")
	}

	return vector, warnings, nil
}

func vectorToPodMap(vector model.Vector) map[string]float64 {
	usageMap := make(map[string]float64)

	for _, sample := range vector {
		labelSet := model.LabelSet(sample.Metric)
		pod := string(labelSet["pod"])
		usageMap[pod] = float64(sample.Value)
	}

	return usageMap
}

func getCombinedResourceUsage(cpuVector model.Vector, memVector model.Vector) (map[string]ResourceUsage, error) {
	resources := make(map[string]ResourceUsage)
	cpuUsages := vectorToPodMap(cpuVector)
	memUsages := vectorToPodMap(memVector)

	for pod, cpuValue := range cpuUsages {
		memValue, ok := memUsages[pod]

		if !ok {
			fmt.Printf("Warning: The pod %s does not have any memory usage\n", pod)
			continue
		}

		resources[pod] = ResourceUsage{
			CpuUsage: cpuValue,
			MemUsage: memValue,
		}
	}

	return resources, nil
}

func GetPodsResourceUsage(node string, startTime time.Time, endTime time.Time) (ResourceUsageSample, promv1.Warnings, error) {
	cpuUsages, cpuWarnings, cpuErrors := GetPodsCPUUsage(node, startTime, endTime)
	memUsages, memWarnings, memErrors := GetPodsMemoryUsage(node, startTime, endTime)

	var warnings []string

	if cpuErrors != nil {
		return ResourceUsageSample{}, cpuWarnings, cpuErrors
	}

	if memErrors != nil {
		return ResourceUsageSample{}, memWarnings, memErrors
	}

	if cpuWarnings != nil {
		warnings = append(warnings, cpuWarnings...)
	}

	if memWarnings != nil {
		warnings = append(warnings, memWarnings...)
	}

	resourceUsages, err := getCombinedResourceUsage(cpuUsages, memUsages)

	podsResourceUsage := ResourceUsageSample{
		Time:           cpuUsages[0].Timestamp.Time(),
		ResourceUsages: resourceUsages,
	}

	return podsResourceUsage, warnings, err
}

/*
 *Returns map with replicaset as key and deployment it belong to as value
 *
 */
func getReplicasetToDeployment(duration time.Duration) (map[string]string, promv1.Warnings, error) {
	//creat query that gets all pods in cluster

	result, warnings, err := Query(fmt.Sprintf("count_over_time(kube_replicaset_owner{owner_kind='Deployment'}[%s])", duration), localAPI, time.Now())
	vector, ok := result.(model.Vector)

	if !ok {
		return nil, nil, fmt.Errorf("Replicaset owner usage query did not return a Vector.")
	}

	if len(vector) == 0 {
		return nil, nil, fmt.Errorf("Replicaset owner query returned empty result.")
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
func getPodsToReplicaset(duration time.Duration) (map[string]string, promv1.Warnings, error) {
	//creat query that gets all pods in cluster

	result, warnings, err := Query(fmt.Sprintf("count_over_time(kube_pod_owner{owner_kind='ReplicaSet'}[%s])", duration), localAPI, time.Now())

	vector, ok := result.(model.Vector)

	if !ok {
		return nil, nil, fmt.Errorf("Pod owner query did not return a Vector.")
	}

	if len(vector) == 0 {
		return nil, nil, fmt.Errorf("Pod owner query returned empty result.")
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
func GetPodsToDeployment(duration time.Duration) map[string]string {
	repTodep := make(map[string]string)
	podsToRep := make(map[string]string)
	resultMap := make(map[string]string)
	podsToRep, warnings, err := getPodsToReplicaset(duration)
	if warnings != nil {
		println("kube_pod_owner warning")
	}
	if err != nil {
		println("kube_pod_owner error")
	}
	repTodep, warnings, err = getReplicasetToDeployment(duration)
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
func GetPodsOfNode(node string, duration time.Duration) ([]string, promv1.Warnings, error) {
	strBuilder := fmt.Sprintf("count_over_time(kube_pod_info{exported_node='%s'}[%s])", node, duration)
	result, warnings, err := Query(strBuilder, localAPI, time.Now())

	vector, ok := result.(model.Vector)

	if !ok {
		return nil, nil, fmt.Errorf("Pods of node query did not return a Vector.")
	}

	if len(vector) == 0 {
		return nil, nil, fmt.Errorf("Pods of node query returned empty result.")
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
