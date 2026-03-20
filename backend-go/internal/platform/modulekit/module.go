package modulekit

import "github.com/go-chi/chi/v5"

type Module interface {
	Mount(r chi.Router)
}
