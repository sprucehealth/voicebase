package layout

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/httputil"
)

type diagDetailsLayoutUploadHandler struct {
	dataAPI      api.DataAPI
	diagnosisAPI diagnosis.API
}

type diagnosisLayoutItems struct {
	Items []*diagnosisLayoutItem `json:"diagnosis_layouts"`
}

type diagnosisLayoutItem struct {
	CodeID        string          `json:"code_id"`
	LayoutVersion *common.Version `json:"layout_version"`
	Questions     json.RawMessage `json:"questions"`
}

func NewDiagnosisDetailsIntakeUploadHandler(dataAPI api.DataAPI, diagnosisAPI diagnosis.API) http.Handler {
	return httputil.SupportedMethods(
		apiservice.NoAuthorizationRequired(
			apiservice.SupportedRoles(
				&diagDetailsLayoutUploadHandler{
					dataAPI:      dataAPI,
					diagnosisAPI: diagnosisAPI,
				}, []string{api.ADMIN_ROLE})), []string{"POST"})
}

func (d *diagDetailsLayoutUploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rd := &diagnosisLayoutItems{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// ensure that the diagnosis codes exist
	codeIDs := make([]string, len(rd.Items))
	for i, item := range rd.Items {
		codeIDs[i] = item.CodeID
	}

	if res, nonExistentCodeIDs, err := d.diagnosisAPI.DoCodesExist(codeIDs); err != nil {
		apiservice.WriteError(err, w, r)
		return
	} else if !res {
		apiservice.WriteValidationError(fmt.Sprintf("Following codes do not exist: %v", nonExistentCodeIDs), w, r)
		return
	}

	// ensure that for each of the incoming diagnosis the layout inputted is higher than the layout already
	// supported for the version
	var errors []string
	for _, item := range rd.Items {
		existingVersion, err := d.dataAPI.ActiveDiagnosisDetailsIntakeVersion(item.CodeID)
		if api.IsErrNotFound(err) {
			continue
		} else if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
		if !existingVersion.LessThan(item.LayoutVersion) {
			errors = append(errors,
				fmt.Sprintf("Incoming layout version %s is less than existing layout version %s for codeID %s",
					item.LayoutVersion.String(), existingVersion.String(), item.CodeID))
		}
	}
	if len(errors) > 0 {
		apiservice.WriteValidationError(strings.Join(errors, "\n"), w, r)
		return
	}

	// for each layout entry, create a template, fill in the questions and then create the actual layout
	for _, item := range rd.Items {

		// unmarshal the quesitons into two separate objects so that
		// we have a copy for the template and then a copy into which to fill the
		// question information
		var qIntake []*info_intake.Question
		if err := json.Unmarshal(item.Questions, &qIntake); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		layout := diagnosis.NewQuestionIntake(qIntake)
		template := &common.DiagnosisDetailsIntake{
			CodeID:  item.CodeID,
			Version: item.LayoutVersion,
			Active:  true,
			Layout:  &layout,
		}

		if err := json.Unmarshal(item.Questions, &qIntake); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		if err := api.FillQuestions(qIntake, d.dataAPI, api.EN_LANGUAGE_ID); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		layout = diagnosis.NewQuestionIntake(qIntake)
		info := &common.DiagnosisDetailsIntake{
			CodeID:  item.CodeID,
			Version: item.LayoutVersion,
			Active:  true,
			Layout:  &layout,
		}

		// save the template and the fleshed out object into the database
		if err := d.dataAPI.SetDiagnosisDetailsIntake(template, info); err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	apiservice.WriteJSONSuccess(w)
}
