package modulekit

import "github.com/go-chi/chi/v5"

type Module interface {
	Mount(r chi.Router)
}

// PublicMounter is implemented by modules that also serve anonymous endpoints
// on the /public/v1 node (spec api_spec_public.md). The router passed in is
// rooted at /public/v1 and carries no auth middleware — handlers mounted here
// must resolve to the anonymous (viewer-less) view.
type PublicMounter interface {
	MountPublic(r chi.Router)
}
