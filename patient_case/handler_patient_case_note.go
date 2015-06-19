package patient_case

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/schema"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/patient_case/model"
	"github.com/sprucehealth/backend/patient_case/response"
	"github.com/sprucehealth/backend/www"

	"github.com/sprucehealth/backend/libs/httputil"
)

type PCNHRequiredAccess int

const (
	PCNHNoteOwner PCNHRequiredAccess = 1 << iota
	PCNHCaseRead
	PCNHCaseWrite
)

func (ra PCNHRequiredAccess) Has(a PCNHRequiredAccess) bool {
	return (ra & a) != 0
}

type PatientCaseNoteGETRequest struct {
	CaseID int64 `schema:"case_id,required"`
}

type PatientCaseNoteGETResponse struct {
	PatientCaseNotes []*response.PatientCaseNote `json:"case_notes"`
}

type PatientCaseNotePOSTRequest struct {
	CaseID   int64  `json:"case_id,string"`
	NoteText string `json:"note_text"`
}

type PatientCaseNotePOSTResponse struct {
	ID int64 `json:"id,string"`
}

type PatientCaseNotePUTRequest struct {
	ID       int64  `json:"id,string"`
	NoteText string `json:"note_text"`
}

type PatientCaseNoteDELETERequest struct {
	ID int64 `schema:"id,required"`
}

type patientCaseNoteHandler struct {
	apiDomain string
	dataAPI   api.DataAPI
}

func NewPatientCaseNoteHandler(dataAPI api.DataAPI, apiDomain string) http.Handler {
	return httputil.SupportedMethods(apiservice.MethodGranularAuthorizationRequired(&patientCaseNoteHandler{dataAPI: dataAPI, apiDomain: apiDomain}), httputil.Get, httputil.Put, httputil.Post, httputil.Delete)
}

func (h *patientCaseNoteHandler) parseDELETERequest(r *http.Request) (*PatientCaseNoteDELETERequest, error) {
	rd := &PatientCaseNoteDELETERequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *patientCaseNoteHandler) parseGETRequest(r *http.Request) (*PatientCaseNoteGETRequest, error) {
	rd := &PatientCaseNoteGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *patientCaseNoteHandler) parsePOSTRequest(r *http.Request) (*PatientCaseNotePOSTRequest, error) {
	rd := &PatientCaseNotePOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.NoteText == "" || rd.CaseID == 0 {
		return nil, errors.New("case_id, note_text required")
	}

	return rd, nil
}

func (h *patientCaseNoteHandler) parsePUTRequest(r *http.Request) (*PatientCaseNotePUTRequest, error) {
	rd := &PatientCaseNotePUTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.ID == 0 || rd.NoteText == "" {
		return nil, errors.New("id, note_text required")
	}
	return rd, nil
}

// Assert that the person deleting the note is the owner and has access to the specified case
func (h *patientCaseNoteHandler) IsDELETEAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	req, err := h.parseDELETERequest(r)
	if err != nil {
		return false, apiservice.NewBadRequestError(err)
	}
	ctxt.RequestCache[apiservice.RequestData] = req
	return h.isAccountAuthorized(ctxt.AccountID, req.ID, 0, ctxt.Role, PCNHNoteOwner|PCNHCaseRead, ctxt)
}

// Assert that the person has access to the specified case
func (h *patientCaseNoteHandler) IsGETAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	req, err := h.parseGETRequest(r)
	if err != nil {
		return false, apiservice.NewBadRequestError(err)
	}
	ctxt.RequestCache[apiservice.RequestData] = req
	return h.isAccountAuthorized(ctxt.AccountID, 0, req.CaseID, ctxt.Role, PCNHCaseRead, ctxt)
}

// Assert that the person has access to the specified case
func (h *patientCaseNoteHandler) IsPOSTAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	req, err := h.parsePOSTRequest(r)
	if err != nil {
		return false, apiservice.NewBadRequestError(err)
	}
	ctxt.RequestCache[apiservice.RequestData] = req
	return h.isAccountAuthorized(ctxt.AccountID, 0, req.CaseID, ctxt.Role, PCNHCaseRead, ctxt)
}

// Assert that the person modifying the note is the owner and has access to the specified case
func (h *patientCaseNoteHandler) IsPUTAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	req, err := h.parsePUTRequest(r)
	if err != nil {
		return false, apiservice.NewBadRequestError(err)
	}
	ctxt.RequestCache[apiservice.RequestData] = req
	return h.isAccountAuthorized(ctxt.AccountID, req.ID, 0, ctxt.Role, PCNHNoteOwner|PCNHCaseRead, ctxt)
}

