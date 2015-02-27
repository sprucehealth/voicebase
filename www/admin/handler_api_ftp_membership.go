package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/responses"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type ftpMembershipHandler struct {
	dataAPI api.DataAPI
}

type ftpMembershipGETResponse struct {
	Name        string                                       `json:"name"`
	Memberships []*responses.FavoriteTreatmentPlanMembership `json:"memberships"`
}

type ftpMembershipPOSTRequest struct {
	DoctorID   int64  `json:"doctor_id,string"`
	PathwayTag string `json:"pathway_tag"`
	PathwayID  int64
}

type ftpMembershipDELETERequest struct {
	DoctorID   int64  `json:"doctor_id,string"`
	PathwayTag string `json:"pathway_tag"`
	PathwayID  int64
}

func NewFTPMembershipHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&ftpMembershipHandler{dataAPI: dataAPI}, []string{"GET", "POST", "DELETE"})
}

func (h *ftpMembershipHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ftpID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		www.APINotFound(w, r)
		return
	}

	switch r.Method {
	case "GET":
		h.serveGET(w, r, ftpID)
	case "POST":
		request, err := h.parsePOSTRequest(r)
		if err != nil {
			www.BadRequestError(w, r, err)
			return
		}
		h.servePOST(w, r, ftpID, request)
	case "DELETE":
		request, err := h.parseDELETERequest(r)
		if err != nil {
			www.BadRequestError(w, r, err)
			return
		}
		h.serveDELETE(w, r, ftpID, request)
	}
}

func (h *ftpMembershipHandler) parsePOSTRequest(r *http.Request) (*ftpMembershipPOSTRequest, error) {
	rd := &ftpMembershipPOSTRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse body: %s", err)
	}

	if rd.DoctorID == 0 || rd.PathwayTag == "" {
		return nil, fmt.Errorf("insufficent parameters supplied to form complete request body")
	}

	pathway, err := h.dataAPI.PathwayForTag(rd.PathwayTag, api.PONone)
	if err != nil {
		return nil, err
	}
	rd.PathwayID = pathway.ID
	return rd, nil
}

func (h *ftpMembershipHandler) parseDELETERequest(r *http.Request) (*ftpMembershipDELETERequest, error) {
	rd := &ftpMembershipDELETERequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse body: %s", err)
	}

	if rd.DoctorID == 0 || rd.PathwayTag == "" {
		return nil, fmt.Errorf("insufficent parameters supplied to form complete request body")
	}

	pathway, err := h.dataAPI.PathwayForTag(rd.PathwayTag, api.PONone)
	if err != nil {
		return nil, err
	}
	rd.PathwayID = pathway.ID
	return rd, nil
}

func (h *ftpMembershipHandler) servePOST(w http.ResponseWriter, r *http.Request, ftpID int64, rd *ftpMembershipPOSTRequest) {
	_, err := h.dataAPI.CreateFTPMembership(ftpID, rd.DoctorID, rd.PathwayID)
	if err != nil {
		www.APIInternalError(w, r, err)
	}
}

func (h *ftpMembershipHandler) serveDELETE(w http.ResponseWriter, r *http.Request, ftpID int64, rd *ftpMembershipDELETERequest) {
	_, err := h.dataAPI.DeleteFTPMembership(ftpID, rd.DoctorID, rd.PathwayID)
	if err != nil {
		www.APIInternalError(w, r, err)
	}
}

type FullName struct {
	FirstName string
	LastName  string
}

func (h *ftpMembershipHandler) serveGET(w http.ResponseWriter, r *http.Request, ftpID int64) {
	ftp, err := h.dataAPI.FavoriteTreatmentPlan(ftpID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	memberships, err := h.dataAPI.FTPMemberships(ftpID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	doctorIDs := make([]int64, len(memberships))
	for i, m := range memberships {
		doctorIDs[i] = m.DoctorID
	}

	doctors, err := h.dataAPI.Doctors(doctorIDs)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	doctorMap := make(map[int64]FullName)
	for _, d := range doctors {
		doctorMap[d.DoctorID.Int64()] = FullName{
			FirstName: d.FirstName,
			LastName:  d.LastName,
		}
	}

	response := &ftpMembershipGETResponse{Name: ftp.Name, Memberships: make([]*responses.FavoriteTreatmentPlanMembership, len(memberships))}
	for i, m := range memberships {
		d, ok := doctorMap[m.DoctorID]
		if !ok {
			if err != nil {
				www.APIInternalError(w, r, fmt.Errorf("Found membership for doctor that doesn't exist in the system - Doctor ID: %d", m.DoctorID))
				return
			}
		}
		response.Memberships[i] = &responses.FavoriteTreatmentPlanMembership{
			DoctorID:  m.DoctorID,
			FirstName: d.FirstName,
			LastName:  d.LastName,
			PathwayID: m.ClinicalPathwayID,
		}
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}
