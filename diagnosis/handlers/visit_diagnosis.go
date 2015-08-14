package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/patient_visit"
	"golang.org/x/net/context"
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

func NewDiagnosisListHandler(dataAPI api.DataAPI, diagnosisAPI diagnosis.API, dispatcher *dispatch.Dispatcher) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.SupportedRoles(
			apiservice.RequestCacheHandler(
				apiservice.AuthorizationRequired(
					&diagnosisListHandler{
						dataAPI:      dataAPI,
						diagnosisAPI: diagnosisAPI,
						dispatcher:   dispatcher,
					})),
			api.RoleDoctor, api.RoleCC),
		httputil.Get, httputil.Put)
}

func (d *diagnosisListHandler) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	requestCache := apiservice.MustCtxCache(ctx)
	account := apiservice.MustCtxAccount(ctx)

	rd := &DiagnosisListRequestData{}
	if err := apiservice.DecodeRequestData(rd, r); err != nil {
		return false, apiservice.NewValidationError(err.Error())
	} else if rd.VisitID == 0 {
		return false, apiservice.NewValidationError("patient_visit_id required")
	}
	requestCache[apiservice.CKRequestData] = rd

	doctorID, err := d.dataAPI.GetDoctorIDFromAccountID(account.ID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKDoctorID] = doctorID

	patientVisit, err := d.dataAPI.GetPatientVisitFromID(rd.VisitID)
	if err != nil {
		return false, err
	}
	requestCache[apiservice.CKPatientVisit] = patientVisit

	if err := apiservice.ValidateAccessToPatientCase(
		r.Method,
		account.Role,
		doctorID,
		patientVisit.PatientID,
		patientVisit.PatientCaseID.Int64(),
		d.dataAPI); err != nil {
		return false, err
	}

	return true, nil
}

func (d *diagnosisListHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.getDiagnosisList(ctx, w, r)
	case "PUT":
		d.putDiagnosisList(ctx, w, r)
	}
}

func (d *diagnosisListHandler) putDiagnosisList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	visit := requestCache[apiservice.CKPatientVisit].(*common.PatientVisit)
	doctorID := requestCache[apiservice.CKDoctorID].(int64)
	rd := requestCache[apiservice.CKRequestData].(*DiagnosisListRequestData)

	// populate new diagnosis set
	set := &common.VisitDiagnosisSet{
		DoctorID:         doctorID,
		VisitID:          visit.ID.Int64(),
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
		apiservice.WriteError(ctx, err, w, r)
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
		apiservice.WriteError(ctx, err, w, r)
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
				api.RoleDoctor,
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
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	if rd.CaseManagement.Unsuitable {
		_, err = d.dataAPI.UpdatePatientVisit(rd.VisitID, &api.PatientVisitUpdate{
			Status:     ptr.String(common.PVStatusTriaged),
			ClosedDate: ptr.Time(time.Now()),
		})
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		d.dispatcher.Publish(&patient_visit.PatientVisitMarkedUnsuitableEvent{
			DoctorID:       doctorID,
			PatientID:      visit.PatientID,
			CaseID:         visit.PatientCaseID.Int64(),
			PatientVisitID: visit.ID.Int64(),
			Reason:         rd.CaseManagement.Reason,
		})
	} else {
		// move the visit back into the reviewing state if it was previously triaged
		// but now was being modified.
		if visit.Status == common.PVStatusTriaged {
			_, err = d.dataAPI.UpdatePatientVisit(rd.VisitID, &api.PatientVisitUpdate{
				Status: ptr.String(common.PVStatusReviewing),
			})
			if err != nil {
				apiservice.WriteError(ctx, err, w, r)
				return
			}
		}

		d.dispatcher.Publish(&patient_visit.DiagnosisModifiedEvent{
			DoctorID:       doctorID,
			PatientID:      visit.PatientID,
			PatientVisitID: rd.VisitID,
			PatientCaseID:  visit.PatientCaseID.Int64(),
		})
	}

	apiservice.WriteJSONSuccess(w)
}

func (d *diagnosisListHandler) getDiagnosisList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	doctorID := requestCache[apiservice.CKDoctorID].(int64)
	visit := requestCache[apiservice.CKPatientVisit].(*common.PatientVisit)

	diagnosisSet, err := d.dataAPI.ActiveDiagnosisSet(visit.ID.Int64())
	if api.IsErrNotFound(err) && visit.IsFollowup {

		// if we are dealing with a followup visit and there is no active diagnosis
		// set for the followup visit, then pull the diagnosis from the last completed
		// visit. This is to give context to doctors for the diagnosis set
		// for the patient
		visits, err := d.dataAPI.GetVisitsForCase(
			visit.PatientCaseID.Int64(),
			common.TreatedPatientVisitStates())
		if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

		// sort in descending order of creation date to get the latest visit that was treated
		sort.Sort(sort.Reverse(common.ByPatientVisitCreationDate(visits)))

		diagnosisSet, err = d.dataAPI.ActiveDiagnosisSet(visits[0].ID.Int64())
		if api.IsErrNotFound(err) {
			httputil.JSONResponse(w, http.StatusOK, DiagnosisListResponse{})
			return
		} else if err != nil {
			apiservice.WriteError(ctx, err, w, r)
			return
		}

	} else if api.IsErrNotFound(err) {
		httputil.JSONResponse(w, http.StatusOK, DiagnosisListResponse{})
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
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
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	diagnosisDetailsIntakes, err := d.dataAPI.DiagnosisDetailsIntake(layoutVersionIDs, diagnosis.DetailTypes)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	activeLayoutVersions, err := d.dataAPI.DetailsIntakeVersionForDiagnoses(codeIDs)
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
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
				apiservice.WriteError(ctx, err, w, r)
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

	httputil.JSONResponse(w, http.StatusOK, response)
}

func questionIDsFromIntake(intake *common.DiagnosisDetailsIntake) []int64 {
	questions := intake.Layout.(*diagnosis.QuestionIntake).Questions()
	questionIDs := make([]int64, len(questions))
	for i, question := range questions {
		questionIDs[i] = question.QuestionID
	}
	return questionIDs
}
