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

type prometheusInterface interface {
	GetCPUNodeCapacity(node string) (float64, promv1.Warnings, error)
}
