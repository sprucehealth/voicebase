package httputil

import "net/http/httputil"

// Alias useful functions from the stdlib httputil
var (
	DumpRequest    = httputil.DumpRequest
	DumpRequestOut = httputil.DumpRequestOut
	DumpResponse   = httputil.DumpResponse
)
