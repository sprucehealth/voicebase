package apiservice

import "net/http"

type isAuthenticatedHandler int64

func NewIsAuthenticatedHandler() *isAuthenticatedHandler {
	handler := isAuthenticatedHandler(0)
	return &handler
}

func (i *isAuthenticatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// nothing to do other than return a valid response because if the auth token in the request
	// header was invalid or non-existent, then the request would be trapped higher in the chain and a 403 returned
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, SuccessfulGenericJSONResponse())
}
