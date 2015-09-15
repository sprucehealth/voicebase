package patient_case

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/patient_case/model"
	"github.com/sprucehealth/backend/patient_case/response"
	"github.com/sprucehealth/backend/www"
	"github.com/sprucehealth/schema"
	"golang.org/x/net/context"
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

func NewPatientCaseNoteHandler(dataAPI api.DataAPI, apiDomain string) httputil.ContextHandler {
	return httputil.SupportedMethods(
		apiservice.RequestCacheHandler(
			apiservice.MethodGranularAuthorizationRequired(&patientCaseNoteHandler{dataAPI: dataAPI, apiDomain: apiDomain})),
		httputil.Get, httputil.Put, httputil.Post, httputil.Delete)
}

func (h *patientCaseNoteHandler) parseDELETERequest(ctx context.Context, r *http.Request) (*PatientCaseNoteDELETERequest, error) {
	rd := &PatientCaseNoteDELETERequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	return rd, nil
}

func (h *patientCaseNoteHandler) parseGETRequest(ctx context.Context, r *http.Request) (*PatientCaseNoteGETRequest, error) {
	rd := &PatientCaseNoteGETRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}
	if err := schema.NewDecoder().Decode(rd, r.Form); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	return rd, nil
}

func (h *patientCaseNoteHandler) parsePOSTRequest(ctx context.Context, r *http.Request) (*PatientCaseNotePOSTRequest, error) {
	rd := &PatientCaseNotePOSTRequest{}
	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if rd.NoteText == "" || rd.CaseID == 0 {
		return nil, errors.New("case_id, note_text required")
	}

	return rd, nil
}

func (h *patientCaseNoteHandler) parsePUTRequest(ctx context.Context, r *http.Request) (*PatientCaseNotePUTRequest, error) {
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
func (h *patientCaseNoteHandler) IsDELETEAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	account := apiservice.MustCtxAccount(ctx)
	req, err := h.parseDELETERequest(ctx, r)
	if err != nil {
		return false, apiservice.NewBadRequestError(err)
	}
	requestCache := apiservice.MustCtxCache(ctx)
	requestCache[apiservice.CKRequestData] = req
	return h.isAccountAuthorized(ctx, account.ID, req.ID, 0, account.Role, PCNHNoteOwner|PCNHCaseRead)
}

// Assert that the person has access to the specified case
func (h *patientCaseNoteHandler) IsGETAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	account := apiservice.MustCtxAccount(ctx)
	req, err := h.parseGETRequest(ctx, r)
	if err != nil {
		return false, apiservice.NewBadRequestError(err)
	}
	requestCache := apiservice.MustCtxCache(ctx)
	requestCache[apiservice.CKRequestData] = req
	return h.isAccountAuthorized(ctx, account.ID, 0, req.CaseID, account.Role, PCNHCaseRead)
}

// Assert that the person has access to the specified case
func (h *patientCaseNoteHandler) IsPOSTAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	account := apiservice.MustCtxAccount(ctx)
	req, err := h.parsePOSTRequest(ctx, r)
	if err != nil {
		return false, apiservice.NewBadRequestError(err)
	}
	requestCache := apiservice.MustCtxCache(ctx)
	requestCache[apiservice.CKRequestData] = req
	return h.isAccountAuthorized(ctx, account.ID, 0, req.CaseID, account.Role, PCNHCaseRead)
}

// Assert that the person modifying the note is the owner and has access to the specified case
func (h *patientCaseNoteHandler) IsPUTAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	account := apiservice.MustCtxAccount(ctx)
	req, err := h.parsePUTRequest(ctx, r)
	if err != nil {
		return false, apiservice.NewBadRequestError(err)
	}
	requestCache := apiservice.MustCtxCache(ctx)
	requestCache[apiservice.CKRequestData] = req
	return h.isAccountAuthorized(ctx, account.ID, req.ID, 0, account.Role, PCNHNoteOwner|PCNHCaseRead)
}

