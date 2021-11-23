// Package containing data types for cloud provider pricing information
package types

import ()

// Author: Erik Wahlberger
// Type to represent the unit of measure for prices
type Unit uint

// Author: Erik Wahlberger
// Enum for Unit type, containing the possible values to choose from (enum)
const (
	Unknown Unit = iota
	OneHour
	OneMinute
	OneSecond
)

// Author: Erik Wahlberger
// Returns a string representation for the Unit value, or an empty string if an invalid value was provided
func (u Unit) String() string {
	switch u {
	case OneHour:
		return "1 Hour"
	case OneMinute:
		return "1 Minute"
	case OneSecond:
		return "1 Second"
	}

	return ""
}

// Author: Erik Wahlberger
type CloudProviderPrice struct {
	Price float64
	Unit  Unit
}
