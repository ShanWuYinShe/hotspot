package realtime

var defaultHub = NewHub()

func SetDefaultHub(hub *Hub) {
	if hub != nil {
		defaultHub = hub
	}
}
func DefaultHub() *Hub { return defaultHub }

// ResetHub replaces the default hub with a fresh instance.
// Intended for use in test suites to ensure test isolation.
func ResetHub() { defaultHub = NewHub() }
