package www

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/libs/golog"
)

type Template interface {
	Execute(io.Writer, interface{}) error
}

const (
	HTMLContentType = "text/html; charset=utf-8"
	JSONContentType = "application/json"
)

// TODO: make this internal and more informative
var internalErrorTemplate = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>Internal Server Error</title>
</head>
<body>
	Internal Server Error
	{{.Message}}
</body>
</html>
`))

type internalErrorContext struct {
	Message string
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type APIErrorResponse struct {
	Error APIError `json:"error"`
}

func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	golog.LogDepthf(1, golog.ERR, err.Error())
	TemplateResponse(w, http.StatusInternalServerError, internalErrorTemplate, &internalErrorContext{})
}

func TemplateResponse(w http.ResponseWriter, code int, tmpl Template, ctx interface{}) {
	w.Header().Set("Content-Type", HTMLContentType)
	w.WriteHeader(code)
	if err := tmpl.Execute(w, ctx); err != nil {
		golog.LogDepthf(1, golog.ERR, "Failed to render template %+v: %s", tmpl, err.Error())
	}
}

func APIInternalError(w http.ResponseWriter, r *http.Request, err error) {
	golog.LogDepthf(1, golog.ERR, err.Error())
	JSONResponse(w, r, http.StatusInternalServerError, &APIErrorResponse{APIError{Message: "Internal server error"}})
}

func APIBadRequestError(w http.ResponseWriter, r *http.Request, msg string) {
	JSONResponse(w, r, http.StatusBadRequest, &APIErrorResponse{APIError{Message: msg}})
}

func APINotFound(w http.ResponseWriter, r *http.Request) {
	JSONResponse(w, r, http.StatusNotFound, &APIErrorResponse{APIError{Message: "Not found"}})
}

func APIForbidden(w http.ResponseWriter, r *http.Request) {
	JSONResponse(w, r, http.StatusForbidden, &APIErrorResponse{APIError{Message: "Access not allowed"}})
}

func JSONResponse(w http.ResponseWriter, r *http.Request, statusCode int, res interface{}) {
	if b, _ := strconv.ParseBool(r.FormValue("indented")); b {
		body, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			golog.LogDepthf(1, golog.ERR, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", JSONContentType)
		w.WriteHeader(statusCode)
		if _, err := w.Write(body); err != nil {
			golog.LogDepthf(1, golog.ERR, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", JSONContentType)
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		golog.LogDepthf(1, golog.ERR, err.Error())
	}
}
