package httputil

import "net/http"

// Since the name of this pkg overlaps with the stdlib provide access to some useful handlers.
// TODO: now that there's no separate context handler type should really not have these here
var (
	FileServer      = http.FileServer
	NotFoundHandler = http.NotFoundHandler
	RedirectHandler = http.RedirectHandler
	StripPrefix     = http.StripPrefix
)
