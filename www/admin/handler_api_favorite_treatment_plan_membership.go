package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sprucehealth/backend/common"
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

type batchFTPMembershipPOSTRequest struct {
	Requests []*ftpMembershipPOSTRequest `json:"requests"`
}

type ftpMembershipDELETERequest struct {
	DoctorID   int64  `json:"doctor_id,string"`
	PathwayTag string `json:"pathway_tag"`
	PathwayID  int64
}

func NewFTPMembershipHandler(dataAPI api.DataAPI) http.Handler {
	return httputil.SupportedMethods(&ftpMembershipHandler{dataAPI: dataAPI},
		httputil.Get, httputil.Post, httputil.Delete)
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
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePOST(w, r, ftpID, request)
	case "DELETE":
		request, err := h.parseDELETERequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.serveDELETE(w, r, ftpID, request)
	}
}

func (h *ftpMembershipHandler) parsePOSTRequest(r *http.Request) (*batchFTPMembershipPOSTRequest, error) {
	var err error
	rd := &batchFTPMembershipPOSTRequest{}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("Unable to parse input parameters: %s", err)
	}

	if err = json.NewDecoder(r.Body).Decode(&rd); err != nil {
		return nil, fmt.Errorf("Unable to parse body: %v", err)
	}

	pathwayMap := make(map[string]*common.Pathway)
	for _, v := range rd.Requests {
		if v.DoctorID == 0 || v.PathwayTag == "" {
			return nil, fmt.Errorf("insufficent parameters supplied to form complete request body")
		}
		pathway, ok := pathwayMap[v.PathwayTag]
		if !ok {
			pathway, err = h.dataAPI.PathwayForTag(v.PathwayTag, api.PONone)
			if err != nil {
				return nil, err
			}
		}
		v.PathwayID = pathway.ID
	}

	return rd, nil
}

func (h *ftpMembershipHandler) parseDELETERequest(r *http.Request) (*ftpMembershipDELETERequest, error) {
	rd := &ftpMembershipDELETERequest{}
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

func (h *ftpMembershipHandler) servePOST(w http.ResponseWriter, r *http.Request, ftpID int64, rd *batchFTPMembershipPOSTRequest) {
	memberships := make([]*common.FTPMembership, len(rd.Requests))
	for i, v := range rd.Requests {
		memberships[i] = &common.FTPMembership{
			DoctorFavoritePlanID: ftpID,
			DoctorID:             v.DoctorID,
			ClinicalPathwayID:    v.PathwayID,
		}
	}
	if err := h.dataAPI.CreateFTPMemberships(memberships); err != nil {
		www.APIInternalError(w, r, err)
		return
	}
	httputil.JSONResponse(w, http.StatusOK, true)
}

func (h *ftpMembershipHandler) serveDELETE(w http.ResponseWriter, r *http.Request, ftpID int64, rd *ftpMembershipDELETERequest) {
	_, err := h.dataAPI.DeleteFTPMembership(ftpID, rd.DoctorID, rd.PathwayID)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	httputil.JSONResponse(w, http.StatusOK, true)
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
		doctorMap[d.ID.Int64()] = FullName{
			FirstName: d.FirstName,
			LastName:  d.LastName,
		}
	}

	response := &ftpMembershipGETResponse{Name: ftp.Name, Memberships: make([]*responses.FavoriteTreatmentPlanMembership, len(memberships))}
	pathwayMap := make(map[int64]*common.Pathway)
	for i, m := range memberships {
		d, ok := doctorMap[m.DoctorID]
		if !ok {
			if err != nil {
				www.APIInternalError(w, r, fmt.Errorf("Found membership for doctor that doesn't exist in the system - Doctor ID: %d", m.DoctorID))
				return
			}
		}
		pathway, ok := pathwayMap[m.ClinicalPathwayID]
		if !ok {
			pathway, err = h.dataAPI.Pathway(m.ClinicalPathwayID, api.PONone)
			if err != nil {
				www.APIInternalError(w, r, fmt.Errorf("Unable to find clinical pathway associated with membership - Membership ID: %d, Pathway ID: %d", m.ID, m.ClinicalPathwayID))
				return
			}
			pathwayMap[m.ClinicalPathwayID] = pathway
		}
		response.Memberships[i] = &responses.FavoriteTreatmentPlanMembership{
			ID:                      m.ID,
			DoctorID:                m.DoctorID,
			FavoriteTreatmentPlanID: m.DoctorFavoritePlanID,
			FirstName:               d.FirstName,
			LastName:                d.LastName,
			PathwayID:               m.ClinicalPathwayID,
			PathwayName:             pathway.Name,
			PathwayTag:              pathway.Tag,
		}
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}
