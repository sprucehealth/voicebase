package patient_visit

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
)

type diagnosePatientHandler struct {
	dataAPI    api.DataAPI
	authAPI    api.AuthAPI
	dispatcher *dispatch.Dispatcher
}

func NewDiagnosePatientHandler(dataAPI api.DataAPI, authAPI api.AuthAPI, dispatcher *dispatch.Dispatcher) *diagnosePatientHandler {
	cacheInfoForUnsuitableVisit(dataAPI)
	return &diagnosePatientHandler{
		dataAPI:    dataAPI,
		authAPI:    authAPI,
		dispatcher: dispatcher,
	}
}

type GetDiagnosisResponse struct {
	DiagnosisLayout *info_intake.DiagnosisIntake `json:"diagnosis"`
}

type DiagnosePatientRequestData struct {
	PatientVisitId int64 `schema:"patient_visit_id,required"`
}

func (d *diagnosePatientHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case apiservice.HTTP_GET:
		d.getDiagnosis(w, r)
	case apiservice.HTTP_POST:
		d.diagnosePatient(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *diagnosePatientHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)

	doctorID, err := d.dataAPI.GetDoctorIdFromAccountId(ctxt.AccountId)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	switch r.Method {
	case apiservice.HTTP_GET:
		rd := new(DiagnosePatientRequestData)
		if err := apiservice.DecodeRequestData(rd, r); err != nil {
			return false, apiservice.NewValidationError(err.Error(), r)
		} else if rd.PatientVisitId == 0 {
			return false, apiservice.NewValidationError("patient_id must be specified", r)
		}
		ctxt.RequestCache[apiservice.RequestData] = rd

		patientVisit, err := d.dataAPI.GetPatientVisitFromId(rd.PatientVisitId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientVisit] = patientVisit

		if err := apiservice.ValidateAccessToPatientCase(
			r.Method,
			ctxt.Role,
			doctorID,
			patientVisit.PatientId.Int64(),
			patientVisit.PatientCaseId.Int64(),
			d.dataAPI); err != nil {
			return false, err
		}

		if ctxt.Role == api.MA_ROLE {
			// identify the doctor on the case to surface the diagnosis to the MA
			assignments, err := d.dataAPI.GetActiveMembersOfCareTeamForCase(
				patientVisit.PatientCaseId.Int64(),
				false)
			if err != nil {
				return false, err
			}

			var doctorOnCase *common.Doctor
			for _, assignment := range assignments {
				if assignment.ProviderRole == api.DOCTOR_ROLE {
					doctorOnCase, err = d.dataAPI.GetDoctorFromId(assignment.ProviderID)
					if err != nil {
						return false, err
					}
					ctxt.RequestCache[apiservice.DoctorID] = doctorOnCase.DoctorId.Int64()
					break
				}
			}

		}
	case apiservice.HTTP_POST:
		rb := &apiservice.AnswerIntakeRequestBody{}
		if err := apiservice.DecodeRequestData(rb, r); err != nil {
			return false, apiservice.NewValidationError(err.Error(), r)
		} else if rb.PatientVisitId == 0 {
			return false, apiservice.NewValidationError("patient_visit_id must be specified", r)
		}
		ctxt.RequestCache[apiservice.RequestData] = rb

		patientVisit, err := d.dataAPI.GetPatientVisitFromId(rb.PatientVisitId)
		if err != nil {
			return false, err
		}
		ctxt.RequestCache[apiservice.PatientVisit] = patientVisit

		if err := apiservice.ValidateAccessToPatientCase(
			r.Method,
			ctxt.Role,
			doctorID,
			patientVisit.PatientId.Int64(),
			patientVisit.PatientCaseId.Int64(),
			d.dataAPI); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (d *diagnosePatientHandler) getDiagnosis(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	doctorID := ctxt.RequestCache[apiservice.DoctorID].(int64)
	patientVisit := ctxt.RequestCache[apiservice.PatientVisit].(*common.PatientVisit)

	diagnosisLayout, err := GetDiagnosisLayout(d.dataAPI, patientVisit, doctorID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	apiservice.WriteJSON(w, &GetDiagnosisResponse{DiagnosisLayout: diagnosisLayout})
}

func (d *diagnosePatientHandler) diagnosePatient(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	rb := ctxt.RequestCache[apiservice.RequestData].(*apiservice.AnswerIntakeRequestBody)
	doctorID := ctxt.RequestCache[apiservice.DoctorID].(int64)
	patientVisit := ctxt.RequestCache[apiservice.PatientVisit].(*common.PatientVisit)

	if err := apiservice.EnsurePatientVisitInExpectedStatus(d.dataAPI, rb.PatientVisitId, common.PVStatusReviewing); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	layoutVersionID, err := d.dataAPI.GetLayoutVersionIdOfActiveDiagnosisLayout(api.HEALTH_CONDITION_ACNE_ID)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	answers := make(map[int64][]*common.AnswerIntake)
	for _, questionItem := range rb.Questions {
		// enumerate the answers to store from the top level questions as well as the sub questions
		answers[questionItem.QuestionId] = apiservice.PopulateAnswersToStoreForQuestion(
			api.DOCTOR_ROLE,
			questionItem,
			rb.PatientVisitId,
			doctorID,
			layoutVersionID)
	}

	diagnosisIntake := &api.DiagnosisIntake{
		DoctorID:       doctorID,
		PatientVisitID: rb.PatientVisitId,
		LVersionID:     layoutVersionID,
		Intake:         answers,
		SID:            rb.SessionID,
		SCounter:       rb.SessionCounter,
	}

	if err := d.dataAPI.StoreAnswersForQuestion(diagnosisIntake); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// check if the doctor diagnosed the patient's visit as being unsuitable for spruce
	unsuitableReason, wasMarkedUnsuitable := wasVisitMarkedUnsuitableForSpruce(rb)
	if wasMarkedUnsuitable {
		err = d.dataAPI.ClosePatientVisit(rb.PatientVisitId, common.PVStatusTriaged)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		d.dispatcher.Publish(&PatientVisitMarkedUnsuitableEvent{
			DoctorID:       doctorID,
			PatientID:      patientVisit.PatientId.Int64(),
			CaseID:         patientVisit.PatientCaseId.Int64(),
			PatientVisitID: rb.PatientVisitId,
			InternalReason: unsuitableReason,
		})

	} else {
		diagnosis := determineDiagnosisFromAnswers(rb)

		if err := d.dataAPI.UpdateDiagnosisForVisit(
			patientVisit.PatientVisitId.Int64(),
			doctorID,
			diagnosis); err != nil {
			golog.Errorf("Unable to update diagnosis for patient visit: %s", err)
		}

		d.dispatcher.Publish(&DiagnosisModifiedEvent{
			DoctorID:       doctorID,
			PatientID:      patientVisit.PatientId.Int64(),
			PatientVisitID: rb.PatientVisitId,
			PatientCaseID:  patientVisit.PatientCaseId.Int64(),
			Diagnosis:      diagnosis,
		})
	}

	apiservice.WriteJSONSuccess(w)
}
