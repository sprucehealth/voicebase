package httputil

import (
	"encoding/json"
	"net/http"

	"github.com/sprucehealth/backend/libs/golog"
)

var (
	JSONContentType = "application/json"
)

func JSONResponse(w http.ResponseWriter, statusCode int, res interface{}) {
	w.Header().Set("Content-Type", JSONContentType)
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		golog.LogDepthf(1, golog.ERR, err.Error())
	}
}
