package azure

import (
	"errors"
	"fmt"
	"strings"

	kubeHelper "dat067/costestimation/kubernetes"
	"dat067/costestimation/pricing"

	"k8s.io/client-go/kubernetes"
)

const LABEL_AZURE_INSTANCE_TYPE = "node.kubernetes.io/instance-type"
const LABEL_AZURE_REGION = "topology.kubernetes.io/region"

func GetPricedAzureNodes(c *kubernetes.Clientset) ([]kubeHelper.PricedNode, error) {
	nodes, err := kubeHelper.GetNodes(c)

	if err != nil {
		return nil, err
	}

	pricedNodes := make([]kubeHelper.PricedNode, 0, len(nodes))

	for _, node := range nodes {
		labels := node.Labels
		azureInstanceType := labels[LABEL_AZURE_INSTANCE_TYPE]
		azureRegion := labels[LABEL_AZURE_REGION]
		operatingSystem := labels[kubeHelper.LABEL_OPERATING_SYSTEM]

		azureApi := pricing.NewApi()
		response, err := azureApi.Query(pricing.QueryFilter{
			ArmSkuName:    azureInstanceType,
			ArmRegionName: azureRegion,
			CurrencyCode:  pricing.SEK,
			PriceType:     "consumption",
		})

		if err != nil {
			return nil, err
		}

		// Find price per unit of time for the Azure nodes in the Kubernetes cluster
		for _, item := range response.Items {
			if !strings.Contains(item.MeterName, pricing.METER_SPOT) && !strings.Contains(item.MeterName, pricing.METER_LOW_PRIORITY) {
				if strings.ToLower(operatingSystem) == strings.ToLower(kubeHelper.LABEL_OPERATING_SYSTEM_LINUX) {
					if !strings.Contains(item.ProductName, kubeHelper.LABEL_OPERATING_SYSTEM_WINDOWS) {
						if err != nil {
							return nil, errors.New(fmt.Sprintf("Invalid unit of measure returned by Azure Retail Prices API: '%s'", item.UnitOfMeasure))
						}

						pricedNodes = append(pricedNodes, kubeHelper.PricedNode{
							Node:  node,
							Price: item.UnitPrice,
						})
					}
				} else if strings.ToLower(operatingSystem) == strings.ToLower(kubeHelper.LABEL_OPERATING_SYSTEM_WINDOWS) {
					if strings.Contains(item.ProductName, kubeHelper.LABEL_OPERATING_SYSTEM_WINDOWS) {
						if err != nil {
							return nil, errors.New(fmt.Sprintf("Invalid unit of measure returned by Azure Retail Prices API: '%s'", item.UnitOfMeasure))
						}

						pricedNodes = append(pricedNodes, kubeHelper.PricedNode{
							Node:  node,
							Price: item.UnitPrice,
						})

					}
				}
			}
		}
	}

	return pricedNodes, nil
}

func PrintNodes(nodes []kubeHelper.PricedNode) {
	for _, node := range nodes {
		fmt.Printf("Node hostname: %s, node price per hour: %f\n", node.Node.Name, node.Price)
	}
}