func (h *patientCaseNoteHandler) isAccountAuthorized(ctx context.Context, accountID, noteID, caseID int64, role string, requiredAccess PCNHRequiredAccess) (bool, error) {
	doctorID, err := h.dataAPI.GetDoctorIDFromAccountID(accountID)
	if err != nil {
		return false, err
	}
	requestCache := apiservice.MustCtxCache(ctx)
	requestCache[apiservice.CKDoctorID] = doctorID

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
		if hasRead, err := apiservice.DoctorHasAccessToCase(ctx, doctorID, caseID, role, apiservice.ReadAccessRequired, h.dataAPI); err != nil {
			return false, err
		} else if hasRead {
			requiredAccess = requiredAccess ^ PCNHCaseRead
		}
	}
	if requiredAccess.Has(PCNHCaseWrite) {
		if hasRead, err := apiservice.DoctorHasAccessToCase(ctx, doctorID, caseID, role, apiservice.WriteAccessRequired, h.dataAPI); err != nil {
			return false, err
		} else if hasRead {
			requiredAccess = requiredAccess ^ PCNHCaseWrite
		}
	}

	return requiredAccess == 0, nil
}

func (h *patientCaseNoteHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	requestCache := apiservice.MustCtxCache(ctx)
	switch r.Method {
	case httputil.Delete:
		h.serveDELETE(ctx, w, r, requestCache[apiservice.CKRequestData].(*PatientCaseNoteDELETERequest))
	case httputil.Get:
		h.serveGET(ctx, w, r, requestCache[apiservice.CKRequestData].(*PatientCaseNoteGETRequest))
	case httputil.Post:
		h.servePOST(ctx, w, r, requestCache[apiservice.CKRequestData].(*PatientCaseNotePOSTRequest))
	case httputil.Put:
		h.servePUT(ctx, w, r, requestCache[apiservice.CKRequestData].(*PatientCaseNotePUTRequest))
	}
}

func (h *patientCaseNoteHandler) serveGET(ctx context.Context, w http.ResponseWriter, r *http.Request, req *PatientCaseNoteGETRequest) {
	caseNotes, err := h.dataAPI.PatientCaseNotes([]int64{req.CaseID})
	if api.IsErrNotFound(err) {
		httputil.JSONResponse(w, http.StatusOK, &PatientCaseNoteGETResponse{})
		return
	} else if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	// Don't assume size here since we can't know it just from the size of the notes list, sacrifice memory for compute here and double track the ids
	doctorsLookup := make(map[int64]struct{})
	var doctorIDs []int64
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
		apiservice.WriteError(ctx, err, w, r)
		return
	}

	doctorsByID := make(map[int64]*common.Doctor, len(doctors))
	for _, d := range doctors {
		doctorsByID[d.ID.Int64()] = d
	}

	for i := range respNotes {
		d, ok := doctorsByID[respNotes[i].AuthorDoctorID]
		if !ok {
			apiservice.WriteError(ctx, fmt.Errorf("Couldn't map case note author doctor ID %d to a doctor record", respNotes[i].AuthorDoctorID), w, r)
			return
		}
		response.AddPatientCaseNoteOptionalData(respNotes[i], response.NewPatientCaseNoteOptionalData(d, h.apiDomain))
	}

	httputil.JSONResponse(w, http.StatusOK, &PatientCaseNoteGETResponse{
		PatientCaseNotes: respNotes,
	})
}

func (h *patientCaseNoteHandler) servePOST(ctx context.Context, w http.ResponseWriter, r *http.Request, req *PatientCaseNotePOSTRequest) {
	requestCache := apiservice.MustCtxCache(ctx)
	id, err := h.dataAPI.InsertPatientCaseNote(&model.PatientCaseNote{
		CaseID:         req.CaseID,
		AuthorDoctorID: requestCache[apiservice.CKDoctorID].(int64),
		NoteText:       req.NoteText,
	})
	if err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, &PatientCaseNotePOSTResponse{
		ID: id,
	})
}

func (h *patientCaseNoteHandler) servePUT(ctx context.Context, w http.ResponseWriter, r *http.Request, req *PatientCaseNotePUTRequest) {
	if _, err := h.dataAPI.UpdatePatientCaseNote(&model.PatientCaseNoteUpdate{
		ID:       req.ID,
		NoteText: req.NoteText,
	}); err != nil {
		apiservice.WriteError(ctx, err, w, r)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}

func (h *patientCaseNoteHandler) serveDELETE(ctx context.Context, w http.ResponseWriter, r *http.Request, req *PatientCaseNoteDELETERequest) {
	if _, err := h.dataAPI.DeletePatientCaseNote(req.ID); err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, struct{}{})
}
