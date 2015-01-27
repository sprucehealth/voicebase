package handlers

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/libs/httputil"
)

type diagnosisHandler struct {
	dataAPI      api.DataAPI
	diagnosisAPI diagnosis.API
}

func NewDiagnosisHandler(dataAPI api.DataAPI, diagnosisAPI diagnosis.API) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.NoAuthorizationRequired(
				&diagnosisHandler{
					dataAPI:      dataAPI,
					diagnosisAPI: diagnosisAPI,
				},
			), []string{api.DOCTOR_ROLE}), []string{"GET"})
}

func (d *diagnosisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	codeID := r.FormValue("code_id")

	diagnosisMap, err := d.diagnosisAPI.DiagnosisForCodeIDs([]string{codeID})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	diag := diagnosisMap[codeID]
	if diag == nil {
		apiservice.WriteResourceNotFoundError("diagnosis not found", w, r)
		return
	}

	// attempt to get the latest diagnosis details layout for the code
	// given that the doctor app tends to run the latest version of the app
	// and we don't have to worry about selecting which layout to show the doctor
	// for the diagnosis details intake based on the app version
	detailsIntake, err := d.dataAPI.ActiveDiagnosisDetailsIntake(codeID, diagnosis.DetailTypes)
	if !api.IsErrNotFound(err) && err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	outputItem := &DiagnosisOutputItem{
		CodeID:     codeID,
		Code:       diag.Code,
		Title:      diag.Description,
		HasDetails: detailsIntake != nil,
	}

	if detailsIntake != nil {
		outputItem.Questions = detailsIntake.Layout.(*diagnosis.QuestionIntake).Questions()
		outputItem.LayoutVersion = detailsIntake.Version
		outputItem.LatestLayoutVersion = detailsIntake.Version
	}

	apiservice.WriteJSON(w, outputItem)
}
