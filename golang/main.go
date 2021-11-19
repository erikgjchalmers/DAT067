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
}

func printVector(v model.Vector) {
	for _, sample := range v {
		fmt.Printf("Time stamp: %v, metric: %v, value: %v\n", sample.Timestamp, sample.Metric, sample.Value)
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
