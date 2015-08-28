package patient_file

import (
	"net/http"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"golang.org/x/net/context"
)

type patientAppInfoHandler struct {
	dataAPI api.DataAPI
	authAPI api.AuthAPI
}

type appInfo struct {
	Version         *encoding.Version `json:"version"`
	Build           string            `json:"build"`
	Platform        common.Platform   `json:"platform"`
	PlatformVersion string            `json:"platform_version"`
	Device          string            `json:"device"`
	DeviceModel     string            `json:"device_model"`
	LastSeen        time.Time         `json:"last_seen"`
}

func NewPatientAppInfoHandler(dataAPI api.DataAPI, authAPI api.AuthAPI) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(
					&patientAppInfoHandler{
						dataAPI: dataAPI,
						authAPI: authAPI,
					})),
			api.RoleDoctor, api.RoleCC),
		httputil.Get)
}

func (p *patientAppInfoHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	doctorID, err := p.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctorID] = doctorID

	patientIDStr := r.FormValue("patient_id")
	if patientIDStr == "" {
		return false, apiservice.NewValidationError("patient_id not specified")
	}

	patientIDInt, err := strconv.ParseUint(patientIDStr, 10, 64)
	if err != nil {
		return false, apiservice.NewValidationError(err.Error())
	}

	patientID := common.NewPatientID(patientIDInt)
	requestCache[apiservice.CKPatientID] = patientID

	// ensure that the doctor has access to the patient file
	if err := apiservice.ValidateDoctorAccessToPatientFile(
		r.Method,
		account.Role,
		doctorID,
		patientID,
		p.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (p *patientAppInfoHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	patientID := requestCache[apiservice.CKPatientID].(common.PatientID)

	patient, err := p.dataAPI.Patient(patientID, true)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	aInfo, err := p.authAPI.LatestAppInfo(patient.AccountID.Int64())
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError(ctx, "app info not found for patient", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
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

	httputil.JSONResponse(w, http.StatusOK, struct {
		AppInfo *appInfo `json:"app_info"`
	}{
		AppInfo: displayInfo,
	})
}
