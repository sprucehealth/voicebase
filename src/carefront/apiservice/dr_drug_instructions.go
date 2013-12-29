package apiservice

import (
	"carefront/api"
	"carefront/common"
	// "encoding/json"
	// "github.com/gorilla/schema"
	"net/http"
)

type DoctorDrugInstructionsHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type GetDoctorDrugInstructionsRequestData struct {
	DrugInternalName string `json:"drug_internal_name"`
}

type GetDoctorDrugInstructionsResponse struct {
	SupplementalInstructions []*common.DoctorSupplementalInstruction `json:"supplemental_instructions"`
	DrugInternalName         string                                  `json:"drug_internal_name"`
}

func NewDoctorDrugInstructionsHandler(dataApi api.DataAPI) *DoctorDrugInstructionsHandler {
	return &DoctorDrugInstructionsHandler{DataApi: dataApi, accountId: 0}
}

func (d *DoctorDrugInstructionsHandler) AccountIdFromAuthToken(accountId int64) {
	d.accountId = accountId
}

func (d *DoctorDrugInstructionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d.getDrugInstructions(w, r)
	}
}

func (d *DoctorDrugInstructionsHandler) getDrugInstructions(w http.ResponseWriter, r *http.Request) {

}
