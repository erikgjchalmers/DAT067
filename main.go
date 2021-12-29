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

	address := "http://localhost:9090"
	prometheus.CreateAPI(address)

	//Testing stuff related to getDeploymentPrice
	/*
		testEndTime := time.Now()
		testDuration := 24 * time.Hour
		testStartTime := testEndTime.Add(-testDuration)
		testResult, _, _ := prometheus.GetAvgPodResourceUsageOverTime("aks-default-15038067-vmss000000", testStartTime, testEndTime, testDuration)
		nodeCPUTest, _, _ := prometheus.GetCPUNodeCapacity("aks-default-15038067-vmss000000", testStartTime)
		nodeRAMTest, _, _ := prometheus.GetMemoryNodeCapacity("aks-default-15038067-vmss000000", testStartTime)
		deploymentTest, _, pricesOfPods := getDeploymentPrice(testStartTime, testEndTime, testDuration)
		podToDeploymentTest := prometheus.GetPodsToDeployment(testEndTime, testDuration)
		time.Sleep(1 * time.Second)
		nodeCPUTest2, _, _ := prometheus.GetCPUNodeCapacity("aks-default-15038067-vmss000000", testStartTime)
		nodeRAMTest2, _, _ := prometheus.GetMemoryNodeCapacity("aks-default-15038067-vmss000000", testStartTime)
		testResultTwo, _, _ := prometheus.GetAvgPodResourceUsageOverTime("aks-default-15038067-vmss000000", testStartTime, testEndTime, testDuration)
		deploymentTestTwo, _, pricesOfPods2 := getDeploymentPrice(testStartTime, testEndTime, testDuration)
		podToDeploymentTestTwo := prometheus.GetPodsToDeployment(testEndTime, testDuration)

		fmt.Printf("\n Nodes: \n First{%f, %f} Second: {%f, %f}\n", nodeCPUTest, nodeRAMTest, nodeCPUTest2, nodeRAMTest2)

		fmt.Printf("Amount of time points: First: {%d}, Second {%d}\n", len(testResult), len(testResultTwo))
		for i, pod := range testResult {
			pod2 := testResultTwo[i]
			fmt.Printf("Length of podsUsages: First: {%d}, Second {%d}\n", len(pod.ResourceUsages), len(pod2.ResourceUsages))
			for k, v := range pod.ResourceUsages {
				fmt.Printf("First{%f, %f} Second: {%f, %f}\n", v.CpuUsage, v.MemUsage, pod2.ResourceUsages[k].CpuUsage, pod2.ResourceUsages[k].MemUsage)
			}

		}

		fmt.Printf("Length of deployment: First: {%d}, Second {%d}\n", len(deploymentTest), len(deploymentTestTwo))
		for k, v := range deploymentTest {
			pod2 := deploymentTestTwo[k]
			fmt.Printf("First{%f} Second: {%f}\n", v, pod2)
		}

		fmt.Printf("Length of podToDeploments: First: {%d}, Second {%d}\n", len(podToDeploymentTest), len(podToDeploymentTestTwo))
		for k, v := range podToDeploymentTest {
			pod2 := podToDeploymentTestTwo[k]
			fmt.Printf("First{%s} Second: {%s}\n", v, pod2)
		}

		fmt.Printf("Length of podPrices: First: {%d}, Second {%d}\n", len(pricesOfPods), len(pricesOfPods2))
		for k, v := range pricesOfPods {
			pod2 := pricesOfPods2[k]
			fmt.Printf("First{%f} Second: {%f}\n", v, pod2)
		}
		//End of things testing getDeploymentPrice
	*/
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
	wantedDeployment := c.Param("deployment")
	//endTime := time.Now()
	//startTime := endTime.Add(-time.Hour)
	// in postman URL: http://localhost:8080/price/coredns-autoscaler?startTime=2021-12-24T00:00:00.371Z&endTime=2021-12-25T00:00:00.371Z
	endTimeStr := c.Query("endTime")
	fmt.Print("End")
	fmt.Print(endTimeStr)
	startTimeStr := c.Query("startTime")
	fmt.Print(startTimeStr)
	layout := "2006-01-02T15:04:05.000Z"
	fmt.Print("Start")
	fmt.Print(startTimeStr)
	fmt.Print("End")
	fmt.Print(endTimeStr)
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

	pricedMap, err := getDeploymentPrice(startTime, endTime, endTime.Sub(startTime))
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
	for i, s := range priceArray {
		if s.DeploymentName == wantedDeployment {
			c.JSON(http.StatusOK, priceArray[i])
			return
		}

		//couldn't find deployment-name add message
	}
	c.JSON(http.StatusNotFound, "deployment not found")

}

func getDeploymentPrice(startTime time.Time, endTime time.Time, resolution time.Duration) (map[string]float64, error) {
	/*
	 * The following code queries Prometheus on localhost using the simple "up" query.
	 */

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
	/*
		var resolution time.Duration = 0

		resolution = 1 * time.Hour
	*/
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
			t := podsResourceUsage.Time
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
		}
	}

	deploymentMap := prometheus.GetPodsToDeployment(endTime, duration)

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
