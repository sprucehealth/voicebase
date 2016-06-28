package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/libs/golog"
)

func internalError(w http.ResponseWriter, err error) {
	golog.LogDepthf(1, golog.ERR, "Invite: Internal Error: %s", err)
	http.Error(w, "Internal Error", http.StatusInternalServerError)
}
