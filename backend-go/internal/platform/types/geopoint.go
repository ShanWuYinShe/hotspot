package types

// GeoPoint is the shared latitude/longitude shape used across modules. Modules
// alias this instead of redefining it, so a field rename doesn't drift across
// many copies.
type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
