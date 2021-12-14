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

func GetDeploymentToNode() (map[string]string, promv1.Warnings, error) {
	//creat query that gets all pods in cluster

	result, warnings, err := Query("kube_deployment_labels", localAPI)

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
		node := string(labelSet["kubernetes_node"])
		deployment := string(labelSet["deployment"])

		//fmt.Printf("Pod %s is running on node %s and is currently using %.6f cores of CPU.\n", pod, node, float64(sample.Value))
		resultMap[deployment] = node
	}

	return resultMap, warnings, err

}

func GetPodsOfNode(node string) ([]string, promv1.Warnings, error) {
	//strBuilder := fmt.Sprintf("kube_node_status_capacity{resource='cpu', node='%s'}", node)
	//result, warnings, err := Query(strBuilder, localAPI)
	//creat query that gets all pods in cluster
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

func GroupByDeployment() map[string]string {
	var podByDeployment map[string]string
	//var podByReplica map[string]string
	//var replicaByDeployment map[string]string
	podOwnerResult, podOwnerWarnings, podOwnerErr := Query("kube_pod_owner{owner_kind='ReplicaSet'}", localAPI)
	if podOwnerWarnings != nil {
		println("kube_pod_owner warning")
	}
	if podOwnerErr != nil {
		println("kube_pod_owner error")
	}
	/*ReplicaOwnerResult, ReplicaOwnerWarnings, ReplicaOwnerErr := Query("kube_replicaset_owner{owner_kinde='Deployment'}", localAPI)
	if ReplicaOwnerWarnings != nil {
		println("kube_replica_owner warning")
	}
	if ReplicaOwnerErr != nil {
		println("kube_replica_owner error")*/

	vector := podOwnerResult.(model.Vector)
	for _, sample := range vector {
		labelSet := model.LabelSet(sample.Metric)

		metricName := labelSet[model.MetricNameLabel]

		if metricName != "" {
			fmt.Printf("Metric name: %s, time stamp: %s, value: %v\n", metricName, sample.Timestamp.Time(), sample.Value)
		}

		for key, value := range labelSet {
			if key == "owner_name" {
				println("WOOPDIDOO:")
				fmt.Printf("\tLabel name: %s, value: %s\n", key, value)
			}
		}

		fmt.Printf("\n")
	}

	return podByDeployment
}

type prometheusInterface interface {
	GetCPUNodeCapacity(node string) (float64, promv1.Warnings, error)
}
