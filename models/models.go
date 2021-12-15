package models

//@Author Erik Gjers
//Allows calculating cost based on use.
type ICostCalculator interface {
	CalculateCost(
		allocation []float64,
		usage [][]float64,
		nodePrice, hours float64) ([]float64, []float64)
}

//A way to enforce the use of interface. Will give an error when compiling if interface is not implemented on the models.
var _ ICostCalculator = (*BadModel)(nil)
var _ ICostCalculator = (*GoodModel)(nil)

//@Author Erik Gjers
//A model that only takes the first container and considers the rest of the node wasted. Doesn't add the wasted cost.
type BadModel struct {
}

//@Author Erik Gjers
func (m BadModel) CalculateCost(capacity []float64, usage [][]float64, nodePrice float64, hours float64) ([]float64, []float64) {
	costOfFirstContainer := []float64{nodePrice * hours * (usage[0][0] * usage[0][1]) / (capacity[0] * capacity[1])}
	waste := []float64{nodePrice * hours * (1 - (usage[0][0]*usage[0][1])/(capacity[0]*capacity[1]))}
	return costOfFirstContainer, waste
}

//@Author Erik Gjers
//A model that calculates cost based off of several factors: Waste on the node, a balance set between the various dimensions allowing different values for the dimensions.
//Works for any number of dimensions, but will be used for 2 dimensions mostly - CPU and RAM.
type GoodModel struct {
	Balance []float64
}

//@Author Erik Gjers
func (m GoodModel) CalculateCost(nodeResources []float64, usagePerContainer [][]float64, nodePrice float64, hours float64) ([]float64, []float64) {

	//Make sure that Balance is normalized(Is there a way to do this on model declaration?)
	if m.Balance == nil {
		m.Balance = make([]float64, len(nodeResources))
	}
	m.Balance = normalizeSlice(m.Balance)

	//Making a new array to store the data.
	percentUsePerContainer := make([][]float64, len(usagePerContainer))
	for i := range percentUsePerContainer {
		percentUsePerContainer[i] = make([]float64, len(nodeResources))
	}

	//Converting the usage array to percentage and storing in new array.
	for i := range usagePerContainer {
		for j, v := range usagePerContainer[i] {
			percentUsePerContainer[i][j] = v / nodeResources[j]
		}
	}

	wastedResources := make([]float64, len(nodeResources))
	totalUseOfResource := make([]float64, len(nodeResources))
	//For each resource
	for i := range nodeResources {
		//For each container
		for j := range percentUsePerContainer {
			totalUseOfResource[i] += percentUsePerContainer[j][i]
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
	wasteCosts := make([]float64, len(nodeResources))
	for i, con := range percentUsePerContainer {
		var sumOfBaseCostForContainer float64 = 0
		var sumOfWasteForContainer float64 = 0
		for j, costOfDimensionForContainer := range con {
			//The cost for the resources used and also the cost for the wasted resources.
			sumOfBaseCostForContainer += nodePrice * m.Balance[j] * costOfDimensionForContainer
			sumOfWasteForContainer += propOfWastedCost[j] * wastedCost * (con[j] / totalUseOfResource[j])
		}
		costs[i] = (sumOfBaseCostForContainer + sumOfWasteForContainer) * hours
		wasteCosts[i] = sumOfWasteForContainer * hours
	}
	return costs, wasteCosts
}

//@Author Erik Gjers
//Causes a slice to normalize, aka sum to 1.
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
