package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

func main() {
	/*
	 * The following code queries Prometheus on localhost using the simple "up" query.
	 */
	address := "http://localhost:9090"
	query := "up"

	client, err := api.NewClient(api.Config{
		Address: address,
	})

	if err != nil {
		fmt.Printf("An error occured when creating the client: %v\n", err)
		os.Exit(1)
	}

	api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, warnings, err := api.Query(ctx, query, time.Now())

	if err != nil {
		fmt.Printf("An error occured when querying Prometheus: %v\n", err)
		os.Exit(1)
	}

	if len(warnings) > 0 {
		fmt.Printf("Warnings during query: %v\n", warnings)
	}

	fmt.Printf("Data from Prometheus:\n\n")

	switch result.Type() {
	case model.ValVector:
		vector := result.(model.Vector)
		printVector(vector)
		break
	case model.ValMatrix:
		matrix := result.(model.Matrix)
		printMatrix(matrix)
		break
	case model.ValScalar:
		scalar := result.(*model.Scalar)
		printScalar(*scalar)
		break
	case model.ValString:
		str := result.(*model.String)
		printString(*str)
		break
	case model.ValNone:
		fmt.Printf("Error: No compatible value type defined for the query result: %v\n", result)
		os.Exit(1)
	}

	/*
	 * Creates an Azure retail price API and queries it for resources with the specified filters.
	 * Returns a QueryResponse type consisting of the response from the API.
	 */
	fmt.Printf("Data from Azure retails API:\n\n")

	azureApi := NewApi()
	response, err := azureApi.Query(QueryFilter{
		armSkuName:    "Standard_D2as_v4",
		armRegionName: "westeurope",
		currencyCode:  SEK,
		priceType:     "consumption",
	})

	if err != nil {
		fmt.Printf("An error occured while querying the Azure retail price API: %v\n", err)
		os.Exit(1)
	}

	printAzurePrices(response)

	fmt.Printf("Data from Kubernetes API:\n\n")

	clientSet, err := createClientSet()

	if err != nil {
		fmt.Printf("An error occured while creating the Kubernetes clientSet: %v", err)
		os.Exit(1)
	}

	nodes, err := getNodes(clientSet)

	if err != nil {
		fmt.Printf("An error occured while retrieving the nodes from Kubernetes: %v", err)
		os.Exit(1)
	}

	for _, node := range nodes {
		fmt.Printf("Node name: %s\n", node.Name)
		labels := node.Labels

		for key, value := range labels {
			fmt.Printf("\tLabel name: %s, value: %s\n", key, value)
		}

		fmt.Printf("\n")
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

func printAzurePrices(r QueryResponse) {
	fmt.Printf("Currency: %s, customer entity id: %s, customer entity type: %s\n", r.BillingCurrency, r.CustomerEntityId, r.CustomerEntityType)

	for _, item := range r.Items {
		fmt.Printf("\tarmRegionName: %s\n", item.ArmRegionName)
		fmt.Printf("\tarmSkuName: %s\n", item.ArmSkuName)
		fmt.Printf("\tcurrencyCode: %s\n", item.CurrencyCode)
		fmt.Printf("\teffectiveStartDate: %s\n", item.EffectiveStartDate)
		fmt.Printf("\tisPrimaryMeterRegion: %v\n", item.IsPrimaryMeterRegion)
		fmt.Printf("\ttype: %s\n", item.ItemType)
		fmt.Printf("\tlocation: %s\n", item.Location)
		fmt.Printf("\tmeterId: %s\n", item.MeterId)
		fmt.Printf("\tmeterName: %s\n", item.MeterName)
		fmt.Printf("\tproductId: %s\n", item.ProductId)
		fmt.Printf("\tproductName: %s\n", item.ProductName)
		fmt.Printf("\tretailPrice: %f\n", item.RetailPrice)
		fmt.Printf("\tserviceFamily: %s\n", item.ServiceFamily)
		fmt.Printf("\tserviceId: %s\n", item.ServiceId)
		fmt.Printf("\tserviceName: %s\n", item.ServiceName)
		fmt.Printf("\tskuId: %s\n", item.SkuId)
		fmt.Printf("\tskuName: %s\n", item.SkuName)
		fmt.Printf("\ttierMinimumUnits: %d\n", item.TierMinimumUnits)
		fmt.Printf("\tunitOfMeasure: %s\n", item.UnitOfMeasure)
		fmt.Printf("\tunitPrice: %f\n", item.UnitPrice)
		fmt.Printf("\n")
	}

	fmt.Printf("\n")
}