func (h *patientCaseNoteHandler) isAccountAuthorized(accountID, noteID, caseID int64, role string, requiredAccess PCNHRequiredAccess, ctxt *apiservice.Context) (bool, error) {
	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(ctxt.AccountID)
	if err != nil {
		return false, err
	}
	ctxt.RequestCache[apiservice.DoctorID] = doctorID

	if requiredAccess.Has(PCNHNoteOwner) {
		note, err := h.dataAPI.PatientCaseNote(noteID)
		if api.IsErrNotFound(err) {
			return false, apiservice.NewBadRequestError(err)
		}

		if !(note.AuthorDoctorID == doctorID) {
			return false, nil
		}
		requiredAccess = requiredAccess ^ PCNHNoteOwner
		caseID = note.CaseID
	}

	if requiredAccess.Has(PCNHCaseRead) {
		if hasRead, err := apiservice.DoctorHasAccessToCase(doctorID, caseID, role, apiservice.ReadAccessRequired, h.dataAPI, ctxt); err != nil {
			return false, err
		} else if hasRead {
			requiredAccess = requiredAccess ^ PCNHCaseRead
		}
	}
	if requiredAccess.Has(PCNHCaseWrite) {
		if hasRead, err := apiservice.DoctorHasAccessToCase(doctorID, caseID, role, apiservice.WriteAccessRequired, h.dataAPI, ctxt); err != nil {
			return false, err
		} else if hasRead {
			requiredAccess = requiredAccess ^ PCNHCaseWrite
		}
	}

	return requiredAccess == 0, nil
}

func (h *patientCaseNoteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctxt := apiservice.GetContext(r)
	switch r.Method {
	case httputil.Delete:
		h.serveDELETE(w, r, ctxt.RequestCache[apiservice.RequestData].(*PatientCaseNoteDELETERequest))
	case httputil.Get:
		h.serveGET(w, r, ctxt.RequestCache[apiservice.RequestData].(*PatientCaseNoteGETRequest))
	case httputil.Post:
		h.servePOST(w, r, ctxt.RequestCache[apiservice.RequestData].(*PatientCaseNotePOSTRequest))
	case httputil.Put:
		h.servePUT(w, r, ctxt.RequestCache[apiservice.RequestData].(*PatientCaseNotePUTRequest))
	}
}

func (h *patientCaseNoteHandler) serveGET(w http.ResponseWriter, r *http.Request, req *PatientCaseNoteGETRequest) {
	caseNotes, err := h.dataAPI.PatientCaseNotes([]int64{req.CaseID})
	if api.IsErrNotFound(err) {
		httputil.JSONResponse(w, http.StatusOK, &PatientCaseNoteGETResponse{})
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	// Don't assume size here since we can't know it just from the size of the notes list, sacrifice memory for compute here and double track the ids
	doctorsLookup := make(map[int64]struct{})
	doctorIDs := make([]int64, 0)
	respNotes := make([]*response.PatientCaseNote, len(caseNotes[req.CaseID]))
	for i, n := range caseNotes[req.CaseID] {
		respNotes[i] = response.TransformPatientCaseNote(n)
		if _, ok := doctorsLookup[n.AuthorDoctorID]; !ok {
			doctorsLookup[n.AuthorDoctorID] = struct{}{}
			doctorIDs = append(doctorIDs, n.AuthorDoctorID)
		}
	}

	// Query our involved doctors by and map them to IDs so we can build out the optional info
	doctors, err := h.dataAPI.Doctors(doctorIDs)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	doctorsByID := make(map[int64]*common.Doctor, len(doctors))
	for _, d := range doctors {
		doctorsByID[d.ID.Int64()] = d
	}

	for i := range respNotes {
		d, ok := doctorsByID[respNotes[i].AuthorDoctorID]
		if !ok {
			apiservice.WriteError(fmt.Errorf("Couldn't map case note author doctor ID %d to a doctor record", respNotes[i].AuthorDoctorID), w, r)
			return
		}
		response.AddPatientCaseNoteOptionalData(respNotes[i], response.NewPatientCaseNoteOptionalData(d, h.apiDomain))
	}

	httputil.JSONResponse(w, http.StatusOK, &PatientCaseNoteGETResponse{
		PatientCaseNotes: respNotes,
	})
}

func (h *patientCaseNoteHandler) servePOST(w http.ResponseWriter, r *http.Request, req *PatientCaseNotePOSTRequest) {
	ctxt := apiservice.GetContext(r)
	id, err := h.dataAPI.InsertPatientCaseNote(&model.PatientCaseNote{
		CaseID:         req.CaseID,
		AuthorDoctorID: ctxt.RequestCache[apiservice.DoctorID].(int64),
		NoteText:       req.NoteText,
	})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &PatientCaseNotePOSTResponse{
		ID: id,
	})
}

func (h *patientCaseNoteHandler) servePUT(w http.ResponseWriter, r *http.Request, req *PatientCaseNotePUTRequest) {
	if _, err := h.dataAPI.UpdatePatientCaseNote(&model.PatientCaseNoteUpdate{
		ID:       req.ID,
		NoteText: req.NoteText,
	}); err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

func (h *patientCaseNoteHandler) serveDELETE(w http.ResponseWriter, r *http.Request, req *PatientCaseNoteDELETERequest) {
	if _, err := h.dataAPI.DeletePatientCaseNote(req.ID); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}
