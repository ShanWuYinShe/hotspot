package types

// RelativeLocation is the shared location-with-bearing shape used across
// modules. It embeds a GeoPoint for the coordinate. Modules alias this instead
// of redefining it.
type RelativeLocation struct {
	Point          GeoPoint `json:"point"`
	LocationName   string   `json:"location_name"`
	Distance       int      `json:"distance,omitempty"`
	BearingDegrees int      `json:"bearing_degrees,omitempty"`
}
