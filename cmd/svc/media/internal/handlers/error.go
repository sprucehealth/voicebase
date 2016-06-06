package handlers

import (
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/libs/golog"
)

func internalError(w http.ResponseWriter, err error) {
	golog.LogDepthf(1, golog.ERR, "Media: Internal Error: %s", err)
	http.Error(w, "Internal Error", http.StatusInternalServerError)
}

func forbidden(w http.ResponseWriter, err error, errLvl golog.Level) {
	golog.LogDepthf(1, errLvl, "Media: Forbidden: %s", err)
	http.Error(w, "Forbidden", http.StatusForbidden)
}

func badRequest(w http.ResponseWriter, err error, errLvl golog.Level) {
	golog.LogDepthf(1, errLvl, "Media: Bad Request: %s", err)
	http.Error(w, fmt.Sprintf("Bad Request: %s", err), http.StatusBadRequest)
}
