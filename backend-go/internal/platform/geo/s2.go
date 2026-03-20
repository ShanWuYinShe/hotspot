package geo

import (
	"fmt"
	"math"

	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
)

type Bounds struct {
	North float64
	South float64
	East  float64
	West  float64
}

type Point struct {
	Latitude  float64
	Longitude float64
}

func ValidateBounds(b Bounds) error {
	if b.North < b.South || b.East < b.West {
		return fmt.Errorf("invalid bounds")
	}
	if b.North > 90 || b.South < -90 || b.East > 180 || b.West < -180 {
		return fmt.Errorf("invalid bounds")
	}
	return nil
}

func Center(b Bounds) Point {
	return Point{Latitude: (b.North + b.South) / 2, Longitude: (b.East + b.West) / 2}
}

func ApproxViewportSpanMeters(b Bounds) float64 {
	center := Center(b)
	latMeters := math.Abs(b.North-b.South) * 111000
	lngMeters := math.Abs(b.East-b.West) * 111000 * math.Cos(center.Latitude*math.Pi/180)
	return math.Hypot(latMeters, lngMeters)
}

func SuggestedLevel(spanMeters float64) int {
	switch {
	case spanMeters > 1_000_000:
		return 8
	case spanMeters > 200_000:
		return 10
	case spanMeters > 50_000:
		return 12
	case spanMeters > 10_000:
		return 14
	case spanMeters > 2_000:
		return 16
	default:
		return 18
	}
}

func CellTokenAtLevel(p Point, level int) string {
	ll := s2.LatLngFromDegrees(p.Latitude, p.Longitude)
	cellID := s2.CellIDFromLatLng(ll).Parent(level)
	return cellID.ToToken()
}

func CoveringTokens(b Bounds, level int, maxCells int) []string {
	rect := s2.RectFromLatLng(s2.LatLngFromDegrees(b.South, b.West))
	rect = rect.AddPoint(s2.LatLngFromDegrees(b.North, b.East))
	coverer := &s2.RegionCoverer{MinLevel: level, MaxLevel: level, MaxCells: maxCells}
	ids := coverer.Covering(rect)
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.ToToken())
	}
	return out
}

func DistanceMeters(a, b Point) int {
	p1 := s2.LatLngFromDegrees(a.Latitude, a.Longitude)
	p2 := s2.LatLngFromDegrees(b.Latitude, b.Longitude)
	return int(s1.Angle(p1.Distance(p2)).Radians() * 6371000)
}

func BearingDegrees(a, b Point) int {
	lat1 := a.Latitude * math.Pi / 180
	lat2 := b.Latitude * math.Pi / 180
	dLng := (b.Longitude - a.Longitude) * math.Pi / 180
	y := math.Sin(dLng) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLng)
	bearing := math.Atan2(y, x) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	return int(math.Round(bearing))
}
