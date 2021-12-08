package models

// [price/hour] * hour
type ICostCalculator interface {
	CalculateCost(
		allocation []float64,
		usage [][]float64,
		nodePrice, hours float64) []float64
}

//Badmodel
type BadModel struct {
}

func (m BadModel) CalculateCost(capacity []float64, usage []float64, nodePrice float64, hours float64) []float64 {
	return []float64{nodePrice * hours * (usage[0] * usage[1]) / (capacity[0] * capacity[1])}
}

//Goodmodel
type GoodModel struct {
	balance []float64
}

func (m GoodModel) CalculateCost(nodeResources []float64, usagePerContainer [][]float64, nodePrice float64, hours float64) []float64 {
	wastedResources := make([]float64, len(nodeResources))
	//For each resource
	for i, _ := range nodeResources {
		totalUseOfResource := 0.0
		//For each container
		for j, _ := range usagePerContainer {
			totalUseOfResource += usagePerContainer[j][i]
		}
		wastedResources[i] = nodeResources[i] - totalUseOfResource
	}

	//Calculate base cost and apply
	return []float64{1}
}

func normalizeSlice(arr []float64) []float64 {
	var sum float64 = 0
	for _, n := range arr {
		sum += n
	}
	toReturn := make([]float64, len(arr))
	if sum == 0 {
		for i, _ := range arr {
			toReturn[i] = 1.0 / float64(len(arr))
		}
		return toReturn
	}
	for i, n := range arr {
		toReturn[i] = n / sum
	}
	return toReturn
}
