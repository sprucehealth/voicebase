package patient_file

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
)

type patientAppInfoHandler struct {
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

type appInfo struct {
	Version         *common.Version `json:"version"`
	Build           string          `json:"build"`
	Platform        common.Platform `json:"platform"`
	PlatformVersion string          `json:"platform_version"`
	Device          string          `json:"device"`
	DeviceModel     string          `json:"device_model"`
	LastSeen        time.Time       `json:"last_seen"`
}

func NewPatientAppInfoHandler(dataAPI api.DataAPI, authAPI api.AuthAPI) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&patientAppInfoHandler{
					dataAPI: dataAPI,
					authAPI: authAPI,
				}), []string{api.DOCTOR_ROLE, api.MA_ROLE}),
		[]string{"GET"})
}

func (p *patientAppInfoHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	doctorID, err := p.dataAPI.GetDoctorIdFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	patientIDStr := r.FormValue("patient_id")
	if patientIDStr == "" {
		return false, apiservice.NewValidationError("patient_id not specified", r)
	}

	patientID, err := strconv.ParseInt(patientIDStr, 10, 64)
	if err != nil {
		return false, apiservice.NewValidationError(err.Error(), r)
	}
	ctxt.RequestCache[apiservice.PatientID] = patientID

	// ensure that the doctor has access to the patient file
	if err := apiservice.ValidateDoctorAccessToPatientFile(
		r.Method,
		ctxt.Role,
		doctorID,
		patientID,
		p.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (p *patientAppInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	patientID := ctxt.RequestCache[apiservice.PatientID].(int64)

	patient, err := p.dataAPI.Patient(patientID, true)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	aInfo, err := p.authAPI.LatestAppInfo(patient.AccountId.Int64())
	if err == api.NoRowsError {
		apiservice.WriteResourceNotFoundError("app info not found for patient", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	displayInfo := &appInfo{
		Version:         aInfo.Version,
		Build:           aInfo.Build,
		Platform:        aInfo.Platform,
		PlatformVersion: aInfo.PlatformVersion,
		Device:          aInfo.Device,
		DeviceModel:     aInfo.DeviceModel,
		LastSeen:        aInfo.LastSeen,
	}

	apiservice.WriteJSON(w, struct {
		AppInfo *appInfo `json:"app_info"`
	}{
		AppInfo: displayInfo,
	})
}
