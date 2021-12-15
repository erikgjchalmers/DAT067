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
	costmodel := models.GoodModel{Balance: []float64{2, 1}}
	price, _ := costmodel.CalculateCost([]float64{memCapacity, cpuCapacity}, [][]float64{{memUsage, cpuUsage}}, 10, 1)
	fmt.Printf("Your node costs %f dollars.\n", price[0])

	fmt.Printf("Data from Prometheus:\n\n")

	// print all pods belonging to a specific node
	resultPods := []string{}
	resultPods, warnings, err = prometheus.GetPodsOfNode("aks-standard1-15038067-vmss000001")
	fmt.Print("aks-standard1-15038067-vmss000001 hosting pods:\n\n")
	fmt.Println(resultPods)
	fmt.Print("\n\n")
	resultPods, warnings, err = prometheus.GetPodsOfNode("aks-default-15038067-vmss000000")
	fmt.Print("aks-default-15038067-vmss000000 hosting pods:\n\n")
	fmt.Println(resultPods)
	fmt.Print("\n\n")
	resultPods, warnings, err = prometheus.GetPodsOfNode("aks-standard1-15038067-vmss000000")
	fmt.Print("aks-standard1-15038067-vmss000000 hosting pods:\n\n")
	fmt.Println(resultPods)
	fmt.Print("\n\n")

	if err != nil {
		fmt.Printf("An error occured when querying Prometheus: %v\n", err)
		os.Exit(1)
	}

	if len(warnings) > 0 {
		fmt.Printf("Warnings during query: %v\n", warnings)
	}

	//print all pods and which deployment it belongs to
	resultMap := make(map[string]string)
	resultMap = prometheus.GetPodsToDeployment()
	fmt.Print("print all pods and which deployment it belongs to:\n\n")
	fmt.Print(resultMap)
	fmt.Print(len(resultMap))

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
		return
	}

	pricedNodes, err := azure.GetPricedAzureNodes(clientSet)

	if err != nil {
		fmt.Printf("An error occured while retrieving Azure node prices: '%v'", err)
		return
	}

	azure.PrintNodes(pricedNodes)
	resourceUsages, warnings, err := prometheus.GetPodsResourceUsage("aks-standard1-15038067-vmss000001")

	if err != nil {
		fmt.Printf("An error occured while retrieving pod resource usage from Prometheus: %s", err)
		return
	}

	for key, value := range resourceUsages {
		fmt.Printf("The pod '%s' is currently using %.6f CPU cores and %.1f bytes of RAM.\n", key, value.CpuUsage, value.MemUsage)
	}
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
