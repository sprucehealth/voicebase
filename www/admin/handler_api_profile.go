package admin

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/third_party/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/www"
)

type doctorProfileForm struct {
	ShortTitle       string `json:"short_title"`
	LongTitle        string `json:"long_title"`
	ShortDisplayName string `json:"short_display_name"`
	LongDisplayName  string `json:"long_display_name"`
	FullName         string `json:"full_name"`
	WhySpruce        string `json:"why_spruce"`
	Qualifications   string `json:"qualifications"`
	MedicalSchool    string `json:"medical_school"`
	Residency        string `json:"residency"`
	Fellowship       string `json:"fellowship"`
	Experience       string `json:"experience"`
}

type doctorProfileAPIHandler struct {
	dataAPI api.DataAPI
}

func NewDoctorProfileAPIHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&doctorProfileAPIHandler{
		dataAPI: dataAPI,
	}, []string{"GET", "POST"})
}

func (h *doctorProfileAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	doctor, err := h.dataAPI.GetDoctorFromId(doctorID)
	if err == api.NoRowsError {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	switch r.Method {
	case "GET":
		profile, err := h.dataAPI.CareProviderProfile(doctor.AccountId.Int64())
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		// Prepopulate from the onboarding answer to "Excited About Spruce"
		if profile.WhySpruce == "" {
			attr, err := h.dataAPI.DoctorAttributes(doctorID, []string{api.AttrExcitedAboutSpruce})
			if err != nil {
				golog.Errorf(err.Error())
			} else {
				profile.WhySpruce = attr[api.AttrExcitedAboutSpruce]
			}
		}

		form := &doctorProfileForm{
			ShortTitle:       doctor.ShortTitle,
			LongTitle:        doctor.LongTitle,
			ShortDisplayName: doctor.ShortDisplayName,
			LongDisplayName:  doctor.LongDisplayName,
			FullName:         profile.FullName,
			WhySpruce:        profile.WhySpruce,
			Qualifications:   profile.Qualifications,
			MedicalSchool:    profile.MedicalSchool,
			Residency:        profile.Residency,
			Fellowship:       profile.Fellowship,
			Experience:       profile.Experience,
		}
		www.JSONResponse(w, r, http.StatusOK, form)
	case "POST":
		var form doctorProfileForm
		if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		drUpdate := &api.DoctorUpdate{
			ShortTitle:       &form.ShortTitle,
			LongTitle:        &form.LongTitle,
			ShortDisplayName: &form.ShortDisplayName,
			LongDisplayName:  &form.LongDisplayName,
		}
		if err := h.dataAPI.UpdateDoctor(doctorID, drUpdate); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		profile := &common.CareProviderProfile{
			FullName:       form.FullName,
			WhySpruce:      form.WhySpruce,
			Qualifications: form.Qualifications,
			MedicalSchool:  form.MedicalSchool,
			Residency:      form.Residency,
			Fellowship:     form.Fellowship,
			Experience:     form.Experience,
		}
		if err := h.dataAPI.UpdateCareProviderProfile(doctor.AccountId.Int64(), profile); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		www.JSONResponse(w, r, http.StatusOK, nil)
	}
}
