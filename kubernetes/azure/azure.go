package azure

import (
	"errors"
	"fmt"
	"strings"

	kubeHelper "dat067/costestimation/kubernetes"
	kubeTypes "dat067/costestimation/kubernetes/types"
	azurePrices "dat067/costestimation/pricing/azure"
	priceTypes "dat067/costestimation/pricing/types"

	"k8s.io/client-go/kubernetes"
)

const LABEL_AZURE_INSTANCE_TYPE = "node.kubernetes.io/instance-type"
const LABEL_AZURE_REGION = "topology.kubernetes.io/region"

func GetPricedAzureNodes(c *kubernetes.Clientset) ([]kubeTypes.PricedNode, error) {
	nodes, err := kubeHelper.GetNodes(c)

	if err != nil {
		return nil, err
	}

	pricedNodes := make([]kubeTypes.PricedNode, 0, len(nodes))

	for _, node := range nodes {
		labels := node.Labels
		azureInstanceType := labels[LABEL_AZURE_INSTANCE_TYPE]
		azureRegion := labels[LABEL_AZURE_REGION]
		operatingSystem := labels[kubeHelper.LABEL_OPERATING_SYSTEM]

		azureApi := azurePrices.NewApi()
		response, err := azureApi.Query(azurePrices.QueryFilter{
			ArmSkuName:    azureInstanceType,
			ArmRegionName: azureRegion,
			CurrencyCode:  azurePrices.SEK,
			PriceType:     "consumption",
		})

		if err != nil {
			return nil, err
		}

		// Find price per unit of time for the Azure nodes in the Kubernetes cluster
		for _, item := range response.Items {
			if !strings.Contains(item.MeterName, azurePrices.METER_SPOT) && !strings.Contains(item.MeterName, azurePrices.METER_LOW_PRIORITY) {
				if strings.ToLower(operatingSystem) == strings.ToLower(kubeHelper.LABEL_OPERATING_SYSTEM_LINUX) {
					if !strings.Contains(item.ProductName, kubeHelper.LABEL_OPERATING_SYSTEM_WINDOWS) {
						unit, err := azurePrices.ParseUnit(item.UnitOfMeasure)

						if err != nil {
							return nil, errors.New(fmt.Sprintf("Invalid unit of measure returned by Azure Retail Prices API: '%s'", item.UnitOfMeasure))
						}

						pricedNodes = append(pricedNodes, kubeTypes.PricedNode{
							Node: node,
							Price: priceTypes.CloudProviderPrice{
								Price: item.UnitPrice,
								Unit:  unit,
							},
						})
					}
				} else if strings.ToLower(operatingSystem) == strings.ToLower(kubeHelper.LABEL_OPERATING_SYSTEM_WINDOWS) {
					if strings.Contains(item.ProductName, kubeHelper.LABEL_OPERATING_SYSTEM_WINDOWS) {
						unit, err := azurePrices.ParseUnit(item.UnitOfMeasure)

						if err != nil {
							return nil, errors.New(fmt.Sprintf("Invalid unit of measure returned by Azure Retail Prices API: '%s'", item.UnitOfMeasure))
						}

						pricedNodes = append(pricedNodes, kubeTypes.PricedNode{
							Node: node,
							Price: priceTypes.CloudProviderPrice{
								Price: item.UnitPrice,
								Unit:  unit,
							},
						})

					}
				}
			}
		}
	}

	return pricedNodes, nil
}

func PrintNodes(nodes []kubeTypes.PricedNode) {
	for _, node := range nodes {
		fmt.Printf("Node hostname: %s, node price: %f, node price unit of measure: %s\n", node.Node.Name, node.Price.Price, node.Price.Unit.String())
	}
}
