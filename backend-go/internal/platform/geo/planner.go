package geo

import "math"

type SearchPlan struct {
	Reference Point
	Radius    int
	SortBy    string
	Level     int
	covering  map[string]struct{}
}

func NewSearchPlan(reference Point, radius int, sortBy string) SearchPlan {
	if radius <= 0 {
		radius = 5000
	}
	if radius > 50000 {
		radius = 50000
	}
	span := float64(radius * 2)
	level := SuggestedLevel(span)
	bounds := BoundsAround(reference, radius)
	tokens := CoveringTokens(bounds, level, 32)
	covering := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		covering[token] = struct{}{}
	}
	if sortBy == "" {
		sortBy = "relevance"
	}
	return SearchPlan{Reference: reference, Radius: radius, SortBy: sortBy, Level: level, covering: covering}
}

func BoundsAround(reference Point, radiusMeters int) Bounds {
	latDelta := float64(radiusMeters) / 111000.0
	cosLat := math.Cos(reference.Latitude * math.Pi / 180)
	if math.Abs(cosLat) < 0.01 {
		cosLat = 0.01
	}
	lngDelta := float64(radiusMeters) / (111000.0 * cosLat)
	return Bounds{
		North: clamp(reference.Latitude+latDelta, -90, 90),
		South: clamp(reference.Latitude-latDelta, -90, 90),
		East:  clamp(reference.Longitude+lngDelta, -180, 180),
		West:  clamp(reference.Longitude-lngDelta, -180, 180),
	}
}

func (p SearchPlan) Match(candidate Point) (distance int, bearing int, ok bool) {
	token := CellTokenAtLevel(candidate, p.Level)
	if _, found := p.covering[token]; !found {
		return 0, 0, false
	}
	distance = DistanceMeters(p.Reference, candidate)
	if distance > p.Radius {
		return 0, 0, false
	}
	bearing = BearingDegrees(p.Reference, candidate)
	return distance, bearing, true
}

func clamp(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
