package models

// [price/hour] * hour
type IcostCalculator interface {
	CalculateCost(allocation, usage []float64,
		nodePrice, hours float64) float64
}

//Badmodel
type BadModel struct {
}

func (m BadModel) CalculateCost(capacity []float64, usage []float64, nodePrice float64, hours float64) float64 {
	return nodePrice * hours * (usage[0] * usage[1]) / (capacity[0] * capacity[1])
}

//Goodmodel
type GoodModel struct {
	balance []float64
}

func (m GoodModel) CalculateCost(max [:]float64, usage [:]float64, nodePrice float64, hours float64) float64 {
	return 5.0
}
