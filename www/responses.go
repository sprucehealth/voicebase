package www

import (
	"html/template"
	"io"
	"net/http"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

type Template interface {
	Execute(io.Writer, interface{}) error
}

// Response content types
const (
	HTMLContentType = "text/html; charset=utf-8"
)

// TODO: make this internal and more informative
var errorTemplate = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>{{.Title}}</title>
</head>
<body>
	<h1>{{.Title}}</h1>
	{{.Message}}
</body>
</html>
`))

type errorContext struct {
	Title   string
	Message string
}

// APIError represents an error for an API request
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// APIErrorResponse is the response for an error from any API handler
type APIErrorResponse struct {
	Error APIError `json:"error"`
}

// BadRequestError writes a response with status code of 400 and an HTML page saying "Bad Request".
// The provided error is logged as a warning but not returned in the response.
func BadRequestError(w http.ResponseWriter, r *http.Request, err error) {
	golog.LogDepthf(1, golog.WARN, err.Error())
	TemplateResponse(w, http.StatusBadRequest, errorTemplate, &errorContext{Title: "Bad Request"})
}

// InternalServerError writes a response with status code of 500 and an HTML page saying "Internal Server Error".
// The provided error is logged but not returned in the response.
func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	golog.LogDepthf(1, golog.ERR, err.Error())
	TemplateResponse(w, http.StatusInternalServerError, errorTemplate, &errorContext{Title: "Internal Server Error"})
}

// TemplateResponse writes a response with the provided status code and rendered template
// as the body. The content-type of the response is "text/html".
func TemplateResponse(w http.ResponseWriter, code int, tmpl Template, ctx interface{}) {
	w.Header().Set("Content-Type", HTMLContentType)
	w.WriteHeader(code)
	if err := tmpl.Execute(w, ctx); err != nil {
		golog.LogDepthf(1, golog.ERR, "Failed to render template %+v: %s", tmpl, err.Error())
	}
}

// APIInternalError writes a JSON error response with the message "Internal server error" and status code of 500.
// The provided error is logged but not returned in the response.
func APIInternalError(w http.ResponseWriter, r *http.Request, err error) {
	golog.LogDepthf(1, golog.ERR, err.Error())
	httputil.JSONResponse(w, http.StatusInternalServerError, &APIErrorResponse{APIError{Message: "Internal server error"}})
}

// APIBadRequestError writes a JSON error response with the given message as content and status code of 400.
func APIBadRequestError(w http.ResponseWriter, r *http.Request, msg string) {
	httputil.JSONResponse(w, http.StatusBadRequest, &APIErrorResponse{APIError{Message: msg}})
}

// APINotFound writes a JSON error response with the message "Not found" and status code of 404.
func APINotFound(w http.ResponseWriter, r *http.Request) {
	httputil.JSONResponse(w, http.StatusNotFound, &APIErrorResponse{APIError{Message: "Not found"}})
}

// APIForbidden writes a JSON error response with the message "Access not allowed" and status code of 403.
func APIForbidden(w http.ResponseWriter, r *http.Request) {
	httputil.JSONResponse(w, http.StatusForbidden, &APIErrorResponse{APIError{Message: "Access not allowed"}})
}
