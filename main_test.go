package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateCost(t *testing.T) {
	assert.Equal(t, 100.0, calculateCost(1, 1, .1, .1, 100, 1))
}
