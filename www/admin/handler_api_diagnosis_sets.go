package admin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

type diagnosisSetsHandler struct {
	dataAPI      api.DataAPI
	diagnosisAPI diagnosis.API
}

func NewDiagnosisSetsHandler(dataAPI api.DataAPI, diagnosisAPI diagnosis.API) http.Handler {
	return httputil.SupportedMethods(&diagnosisSetsHandler{
		dataAPI:      dataAPI,
		diagnosisAPI: diagnosisAPI,
	}, httputil.Get, httputil.Patch)
}

type diagnosisItem struct {
	Code   string `json:"code"`
	CodeID string `json:"codeID"`
	Name   string `json:"name"`
}

type diagnosisSetsResponse struct {
	Title     string           `json:"title"`
	Diagnoses []*diagnosisItem `json:"items"`
}

type diagnosisSetUpdateRequest struct {
	PathwayTag string   `json:"pathwayTag"`
	Delete     []string `json:"delete,omitempty"`
	Title      string   `json:"title,omitempty"`
	Create     []string `json:"create,omitempty"`
}

func (d *diagnosisSetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// account := context.Get(r, www.CKAccount).(*common.Account)
	// audit.LogAction(account.ID, "AdminAPI", "ListDiagnosisSets", nil)

	switch r.Method {
	case httputil.Get:
		pathwayTag := r.FormValue("pathway_tag")
		if pathwayTag == "" {
			www.BadRequestError(w, r, errors.New("pathway_tag required"))
			return
		}
		d.get(pathwayTag, w, r)
	case httputil.Patch:
		var rd diagnosisSetUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&rd); err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		} else if rd.PathwayTag == "" {
			www.APIBadRequestError(w, r, "missing pathwayTag")
			return
		}

		d.patch(&rd, w, r)
	}
}

func (d *diagnosisSetsHandler) patch(rd *diagnosisSetUpdateRequest, w http.ResponseWriter, r *http.Request) {
	if err := d.dataAPI.PatchCommonDiagnosisSet(rd.PathwayTag, &api.DiagnosisSetPatch{
		Title:  &rd.Title,
		Delete: rd.Delete,
		Create: rd.Create,
	}); err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	d.get(rd.PathwayTag, w, r)
}

func (d *diagnosisSetsHandler) get(pathwayTag string, w http.ResponseWriter, r *http.Request) {

	title, diagnosisCodeIDs, err := d.dataAPI.CommonDiagnosisSet(pathwayTag)
	if !api.IsErrNotFound(err) && err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	diagnosisMap, err := d.diagnosisAPI.DiagnosisForCodeIDs(diagnosisCodeIDs)
	if err != nil {
		www.InternalServerError(w, r, err)
		return
	}

	response := diagnosisSetsResponse{
		Title:     title,
		Diagnoses: make([]*diagnosisItem, len(diagnosisCodeIDs)),
	}

	for i, codeID := range diagnosisCodeIDs {
		response.Diagnoses[i] = &diagnosisItem{
			Code:   diagnosisMap[codeID].Code,
			CodeID: codeID,
			Name:   diagnosisMap[codeID].Description,
		}
	}

	httputil.JSONResponse(w, http.StatusOK, response)
}
