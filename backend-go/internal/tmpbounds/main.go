package main

import (
	"fmt"

	"hotspot/backend-go/internal/platform/geo"
)

func main() {
	b := geo.BoundsAround(geo.Point{Latitude: 34.2604, Longitude: 117.1855}, 5000)
	mins, maxs := geo.CoveringArrays(b)
	fmt.Printf("mins: %v\nmaxs: %v\n", mins, maxs)
}
