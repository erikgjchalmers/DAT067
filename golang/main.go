package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
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

	fmt.Printf("Result: \n%v\n", result)
}
