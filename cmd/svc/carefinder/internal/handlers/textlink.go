package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"bytes"

	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/dal"

	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type textLinkHandler struct {
	doctorDAL dal.DoctorDAL
	webHost   string
}

type textLinkRequestData struct {
	DoctorID string `json:"doctor_id"`
	Number   string `json:"number"`
}

type textLinkResponseData struct {
	Success bool `json:"success"`
}

func NewTextLinkHandler(doctorDAL dal.DoctorDAL, webURL string) httputil.ContextHandler {
	u, err := url.Parse(webURL)
	if err != nil {
		panic(err)
	}

	return httputil.SupportedMethods(&textLinkHandler{
		doctorDAL: doctorDAL,
		webHost:   u.Scheme + "://" + u.Host,
	}, httputil.Post)
}

func (t *textLinkHandler) ServeHTTP(context context.Context, w http.ResponseWriter, r *http.Request) {
	var req textLinkRequestData
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		www.APIBadRequestError(w, r, "could not decode request body")
		return
	}

	if req.DoctorID == "" {
		www.APIBadRequestError(w, r, "missing doctor id")
		return
	}

	doctor, err := t.doctorDAL.Doctor(req.DoctorID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	if !doctor.IsSpruceDoctor {
		www.APIBadRequestError(w, r, "can only start an online visit with a spruce doctor")
		return
	}

	// add a flag to indicate that the patient is indeed a spruce patient
	// so that it doesn't get picked up as a practice extension patient.
	// also add the provider id to attribute this patient with the provider they picked.
	jsonData, err := json.Marshal(map[string]interface{}{
		"number": req.Number,
		"code":   doctor.ReferralCode,
		"params": map[string][]string{
			"is_spruce_patient": {"true"},
			"care_provider_id":  {strconv.FormatInt(doctor.SpruceProviderID, 10)},
		},
	})
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	// once we have verified that the doctor is a spruce doctor, lets forward the request
	// to the API that texts a link to download the app for the user. The reason to leverage this API
	// is because we get rate limiting and validation of the promo code for free. Rate limiting is important
	// here because this is an unauthenticated endpoint with the potential for anyone to find it and send multiple
	// messages to a number.
	postReq, err := http.NewRequest("POST", fmt.Sprintf("%s/api/textdownloadlink", t.webHost), bytes.NewReader(jsonData))
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	postReq.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(postReq)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	if res.StatusCode != http.StatusOK {
		var e www.APIErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			www.InternalServerError(w, r, err)
			return
		}
		res.Body.Close()

		www.APIGeneralError(w, r, e.Error.Type, e.Error.Message)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, textLinkResponseData{Success: true})
}
