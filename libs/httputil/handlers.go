package httputil

import "net/http"

// RedirectHandler is a context aware version of http.RedirectHandler
func RedirectHandler(url string, code int) ContextHandler {
	return ToContextHandler(http.RedirectHandler(url, code))
}

// NotFoundHandler is a context aware version of http.NotFoundHandler
func NotFoundHandler() ContextHandler {
	return ToContextHandler(http.NotFoundHandler())
}

// StripPrefix is a context aware version of http.StripPrefix
func StripPrefix(prefix string, h ContextHandler) ContextHandler {
	return ToContextHandler(http.StripPrefix(prefix, FromContextHandler(h)))
}

// FileServer is a context aware version of http.FileServer
func FileServer(root http.FileSystem) ContextHandler {
	return ToContextHandler(http.FileServer(root))
}
