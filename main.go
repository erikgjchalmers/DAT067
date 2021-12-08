package main

import (
	"fmt"
	"os"

	"dat067/costestimation/kubernetes"
	"dat067/costestimation/kubernetes/azure"
	"dat067/costestimation/models"
	"dat067/costestimation/prometheus"

	//model shouldn't be needed after test printing functionality removed.
	"github.com/prometheus/common/model"
)

func main() {
	/*
	 * The following code queries Prometheus on localhost using the simple "up" query.
	 */
	address := "http://localhost:9090"
	//query := "up"

	prometheus.CreateAPI(address)
	memCapacity, warnings, err := prometheus.GetMemoryNodeCapacity("aks-standard1-15038067-vmss000001")
	memUsage, warnings, err := prometheus.GetMemoryNodeUsage("aks-standard1-15038067-vmss000001")
	cpuCapacity, warnings, err := prometheus.GetCPUNodeCapacity("aks-standard1-15038067-vmss000001")
	cpuUsage, warnings, err := prometheus.GetCPUNodeUsage("aks-standard1-15038067-vmss000001")
	costmodel := models.GoodModel{}
	price := costmodel.CalculateCost({memCapacity, cpuCapacity},[memUsage, cpuUsage], 10, 1)
	fmt.Printf("Your node costs %f dollars.\n", price)

	//result, warnings, err := prometheus.Query(query, api)
	//fmt.Println("WOPDIDOO:", result)

	if err != nil {
		fmt.Printf("An error occured when querying Prometheus: %v\n", err)
		os.Exit(1)
	}

	if len(warnings) > 0 {
		fmt.Printf("Warnings during query: %v\n", warnings)
	}

	fmt.Printf("Data from Prometheus:\n\n")

	/* 	switch result.Type() {
	   	case model.ValVector:
	   		vector := result.(model.Vector)
	   		printVector(vector)
	   		fmt.Printf("Valvector")
	   		break
	   	case model.ValMatrix:
	   		matrix := result.(model.Matrix)
	   		printMatrix(matrix)
	   		fmt.Printf("ValMatrix")
	   		break
	   	case model.ValScalar:
	   		scalar := result.(*model.Scalar)
	   		printScalar(*scalar)
	   		fmt.Printf("ValScalar")
	   		break
	   	case model.ValString:
	   		str := result.(*model.String)
	   		printString(*str)
	   		fmt.Printf("ValString")
	   		break
	   	case model.ValNone:
	   		fmt.Printf("Error: No compatible value type defined for the query result: %v\n", result)
	   		os.Exit(1)
	   	} */

	fmt.Printf("Data from Kubernetes API:\n\n")

	clientSet, err := kubernetes.CreateClientSet()

	if err != nil {
		fmt.Printf("An error occured when creating the Kubernetes client: '%v'", err)
	}

	pricedNodes, err := azure.GetPricedAzureNodes(clientSet)

	if err != nil {
		fmt.Printf("An error occured while retrieving Azure node prices: '%v'", err)
	}

	azure.PrintNodes(pricedNodes)
}

func printVector(v model.Vector) {
	for _, sample := range v {
		labelSet := model.LabelSet(sample.Metric)

		metricName := labelSet[model.MetricNameLabel]

		if metricName != "" {
			fmt.Printf("Metric name: %s, time stamp: %s, value: %v\n", metricName, sample.Timestamp.Time(), sample.Value)
		}

		for key, value := range labelSet {
			if key != model.MetricNameLabel {
				fmt.Printf("\tLabel name: %s, value: %s\n", key, value)
			}
		}

		fmt.Printf("\n")
	}
}

func printMatrix(m model.Matrix) {
	for _, sampleStream := range m {
		fmt.Printf("Metric: %v\n", (*sampleStream).Metric.String())
		for _, samplePair := range (*sampleStream).Values {
			fmt.Printf("\tTime stamp; %v, value; %v\n", samplePair.Timestamp, samplePair.Value)
		}
	}
}

func printScalar(s model.Scalar) {
	fmt.Printf("Time stamp: %v, value: %v\n", s.Timestamp, s.Value.String())
}

func printString(s model.String) {
	fmt.Printf("Time stamp: %v, value: %v\n", s.Timestamp, s.Value)
}
