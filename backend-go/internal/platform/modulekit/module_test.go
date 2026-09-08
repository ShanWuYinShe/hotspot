package modulekit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type dummyModule struct {
	mounted       bool
	publicMounted bool
}

func (d *dummyModule) Mount(r chi.Router) {
	d.mounted = true
	r.Get("/private/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (d *dummyModule) MountPublic(r chi.Router) {
	d.publicMounted = true
	r.Get("/public/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestModuleAndPublicMounterInterfaces(t *testing.T) {
	d := &dummyModule{}
	var mod Module = d
	var pub PublicMounter = d

	r := chi.NewRouter()
	mod.Mount(r)
	pub.MountPublic(r)

	if !d.mounted || !d.publicMounted {
		t.Fatalf("expected mounted and publicMounted, got %v, %v", d.mounted, d.publicMounted)
	}

	// 验证路由真实生效
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, httptest.NewRequest(http.MethodGet, "/private/hello", nil))
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected private route to return 200, got %d", rr1.Code)
	}

	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/public/hello", nil))
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected public route to return 200, got %d", rr2.Code)
	}
}
