package models

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

const epsilon float64 = 0.001

func TestImportantFunction(t *testing.T) {
	//assert = assert.New(t)
	assert.InDelta(t, 3, 3, epsilon)
}

func TestNormalizeSlice(t *testing.T) {
	slice := []float64{1}
	slice = normalizeSlice(slice)
	assert.InDelta(t, 1, slice[0], epsilon)

	slice = []float64{2, 2}
	slice = normalizeSlice(slice)
	assert.InDelta(t, 0.5, slice[0], epsilon)

	slice = []float64{9, 1}
	slice = normalizeSlice(slice)
	assert.InDelta(t, 0.1, slice[1], epsilon)
	assert.InDelta(t, 0.9, slice[0], epsilon)

	slice = []float64{rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64()}
	slice = normalizeSlice(slice)
	sum := 0.0
	for _, n := range slice {
		sum += n
	}
	assert.InDelta(t, 1, sum, epsilon)
}
func TestGoodModel(t *testing.T) {
	m := GoodModel{[]float64{1}}
	var prices []float64 = m.CalculateCost([]float64{1}, [][]float64{{1}}, 100, 1)
	assert.InDelta(t, 100, prices[0], epsilon)

	prices = m.CalculateCost([]float64{1}, [][]float64{{.5}}, 100, 1)
	assert.InDelta(t, 100, prices[0], epsilon)

	m.Balance = []float64{1, 1}
	prices = m.CalculateCost([]float64{100, 100}, [][]float64{{25, 50}, {50, 50}}, 100, 1)
	var sum float64 = 0
	for _, v := range prices {
		sum += v
	}
	assert.InDelta(t, 100, sum, epsilon)
	assert.InDelta(t, 43.75, prices[0], epsilon)
	assert.InDelta(t, 56.25, prices[1], epsilon)

	var randomPrice = rand.Float64()
	//prices = m.CalculateCost([]float64{100, 100}, [][]float64{{rand.Float64() * 50, rand.Float64() * 50}, {rand.Float64() * 50, rand.Float64() * 50}}, randomPrice, 1)
	m.Balance = []float64{3, 1}
	prices = m.CalculateCost([]float64{100, 100}, [][]float64{{25, 50}, {50, 50}}, randomPrice, 1)
	sum = 0
	for _, v := range prices {
		sum += v
	}
	assert.InDelta(t, randomPrice, sum, epsilon)

	m.Balance = []float64{1, 1, 1}
	prices = m.CalculateCost([]float64{100, 100, 100}, [][]float64{{25, 50, 50}, {50, 50, 50}, {10, 0, 0}}, 100, 1)
	sum = 0
	for _, v := range prices {
		sum += v
	}
	assert.InDelta(t, 100, sum, epsilon)

	m.Balance = []float64{8, 1, 1}
	prices = m.CalculateCost([]float64{100, 100, 100}, [][]float64{{25, 50, 50}, {50, 50, 50}, {10, 0, 0}}, 100, 1)
	sum = 0
	for _, v := range prices {
		sum += v
	}
	assert.InDelta(t, 100, sum, epsilon)
	assert.InDelta(t, 36, prices[0], epsilon)
	assert.InDelta(t, 56, prices[1], epsilon)
}
