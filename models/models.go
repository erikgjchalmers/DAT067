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

	//Make sure that balance is normalized(Is there a way to do this on model declaration?)
	m.balance = normalizeSlice(m.balance)

	//Converting the usage array to percentage.
	for i := range usagePerContainer {
		for j, v := range usagePerContainer[i] {
			usagePerContainer[i][j] = v / nodeResources[j]
		}
	}

	wastedResources := make([]float64, len(nodeResources))
	//For each resource
	for i := range nodeResources {
		totalUseOfResource := 0.0
		//For each container
		for j := range usagePerContainer {
			totalUseOfResource += usagePerContainer[j][i]
		}
		wastedResources[i] = 1 - totalUseOfResource
	}
	println(wastedResources)

	//Calculate costs
	costs := make([]float64, len(nodeResources))
	for i := range usagePerContainer {
		var sumOfCostsForContainer float64 = 0
		for _, costOfDimensionForContainer := range usagePerContainer[i] {
			sumOfCostsForContainer += nodePrice * m.balance[i] * costOfDimensionForContainer
		}
		costs[i] = sumOfCostsForContainer
	}
	return costs
}

func normalizeSlice(arr []float64) []float64 {
	var sum float64 = 0
	for _, n := range arr {
		sum += n
	}
	toReturn := make([]float64, len(arr))
	if sum == 0 {
		for i := range arr {
			toReturn[i] = 1.0 / float64(len(arr))
		}
		return toReturn
	}
	for i, n := range arr {
		toReturn[i] = n / sum
	}
	return toReturn
}
