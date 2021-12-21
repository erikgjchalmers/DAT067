package main

import (
	"fmt"
	"os"
	"time"

	"dat067/costestimation/kubernetes"
	"dat067/costestimation/kubernetes/azure"
	"dat067/costestimation/models"
	"dat067/costestimation/prometheus"
	officialkube "k8s.io/client-go/kubernetes"

	//model shouldn't be needed after test printing functionality removed.
	"github.com/prometheus/common/model"
)

var clientSet *officialkube.Clientset
var pricedNodes []kubernetes.PricedNode

func main() {
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	var err error
	clientSet, err = kubernetes.CreateClientSet()

	if err != nil {
		fmt.Errorf("An error occured when creating the Kubernetes client: '%v'", err)
		os.Exit(-1)
	}

	pricedNodes, err = azure.GetPricedAzureNodes(clientSet)

	if err != nil {
		fmt.Errorf("An error occured while retrieving Azure node prices: '%v'", err)
		os.Exit(-1)
	}

	price, err := getDeploymentPrice("prometheus-server", startTime, endTime)

	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println("The price of prometheus-server during the last 24 hours is", price)

	price, err = getDeploymentPrice("Does not exist", startTime, endTime)

	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func getDeploymentPrice(deploymentName string, startTime time.Time, endTime time.Time) (float64, error) {
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

	duration := endTime.Sub(startTime)
	durationHours := duration.Hours()

	var resolution time.Duration = 0

	resolution = time.Hour

	if resolution == 0 {
		resolution = duration
	}

	//Get nodes.
	//Get nodes cost.
	//pricedNodes, err := azure.GetPricedAzureNodes(clientSet)

	//Get pods
	//Get pod resources
	podPrices := make(map[string]float64)

	for _, node := range pricedNodes {
		podsResourceUsages, warnings, err := prometheus.GetAvgPodResourceUsageOverTime(node.Node.Name, startTime, endTime, resolution)
		for _, podsResourceUsage := range podsResourceUsages {
			pods := podsResourceUsage.ResourceUsages

			if err != nil {
				fmt.Printf("An error occured when querying Prometheus: %v\n", err)
				os.Exit(1)
			}

			if len(warnings) > 0 {
				fmt.Printf("Warnings during query: %v\n", warnings)
			}

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
				node.Price, resolution.Hours())
			index = 0
			for pod := range pods {
				podPrices[pod] += price[index]
				index += 1
			}
		}
	}

	fmt.Printf("podPrices has %d pods.\n", len(podPrices))

	for pod, _ := range podPrices {
		fmt.Println(pod)
	}

	fmt.Println()
	fmt.Println()
	fmt.Println()

	deploymentMap := prometheus.GetPodsToDeployment(duration)
	fmt.Printf("deploymentMap has %d pods.\n", len(deploymentMap))

	for pod, _ := range deploymentMap {
		fmt.Println(pod)
	}

	//Sum all pod costs to relevant deployment cost.
	priceMap := make(map[string]float64)

	for pod, deployment := range deploymentMap {
		price, ok := podPrices[pod]
		if !ok {
			fmt.Println("NOT OK!")
			fmt.Printf("The pod '%s' does not exist in podPrices\n", pod)
			continue
		}

		priceMap[deployment] += price
	}
	fmt.Println(len(priceMap))
	//Print all the deployment costs.
	fmt.Printf("\n")
	for d, p := range priceMap {
		fmt.Printf("%s has a cost of %f \n", d, p)
	}
	fmt.Printf("\nNode prices: \n")
	sumNode := 0.0
	for _, node := range pricedNodes {
		fmt.Printf("Node %s costs %f.\n", node.Node.Name, durationHours*node.Price)
		sumNode += durationHours * node.Price
	}
	sumPrice := 0.0
	for _, v := range podPrices {
		sumPrice += v
	}
	sumPriceMap := 0.0
	for _, v := range priceMap {
		sumPriceMap += v
	}
	fmt.Printf("Cost of nodes was %f. Total cost of pods was %f. \nThe pods being used in deployments amount to %f. \n", sumNode, sumPrice, sumPriceMap)
	/*
		fmt.Println()
		fmt.Println("Average CPU usage over time (in CPU cores):")
		endTime := time.Now()
		startTime := endTime.AddDate(0, 0, -1)
		resolution := 1 * time.Hour


		cpuUsage, warnings, err := prometheus.GetAvgCpuUsageOverTime(startTime, endTime, resolution)

		if warnings != nil {
			println("Warnings:", warnings)
		}

		if err != nil {
			fmt.Printf("An error occured while fetching the average pod CPU usages: %s\n", err)
			os.Exit(-1)
		}

		printMatrix(cpuUsage)

		fmt.Println()
		fmt.Println("Average mem usage over time (in bytes):")
		memUsage, warnings, err := prometheus.GetAvgMemUsageOverTime(startTime, endTime, resolution)

		if warnings != nil {
			println("Warnings:", warnings)
		}

		if err != nil {
			fmt.Printf("An error occured while fetching the average pod RAM usages: %s\n", err)
			os.Exit(-1)
		}

		printMatrix(memUsage)*/

	returnPrice, ok := priceMap[deploymentName]

	if !ok {
		return -1, fmt.Errorf("The name '%s' is not a valid deployment\n", deploymentName)
	}

	return returnPrice, nil
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
			fmt.Printf("\tTime stamp; %v, value; %v\n", samplePair.Timestamp.Time(), samplePair.Value)
		}
	}
}

func printScalar(s model.Scalar) {
	fmt.Printf("Time stamp: %v, value: %v\n", s.Timestamp, s.Value.String())
}

func printString(s model.String) {
	fmt.Printf("Time stamp: %v, value: %v\n", s.Timestamp, s.Value)
}
