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
	return getResourceUsageQuery("cpu", node)
}

func GetMemoryNodeUsage(node string) (float64, promv1.Warnings, error) {
	return getResourceUsageQuery("memory", node)
}

func getResourceUsageQuery(resource string, node string) (float64, promv1.Warnings, error) {
	resourceUsageQuery := fmt.Sprintf("kube_node_status_capacity{resource='%s', node='%s'} - avg_over_time(kube_node_status_allocatable{resource='%s', node='%s'}[1h])", resource, node, resource, node)
	result, warnings, err := Query(resourceUsageQuery, localAPI)
	vector := result.(model.Vector)
	sample := vector[0]
	valueField := sample.Value
	value := float64(valueField)
	return value, warnings, err
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

		var deployment = repTodep[element]
		resultMap[key] = deployment

	}
	return resultMap

}

/*
*Returns a string slice with all pods in a specific node. Take in node as argument
*
 */
func GetPodsOfNode(node string) ([]string, promv1.Warnings, error) {
	strBuilder := fmt.Sprintf("kube_pod_info{node='%s'}", node)
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
