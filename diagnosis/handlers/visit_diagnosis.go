package handlers

import (
	"net/http"
	"sort"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/patient_visit"
)

type diagnosisListHandler struct {
	dataAPI      api.DataAPI
	diagnosisAPI diagnosis.API
	dispatcher   *dispatch.Dispatcher
}

type DiagnosisListRequestData struct {
	VisitID        int64                 `schema:"patient_visit_id,required" json:"patient_visit_id,string"`
	Diagnoses      []*DiagnosisInputItem `json:"diagnoses"`
	InternalNote   string                `json:"internal_note"`
	CaseManagement CaseManagementItem    `json:"case_management"`
}

type DiagnosisInputItem struct {
	CodeID         string                           `json:"code_id"`
	LayoutVersion  *common.Version                  `json:"layout_version"`
	SessionID      string                           `json:"session_id"`
	SessionCounter uint                             `json:"counter"`
	Answers        []*apiservice.QuestionAnswerItem `json:"answers"`
}

type CaseManagementItem struct {
	Unsuitable bool   `json:"unsuitable"`
	Reason     string `json:"reason"`
}

type DiagnosisListResponse struct {
	Notes          string                 `json:"internal_note"`
	Diagnoses      []*DiagnosisOutputItem `json:"diagnoses"`
	CaseManagement CaseManagementItem     `json:"case_management"`
}

type DiagnosisOutputItem struct {
	CodeID              string                           `json:"code_id"`
	Code                string                           `json:"display_diagnosis_code"`
	Title               string                           `json:"title"`
	Synonyms            string                           `json:"synonyms"`
	HasDetails          bool                             `json:"has_details"`
	LayoutVersion       *common.Version                  `json:"layout_version"`
	LatestLayoutVersion *common.Version                  `json:"latest_layout_version"`
	Questions           []*info_intake.Question          `json:"questions,omitempty"`
	Answers             []*apiservice.QuestionAnswerItem `json:"answers,omitempty"`
}

func NewDiagnosisListHandler(dataAPI api.DataAPI, diagnosisAPI diagnosis.API, dispatcher *dispatch.Dispatcher) http.Handler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.AuthorizationRequired(
				&diagnosisListHandler{
					dataAPI:      dataAPI,
					diagnosisAPI: diagnosisAPI,
					dispatcher:   dispatcher,
				}), []string{api.DOCTOR_ROLE, api.MA_ROLE}),
		[]string{"GET", "PUT"})
}

