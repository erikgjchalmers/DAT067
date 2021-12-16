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

	//Get nodes.
	//Get nodes cost.
	//pricedNodes, err := azure.GetPricedAzureNodes(clientSet)

	//Get pods
	//Get pod resources
	podPrices := make(map[string]float64)
	for _, node := range pricedNodes {
		pods, _, _ := prometheus.GetPodsResourceUsage(node.Node.Name)
		monster := make([][]float64, len(pods))
		index := 0
		for _, resourceUsage := range pods {
			monster[index] = []float64{resourceUsage.MemUsage, resourceUsage.CpuUsage}
			index += 1
		}

		//TODO: We get all the pods on a node, even those not belonging to a deployment.
		//Calculate pods' cost
		nodeMem, _, _ := prometheus.GetMemoryNodeCapacity(node.Node.Name)
		nodeCPU, _, _ := prometheus.GetCPUNodeCapacity(node.Node.Name)
		costCalculator := models.GoodModel{Balance: []float64{1, 1}}
		price, _ := costCalculator.CalculateCost(
			[]float64{
				nodeMem,
				nodeCPU},
			monster,
			node.Price, 1)
		index = 0
		for pod := range pods {
			podPrices[pod] = price[index]
			index += 1
		}
	}
	deploymentMap := prometheus.GetPodsToDeployment()
	//Sum all pod costs to relevant deployment cost.
	priceMap := make(map[string]float64)

	for pod, deployment := range deploymentMap {
		price, ok := podPrices[pod]
		if !ok {
			continue
		}
		priceMap[deployment] += price
	}
	//Print all the deployment costs.
	fmt.Printf("\n")
	for d, p := range priceMap {
		fmt.Printf("%s has a cost of %f \n", d, p)
	}

	sumNode := 0.0
	for _, node := range pricedNodes {
		sumNode += node.Price
	}
	sumPrice := 0.0
	for _, v := range podPrices {
		sumPrice += v
	}
	sumPriceMap := 0.0
	for _, v := range priceMap {
		sumPriceMap += v
	}
	fmt.Printf("Cost of nodes was %f. Total cost of pods was %f. \nThe ones being used in deployments amount to %f. \n", sumNode, sumPrice, sumPriceMap)
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
