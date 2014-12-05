package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-librato/librato"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type libratoCompositeAPIHandler struct {
	lc *librato.Client
}

func NewLibratoCompositeAPIHandler(lc *librato.Client) http.Handler {
	return httputil.SupportedMethods(&libratoCompositeAPIHandler{
		lc: lc,
	}, []string{"GET"})
}

func (h *libratoCompositeAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.lc == nil {
		www.JSONResponse(w, r, http.StatusOK, nil)
		return
	}

	compose := r.FormValue("compose")
	if compose == "" {
		www.APIBadRequestError(w, r, "compose must not be empty")
		return
	}
	resolution, err := strconv.Atoi(r.FormValue("resolution"))
	if err != nil {
		www.APIBadRequestError(w, r, "resolution must be an integer")
		return
	}
	var startTime time.Time
	ts, err := strconv.ParseInt(r.FormValue("start_time"), 10, 64)
	if err != nil {
		www.APIBadRequestError(w, r, "start_time must be an integer")
		return
	}
	startTime = time.Unix(ts, 0)
	var endTime time.Time
	var count int
	if s := r.FormValue("end_time"); s != "" {
		ts, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			www.APIBadRequestError(w, r, "end_time must be an integer")
			return
		}
		endTime = time.Unix(ts, 0)
	}
	if s := r.FormValue("count"); s != "" {
		count, err = strconv.Atoi(s)
		if err != nil {
			www.APIBadRequestError(w, r, "count must be an integer")
			return
		}
	}

	res, err := h.lc.QueryComposite(compose, resolution, startTime, endTime, count)
	if err != nil {
		// TODO: would be nice to distinguish API errors from communication errors
		www.APIInternalError(w, r, err)
		return
	}
	www.JSONResponse(w, r, http.StatusOK, res)
}
