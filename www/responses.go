package www

import (
	"github.com/sprucehealth/backend/libs/golog"
	"html/template"
	"io"
	"net/http"
)

type Template interface {
	Execute(io.Writer, interface{}) error
}

const HTMLContentType = "text/html; charset=utf-8"

// TODO: make this internal and more informative
var internalErrorTemplate = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
<head><title>Internal Server Error</title></head>
<body>
	Internal Server Error
	{{.Message}}
</body>
</html>
`))

type internalErrorContext struct {
	Message string
}

func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	golog.Logf(2, golog.ERR, err.Error())
	TemplateResponse(w, http.StatusInternalServerError, internalErrorTemplate, &internalErrorContext{})
}

func TemplateResponse(w http.ResponseWriter, code int, tmpl Template, ctx interface{}) {
	w.Header().Set("Content-Type", HTMLContentType)
	w.WriteHeader(code)
	if err := tmpl.Execute(w, ctx); err != nil {
		golog.Logf(2, golog.ERR, "Failed to render template %+v: %s", tmpl, err.Error())
	}
}
