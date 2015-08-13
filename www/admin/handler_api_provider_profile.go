package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/audit"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
)

type providerProfileForm struct {
	ShortTitle          string `json:"short_title"`
	LongTitle           string `json:"long_title"`
	ShortDisplayName    string `json:"short_display_name"`
	LongDisplayName     string `json:"long_display_name"`
	FullName            string `json:"full_name"`
	WhySpruce           string `json:"why_spruce"`
	Qualifications      string `json:"qualifications"`
	MedicalSchool       string `json:"medical_school"`
	GraduateSchool      string `json:"graduate_school"`
	UndergraduateSchool string `json:"undergraduate_school"`
	Residency           string `json:"residency"`
	Fellowship          string `json:"fellowship"`
	Experience          string `json:"experience"`
}

type providerProfileAPIHandler struct {
	dataAPI api.DataAPI
}

func newProviderProfileAPIHandler(dataAPI api.DataAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(&providerProfileAPIHandler{
		dataAPI: dataAPI,
	}, httputil.Get, httputil.Put)
}

func (h *providerProfileAPIHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(mux.Vars(ctx)["id"], 10, 64)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	doctor, err := h.dataAPI.GetDoctorFromID(doctorID)
	if api.IsErrNotFound(err) {
		www.APINotFound(w, r)
		return
	} else if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	account := www.MustCtxAccount(ctx)

	switch r.Method {
	case httputil.Get:
		audit.LogAction(account.ID, "AdminAPI", "GetProviderProfile", map[string]interface{}{"provider_id": doctorID})

		profile, err := h.dataAPI.CareProviderProfile(doctor.AccountID.Int64())
		if err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		// Prepopulate from the onboarding answers
		if profile.WhySpruce == "" {
			attr, err := h.dataAPI.DoctorAttributes(doctorID, []string{api.AttrExcitedAboutSpruce})
			if err != nil {
				golog.Errorf(err.Error())
			} else {
				profile.WhySpruce = attr[api.AttrExcitedAboutSpruce]
			}
		}
		if profile.Qualifications == "" {
			licenses, err := h.dataAPI.MedicalLicenses(doctorID)
			if err != nil {
				golog.Errorf(err.Error())
			} else if states, err := h.dataAPI.ListStates(); err != nil {
				golog.Errorf(err.Error())
			} else {
				stateNames := make(map[string]string)
				for _, s := range states {
					stateNames[s.Abbreviation] = s.Name
				}

				var lic []string
				for _, l := range licenses {
					if l.Status == common.MLActive {
						lic = append(lic, stateNames[l.State])
					}
				}
				switch len(lic) {
				case 0:
				case 1:
					profile.Qualifications = fmt.Sprintf("%s state medical license", lic[0])
				case 2:
					profile.Qualifications = fmt.Sprintf("%s and %s state medical licenses", lic[0], lic[1])
				default:
					profile.Qualifications = fmt.Sprintf("%s, and %s state medical licenses", strings.Join(lic[:len(lic)-1], ", "), lic[len(lic)-1])
				}
			}
		}

		form := &providerProfileForm{
			ShortTitle:          doctor.ShortTitle,
			LongTitle:           doctor.LongTitle,
			ShortDisplayName:    doctor.ShortDisplayName,
			LongDisplayName:     doctor.LongDisplayName,
			FullName:            profile.FullName,
			WhySpruce:           profile.WhySpruce,
			Qualifications:      profile.Qualifications,
			MedicalSchool:       profile.MedicalSchool,
			GraduateSchool:      profile.GraduateSchool,
			UndergraduateSchool: profile.UndergraduateSchool,
			Residency:           profile.Residency,
			Fellowship:          profile.Fellowship,
			Experience:          profile.Experience,
		}
		httputil.JSONResponse(w, http.StatusOK, form)
	case httputil.Put:
		audit.LogAction(account.ID, "AdminAPI", "UpdateProviderProfile", map[string]interface{}{"provider_id": doctorID})

		var form providerProfileForm
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
			FullName:            form.FullName,
			WhySpruce:           form.WhySpruce,
			Qualifications:      form.Qualifications,
			MedicalSchool:       form.MedicalSchool,
			GraduateSchool:      form.GraduateSchool,
			UndergraduateSchool: form.UndergraduateSchool,
			Residency:           form.Residency,
			Fellowship:          form.Fellowship,
			Experience:          form.Experience,
		}
		if err := h.dataAPI.UpdateCareProviderProfile(doctor.AccountID.Int64(), profile); err != nil {
			www.APIInternalError(w, r, err)
			return
		}

		httputil.JSONResponse(w, http.StatusOK, nil)
	}
}
