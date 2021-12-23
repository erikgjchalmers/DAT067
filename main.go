package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"dat067/costestimation/kubernetes"
	"dat067/costestimation/kubernetes/azure"
	"dat067/costestimation/models"

	//"dat067/costestimation/models"
	"dat067/costestimation/prometheus"
	"net/http"

	officialkube "k8s.io/client-go/kubernetes"

	//http
	"github.com/gin-gonic/gin"
	//model shouldn't be needed after test printing functionality removed.
	"github.com/prometheus/common/model"
)

var clientSet *officialkube.Clientset
var pricedNodes []kubernetes.PricedNode

type ResponseItem struct {
	Price          float64 `json:"price"`
	DeploymentName string  `json:"deployment"`
}

func main() {

	router := gin.Default()
	router.GET("/price", getDeploymentPrices)

	//endTime := time.Now()
	//startTime := endTime.Add(-time.Hour)

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

	router.Run()

	/*price, err := getDeploymentPrice("prometheus-server", startTime, endTime)

	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	fmt.Println()
	fmt.Println()
	fmt.Println()
	fmt.Println("The price of prometheus-server during the last 24 hours is", price)

	//price, err = getDeploymentPrice("Does not exist", startTime, endTime)

	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}*/
}

func getDeploymentPrices(c *gin.Context) {
	endTime := time.Now()
	startTime := endTime.Add(-time.Hour)
	pricedMap, err := getDeploymentPrice(startTime, endTime)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
	priceArray := make([]ResponseItem, len(pricedMap))
	index := 0
	for deployment, price := range pricedMap {

		priceInfoStruct := ResponseItem{
			Price:          price,
			DeploymentName: deployment,
		}
		priceArray[index] = priceInfoStruct
		index++

	}
	c.JSON(http.StatusOK, priceArray)

}

func getDeploymentPrice(startTime time.Time, endTime time.Time) (map[string]float64, error) {
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

	resolution = 5 * time.Minute

	/*
		if resolution == 0 {
			resolution = duration
		}
	*/

	//Get nodes.
	//Get nodes cost.
	//pricedNodes, err := azure.GetPricedAzureNodes(clientSet)

	//Get pods
	//Get pod resources

	podPrices := make(map[string]float64)

	for _, node := range pricedNodes {
		podsResourceUsages, warnings, err := prometheus.GetAvgPodResourceUsageOverTime(node.Node.Name, startTime, endTime, resolution)

		if warnings != nil {
			fmt.Println(warnings)
		}

		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}

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

			cpuUsage := 0.0
			memUsage := 0.0

			for pod, resourceUsage := range pods {
				fmt.Println(pod)
				monster[index] = []float64{resourceUsage.MemUsage, resourceUsage.CpuUsage}
				cpuUsage += resourceUsage.CpuUsage
				memUsage += resourceUsage.MemUsage
				index += 1
			}

			//TODO: We get all the pods on a node, even those not belonging to a deployment.
			//Calculate pods' cost
			nodeMem, _, _ := prometheus.GetMemoryNodeCapacity(node.Node.Name)
			nodeCPU, _, _ := prometheus.GetCPUNodeCapacity(node.Node.Name)
			costCalculator := models.GoodModel{Balance: []float64{1, 1}}
			price, wastedCost := costCalculator.CalculateCost(
				[]float64{
					nodeMem,
					nodeCPU},
				monster,
				node.Price, resolution.Hours())
			index = 0
			totalPodPrice := 0.0
			for pod := range pods {
				if wastedCost[index] < 0 {
					fmt.Printf("The wasted cost for pod %s is %f\n", pod, wastedCost[index])
				}

				totalPodPrice += price[index]
				podPrices[pod] += price[index]
				index += 1
			}

			if cpuUsage > nodeCPU {
				fmt.Printf("Cpu usage too high for pods on node %s\n", node.Node.Name)
			}

			if memUsage > nodeMem {
				fmt.Printf("Mem usage too high for pods on node %s\n", node.Node.Name)
			}

			if math.Abs(totalPodPrice-resolution.Hours()*node.Price) > 1e-10 {
				fmt.Printf("The sum of the pod prices is %f. The node price is %f\n", totalPodPrice, resolution.Hours()*node.Price)
			}
		}
	}

	deploymentMap := prometheus.GetPodsToDeployment(duration)

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
	//Print all the deployment costs.
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
		priceDiff := math.Abs(sumNode - sumPrice)


		for pod, price := podPrices {
			if math.Abs(priceDiff - price) <= 1e-10 {
				fmt.Printf("Pod %s is the culprit\n", pod)
			}
		}
	*/
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

	return priceMap, nil
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
