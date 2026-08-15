package types

// RelativeLocation is the shared location-with-bearing shape used across
// modules. It embeds a GeoPoint for the coordinate. Modules alias this instead
// of redefining it.
//
// Distance (meters) and BearingDegrees are *float64 to preserve full precision
// from geo calculations and to match the API spec which defines both as
// "number". A nil value means the field was not computed (no reference point).
type RelativeLocation struct {
	Point          GeoPoint `json:"point"`
	LocationName   string   `json:"location_name"`
	Distance       *float64 `json:"distance"`
	BearingDegrees *float64 `json:"bearing_degrees"`
}
