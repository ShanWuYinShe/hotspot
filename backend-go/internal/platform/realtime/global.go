package realtime

var defaultHub = NewHub()

func SetDefaultHub(hub *Hub) {
	if hub != nil {
		defaultHub = hub
	}
}
func DefaultHub() *Hub { return defaultHub }
