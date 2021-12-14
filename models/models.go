package models

// [price/hour] * hour
type ICostCalculator interface {
	CalculateCost(
		allocation []float64,
		usage [][]float64,
		nodePrice, hours float64) ([]float64, []float64)
}

//Badmodel
type BadModel struct {
}

func (m BadModel) CalculateCost(capacity []float64, usage []float64, nodePrice float64, hours float64) ([]float64, []float64) {
	costOfFirstContainer := []float64{nodePrice * hours * (usage[0] * usage[1]) / (capacity[0] * capacity[1])}
	waste := []float64{nodePrice * hours * (1 - (usage[0]*usage[1])/(capacity[0]*capacity[1]))}
	return costOfFirstContainer, waste
}

//Goodmodel
type GoodModel struct {
	Balance []float64
}

func (m GoodModel) CalculateCost(nodeResources []float64, usagePerContainer [][]float64, nodePrice float64, hours float64) []float64 {

	//Make sure that Balance is normalized(Is there a way to do this on model declaration?)
	if m.Balance == nil {
		m.Balance = make([]float64, len(nodeResources))
	}
	m.Balance = normalizeSlice(m.Balance)

	//Converting the usage array to percentage.
	//TODO: Currently changes the slice. Fix!
	for i := range usagePerContainer {
		for j, v := range usagePerContainer[i] {
			usagePerContainer[i][j] = v / nodeResources[j]
		}
	}

	wastedResources := make([]float64, len(nodeResources))
	totalUseOfResource := make([]float64, len(nodeResources))
	//For each resource
	for i := range nodeResources {
		//For each container
		for j := range usagePerContainer {
			totalUseOfResource[i] += usagePerContainer[j][i]
		}
		wastedResources[i] = 1 - totalUseOfResource[i]
	}
	//Maybe a check here is needed to make sure that wasted resources are not negative? In case of over 100% use of resources.

	//Generate the actual cost of wasted resources
	var wastedCost float64 = 0
	for i, v := range wastedResources {
		wastedCost += v * nodePrice * m.Balance[i]
	}
	//Generate a vector for distributing wasted resource cost
	propOfWastedCost := make([]float64, len(nodeResources))
	for i := range propOfWastedCost {
		propOfWastedCost[i] = 0
		for j, v := range wastedResources {
			if i == j {
				continue
			}
			propOfWastedCost[i] += v
		}
	}
	propOfWastedCost = normalizeSlice(propOfWastedCost)
	//Calculate costs
	costs := make([]float64, len(nodeResources))
	for i, con := range usagePerContainer {
		var sumOfCostsForContainer float64 = 0
		for j, costOfDimensionForContainer := range con {
			//The cost for the resources used and also the cost for the wasted resources.
			//TODO: Doublecheck this.
			sumOfCostsForContainer += nodePrice*m.Balance[j]*costOfDimensionForContainer + propOfWastedCost[j]*wastedCost*(con[j]/totalUseOfResource[j])
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
