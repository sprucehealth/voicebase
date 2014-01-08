package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
	"github.com/gorilla/schema"
	"net/http"
)

type DoctorAdviceHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type GetDoctorAdviceRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id"`
}

func NewDoctorAdviceHandler(dataApi api.DataAPI) *DoctorAdviceHandler {
	return &DoctorAdviceHandler{DataApi: dataApi, accountId: 0}
}

func (d *DoctorAdviceHandler) AccountIdFromAuthToken(accountId int64) {
	d.accountId = accountId
}

func (d *DoctorAdviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.getAdvicePoints(w, r)
	case "POST":
		d.updateAdvicePoints(w, r)
	}
}

func (d *DoctorAdviceHandler) getAdvicePoints(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	requestData := new(GetDoctorAdviceRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, d.accountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	advicePoints, err := d.DataApi.GetAdvicePointsForDoctor(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get advice points for doctor: "+err.Error())
		return
	}

	selectedAdvicePoints, err := d.DataApi.GetAdvicePointsForPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the selected advice points for this patient visit: "+err.Error())
		return
	}

	responseData := &common.Advice{}
	responseData.AllAdvicePoints = advicePoints
	responseData.SelectedAdvicePoints = selectedAdvicePoints
	responseData.PatientVisitId = requestData.PatientVisitId

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, responseData)
}

func (d *DoctorAdviceHandler) updateAdvicePoints(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	requestData := &common.Advice{}

	err := jsonDecoder.Decode(requestData)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse json request body for updating advice points: "+err.Error())
		return
	}

	doctorId, _, _, statusCode, err := ValidateDoctorAccessToPatientVisitAndGetRelevantData(requestData.PatientVisitId, d.accountId, d.DataApi)
	if err != nil {
		WriteDeveloperError(w, statusCode, err.Error())
		return
	}

	// Go through advice points to add, update and delete advice points before creating the advice points for this patient visit
	// for the user
	newOrUpdatedPointToIdMapping := make(map[string]int64)
	updatedAdvicePoints := make([]*common.DoctorInstructionItem, 0)
	for _, advicePoint := range requestData.AllAdvicePoints {
		switch advicePoint.State {
		case common.STATE_ADDED, common.STATE_MODIFIED:
			err = d.DataApi.AddOrUpdateAdvicePointForDoctor(advicePoint, doctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add or update advice point for doctor. Application may be left in inconsistent state. Error = "+err.Error())
				return
			}
			newOrUpdatedPointToIdMapping[advicePoint.Text] = advicePoint.Id
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		case common.STATE_DELETED:
			err = d.DataApi.MarkAdvicePointToBeDeleted(advicePoint, doctorId)
			if err != nil {
				WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete advice point for doctor: "+err.Error())
				return
			}
		default:
			updatedAdvicePoints = append(updatedAdvicePoints, advicePoint)
		}
		// empty out the state now that it has been taken care of
		advicePoint.State = ""
	}

	// go through regimen steps within the regimen sections to assign ids to the new steps that dont have them
	for _, advicePoint := range requestData.SelectedAdvicePoints {
		updatedOrNewId := newOrUpdatedPointToIdMapping[advicePoint.Text]
		if updatedOrNewId != 0 {
			advicePoint.Id = updatedOrNewId
		}
	}

	err = d.DataApi.CreateAdviceForPatientVisit(requestData.SelectedAdvicePoints, requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add advice for patient visit: "+err.Error())
		return
	}

	requestData.AllAdvicePoints = updatedAdvicePoints
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, requestData)
}
