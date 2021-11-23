package types

import (
	"fmt"
	"testing"
)

func TestCloudProviderPrice(t *testing.T) {
	price := CloudProviderPrice{
		Price: 12345.6789,
		Unit:  OneHour,
	}

	if price.Price != 12345.6789 {
		t.Error("Error on Price property: {} inputted, {} expected, {} received", 12345.6789, 12345.6789, price.Price)
	}

	if price.Unit != OneHour {
		t.Error("Error on Unit property: '{}' inputted, '{}' expected, '{}' received", OneHour.String(), OneHour.String(), price.Unit.String())
	}
}

func TestUnitString(t *testing.T) {
	tests :=
		[]struct {
			Input  Unit
			Output string
		}{
			// Test OneHour value for the Unit
			{
				Input:  OneHour,
				Output: "1 Hour",
			},
			// Test OneMinute value for the unit
			{
				Input:  OneMinute,
				Output: "1 Minute",
			},
			// Test OneSecond value for the unit
			{
				Input:  OneSecond,
				Output: "1 Second",
			},
			// Tests Unknown value for the Unit
			{
				Input:  Unknown,
				Output: "",
			},
			// Tests invalid int value for the Unit
			{
				Input:  100,
				Output: "",
			},
		}
	for _, test := range tests {
		input := test.Input
		expected := test.Output
		result := input.String()

		if expected != result {
			t.Error(fmt.Printf("Error on Unit String() function: %d input, %s expected, %s result", input, expected, result))
		}
	}
}
