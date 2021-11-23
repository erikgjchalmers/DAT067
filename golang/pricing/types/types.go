package types

import ()

type Unit uint

const (
	Unknown Unit = iota
	OneHour
	OneMinute
	OneSecond
)

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

type CloudProviderPrice struct {
	Price float64
	Unit  Unit
}
