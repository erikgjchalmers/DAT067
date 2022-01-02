package main

import (
	"flag"
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
	address := flag.String("url", "http://localhost:9090", "Put the address here, dummy!")
	flag.Parse()

	router := gin.Default()
	//router.GET("/price", getDeploymentPrices)
	router.GET("/price/:deployment", getDeploymentPrices)

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

	prometheus.CreateAPI(*address)
	router.Run()
}

func getDeploymentPrices(c *gin.Context) {
	wantedDeployment := c.Param("deployment")
	// in postman URL: http://localhost:8080/price/coredns-autoscaler?startTime=2021-12-24T00:00:00.371Z&endTime=2021-12-25T00:00:00.371Z
	endTimeStr := c.Query("endTime")
	startTimeStr := c.Query("startTime")
	resolutionStr := c.DefaultQuery("resolution", "None")
	layout := "2006-01-02T15:04:05.000Z"

	endTime, err := time.Parse(layout, endTimeStr)
	if err != nil {
		fmt.Print("ERROR")
		fmt.Print(err.Error())
		os.Exit(-1)
	}

	startTime, err := time.Parse(layout, startTimeStr)
	if err != nil {
		fmt.Print("ERROR")
		fmt.Print(err.Error())
		os.Exit(-1)
	}
	var pricedMap map[string]float64
	//Abomination below
	if resolutionStr != "None" {
		resolution, err := time.ParseDuration(resolutionStr)
		if err != nil {
			fmt.Print("ERROR")
			fmt.Print(err.Error())
			os.Exit(-1)
		}
		pricedMap, err = getDeploymentPrice(startTime, endTime, resolution)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		}
	} else {
		pricedMap, err = getDeploymentPriceOverPeriod(startTime, endTime)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		}
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
	for i, s := range priceArray {
		if s.DeploymentName == wantedDeployment {
			c.JSON(http.StatusOK, priceArray[i])
			return
		}

		//couldn't find deployment-name add message
	}
	c.JSON(http.StatusNotFound, "deployment not found")

}
func getDeploymentPriceOverPeriod(startTime time.Time, endTime time.Time) (map[string]float64, error) {
	return getDeploymentPrice(startTime, endTime, endTime.Sub(startTime))
}

func getDeploymentPrice(startTime time.Time, endTime time.Time, resolution time.Duration) (map[string]float64, error) {

	duration := endTime.Sub(startTime)
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

			tmap := getPrice(podsResourceUsage, node, resolution)
			for k, v := range tmap {
				podPrices[k] += v
			}
		}
	}
	return groupPodPricesToDeployment(podPrices, endTime, duration), nil
}

func groupPodPricesToDeployment(podPrices map[string]float64, endTime time.Time, duration time.Duration) map[string]float64 {
	deploymentMap := prometheus.GetPodsToDeployment(endTime, duration)
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
		fmt.Printf("Node %s costs %f.\n", node.Node.Name, duration.Hours()*node.Price)
		sumNode += duration.Hours() * node.Price
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
	return priceMap
}
func getPrice(podsResourceUsage prometheus.ResourceUsageSample, node kubernetes.PricedNode, resolution time.Duration) map[string]float64 {
	podPrices := make(map[string]float64)
	pods := podsResourceUsage.ResourceUsages
	t := podsResourceUsage.Time

	monster := make([][]float64, len(pods))
	index := 0

	cpuUsage := 0.0
	memUsage := 0.0
	orderOfNames := make([]string, len(pods))
	for name, resourceUsage := range pods {
		orderOfNames[index] = name
		monster[index] = []float64{resourceUsage.MemUsage, resourceUsage.CpuUsage}
		cpuUsage += resourceUsage.CpuUsage
		memUsage += resourceUsage.MemUsage
		index += 1
	}

	//TODO: We get all the pods on a node, even those not belonging to a deployment.
	//Calculate pods' cost
	nodeMem, _, _ := prometheus.GetMemoryNodeCapacity(node.Node.Name, t)
	nodeCPU, _, _ := prometheus.GetCPUNodeCapacity(node.Node.Name, t)
	costCalculator := models.GoodModel{Balance: []float64{1, 1}}
	price, wastedCost := costCalculator.CalculateCost(
		[]float64{
			nodeMem,
			nodeCPU},
		monster,
		node.Price, resolution.Hours())
	index = 0
	totalPodPrice := 0.0
	for _, pod := range orderOfNames {
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
	return podPrices
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
