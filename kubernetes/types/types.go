package types

import (
	priceTypes "dat067/costestimation/pricing/types"
	"k8s.io/api/core/v1"
)

type PricedNode struct {
	Node  v1.Node
	Price priceTypes.CloudProviderPrice
}