func (d *diagnosisListHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	rd := &DiagnosisListRequestData{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	} else if rd.VisitID == 0 {
		return false, apiservice.NewValidationError("patient_visit_id required")
	}
	ctxt.RequestCache[apiservice.RequestData] = rd

	doctorID, err := d.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	patientVisit, err := d.dataAPI.GetPatientVisitFromID(rd.VisitID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.PatientVisit] = patientVisit

	if err := apiservice.ValidateAccessToPatientCase(
		r.Method,
		ctxt.Role,
		doctorID,
		patientVisit.PatientID.Int64(),
		patientVisit.PatientCaseID.Int64(),
		d.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (d *diagnosisListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.getDiagnosisList(w, r)
	case "PUT":
		d.putDiagnosisList(w, r)
	}
}

func (d *diagnosisListHandler) putDiagnosisList(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	visit := ctxt.RequestCache[apiservice.PatientVisit].(*common.PatientVisit)
	doctorID := ctxt.RequestCache[apiservice.DoctorID].(int64)
	rd := ctxt.RequestCache[apiservice.RequestData].(*DiagnosisListRequestData)

	// populate new diagnosis set
	set := &common.VisitDiagnosisSet{
		DoctorID:         doctorID,
		VisitID:          visit.PatientVisitID.Int64(),
		Notes:            rd.InternalNote,
		Unsuitable:       rd.CaseManagement.Unsuitable,
		UnsuitableReason: rd.CaseManagement.Reason,
	}

	codes := make(map[string]*common.Version)
	for _, item := range rd.Diagnoses {
		if item.LayoutVersion != nil {
			codes[item.CodeID] = item.LayoutVersion
		}
	}

	layoutVersionIDs, err := d.dataAPI.LayoutVersionIDsForDiagnosisCodes(codes)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	set.Items = make([]*common.VisitDiagnosisItem, len(rd.Diagnoses))
	setItemMapping := make(map[string]*common.VisitDiagnosisItem)
	for i, item := range rd.Diagnoses {
		layoutVersionID := layoutVersionIDs[item.CodeID]

		setItem := &common.VisitDiagnosisItem{
			CodeID: item.CodeID,
		}
		if layoutVersionID > 0 {
			setItem.LayoutVersionID = &layoutVersionID
		}

		set.Items[i] = setItem
		setItemMapping[item.CodeID] = setItem
	}

	if err := d.dataAPI.CreateDiagnosisSet(set); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// lets populate the list of intakes to store
	intakes := make([]api.IntakeInfo, 0, len(rd.Diagnoses))
	for _, inputItem := range rd.Diagnoses {
		if inputItem.Answers == nil {
			continue
		}

		setItem := setItemMapping[inputItem.CodeID]
		layoutVersionID := layoutVersionIDs[inputItem.CodeID]
		answers := make(map[int64][]*common.AnswerIntake)
		for _, item := range inputItem.Answers {
			// enumerate the answers to store from the top level questions as well as the sub questions
			answers[item.QuestionID] = apiservice.PopulateAnswersToStoreForQuestion(
				api.DOCTOR_ROLE,
				item,
				setItem.ID,
				doctorID,
				layoutVersionID)
		}

		intakes = append(intakes, &api.DiagnosisDetailsIntake{
			DoctorID:             doctorID,
			VisitDiagnosisItemID: setItem.ID,
			LVersionID:           layoutVersionID,
			SID:                  inputItem.SessionID,
			SCounter:             inputItem.SessionCounter,
			Intake:               answers,
		})
	}

	if err := d.dataAPI.StoreAnswersForIntakes(intakes); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	if rd.CaseManagement.Unsuitable {
		err = d.dataAPI.ClosePatientVisit(rd.VisitID, common.PVStatusTriaged)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		d.dispatcher.Publish(&patient_visit.PatientVisitMarkedUnsuitableEvent{
			DoctorID:       doctorID,
			PatientID:      visit.PatientID.Int64(),
			CaseID:         visit.PatientCaseID.Int64(),
			PatientVisitID: visit.PatientVisitID.Int64(),
			InternalReason: rd.CaseManagement.Reason,
		})
	} else {
		d.dispatcher.Publish(&patient_visit.DiagnosisModifiedEvent{
			DoctorID:       doctorID,
			PatientID:      visit.PatientID.Int64(),
			PatientVisitID: rd.VisitID,
			PatientCaseID:  visit.PatientCaseID.Int64(),
		})
	}

	apiservice.WriteJSONSuccess(w)
}

func (d *diagnosisListHandler) getDiagnosisList(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	doctorID := ctxt.RequestCache[apiservice.DoctorID].(int64)
	visit := ctxt.RequestCache[apiservice.PatientVisit].(*common.PatientVisit)

	diagnosisSet, err := d.dataAPI.ActiveDiagnosisSet(visit.PatientVisitID.Int64())
	if api.IsErrNotFound(err) && visit.IsFollowup {

		// if we are dealing with a followup visit and there is no active diagnosis
		// set for the followup visit, then pull the diagnosis from the last completed
		// visit. This is to give context to doctors for the diagnosis set
		// for the patient
		visits, err := d.dataAPI.GetVisitsForCase(
			visit.PatientCaseID.Int64(),
			common.TreatedPatientVisitStates())
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

		// sort in descending order of creation date to get the latest visit that was treated
		sort.Reverse(common.ByPatientVisitCreationDate(visits))

		diagnosisSet, err = d.dataAPI.ActiveDiagnosisSet(visits[0].PatientVisitID.Int64())
		if api.IsErrNotFound(err) {
			apiservice.WriteJSON(w, DiagnosisListResponse{})
			return
		} else if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}

	} else if api.IsErrNotFound(err) {
		apiservice.WriteJSON(w, DiagnosisListResponse{})
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	codeIDs := make([]string, len(diagnosisSet.Items))
	layoutVersionIDs := make([]int64, 0, len(diagnosisSet.Items))
	for i, item := range diagnosisSet.Items {
		codeIDs[i] = item.CodeID
		if item.LayoutVersionID != nil {
			layoutVersionIDs = append(layoutVersionIDs, *item.LayoutVersionID)
		}
	}

	diagnosisMap, err := d.diagnosisAPI.DiagnosisForCodeIDs(codeIDs)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	diagnosisDetailsIntakes, err := d.dataAPI.DiagnosisDetailsIntake(layoutVersionIDs, diagnosis.DetailTypes)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	activeLayoutVersions, err := d.dataAPI.DetailsIntakeVersionForDiagnoses(codeIDs)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// lets get the answers for the items in the diagnosis set that have a layout associated with them
	answersForDiagnosisDetails := make(map[string]map[int64][]common.Answer)
	for _, item := range diagnosisSet.Items {
		if item.LayoutVersionID != nil {
			intake := diagnosisDetailsIntakes[*item.LayoutVersionID]
			questionIDs := questionIDsFromIntake(intake)
			answersForDiagnosisDetails[item.CodeID], err = d.dataAPI.AnswersForQuestions(
				questionIDs, &api.DiagnosisDetailsIntake{
					DoctorID:             doctorID,
					VisitDiagnosisItemID: item.ID,
					LVersionID:           *item.LayoutVersionID,
				})
			if err != nil {
				apiservice.WriteError(err, w, r)
				return
			}
		}
	}

	// lets craft the response
	response := DiagnosisListResponse{
		Notes:     diagnosisSet.Notes,
		Diagnoses: make([]*DiagnosisOutputItem, len(diagnosisSet.Items)),
		CaseManagement: CaseManagementItem{
			Unsuitable: diagnosisSet.Unsuitable,
			Reason:     diagnosisSet.UnsuitableReason,
		},
	}

	for i, item := range diagnosisSet.Items {
		diagnosisInfo := diagnosisMap[item.CodeID]
		activeLayoutVersion := activeLayoutVersions[item.CodeID]

		outputItem := &DiagnosisOutputItem{
			CodeID:              item.CodeID,
			Code:                diagnosisInfo.Code,
			Title:               diagnosisInfo.Description,
			HasDetails:          activeLayoutVersion != nil,
			LatestLayoutVersion: activeLayoutVersion,
		}
		response.Diagnoses[i] = outputItem

		if item.LayoutVersionID != nil {
			intake := diagnosisDetailsIntakes[*item.LayoutVersionID]
			outputItem.LayoutVersion = intake.Version
			outputItem.Questions = intake.Layout.(*diagnosis.QuestionIntake).Questions()
		}

		answers := answersForDiagnosisDetails[item.CodeID]
		outputItem.Answers = apiservice.TransformAnswers(answers)
	}

	apiservice.WriteJSON(w, response)
}

func questionIDsFromIntake(intake *common.DiagnosisDetailsIntake) []int64 {
	questions := intake.Layout.(*diagnosis.QuestionIntake).Questions()
	questionIDs := make([]int64, len(questions))
	for i, question := range questions {
		questionIDs[i] = question.QuestionID
	}
	return questionIDs
}
