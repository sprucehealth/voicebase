package apiservice

import (
	"carefront/api"
	"carefront/common"
	"github.com/gorilla/schema"
	"net/http"
	"strings"
)

type DoctorDrugInstructionsHandler struct {
	DataApi   api.DataAPI
	accountId int64
}

type GetDoctorDrugInstructionsRequestData struct {
	DrugInternalName string `schema:"drug_internal_name"`
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
	r.ParseForm()
	requestData := new(GetDoctorDrugInstructionsRequestData)
	decoder := schema.NewDecoder()
	err := decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	doctorId, err := d.DataApi.GetDoctorIdFromAccountId(d.accountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the doctor id from the account id "+err.Error())
		return
	}

	// break up drug name into its components
	indexOfParanthesis := strings.Index(requestData.DrugInternalName, "(")
	indexOfClosingParanthesis := strings.Index(requestData.DrugInternalName, ")")
	indexOfHyphen := strings.Index(requestData.DrugInternalName, "-")
	drugName := strings.TrimSpace(requestData.DrugInternalName[:indexOfParanthesis])
	drugRoute := strings.TrimSpace(requestData.DrugInternalName[indexOfParanthesis+1 : indexOfHyphen])
	drugForm := strings.TrimSpace(requestData.DrugInternalName[indexOfHyphen+1 : indexOfClosingParanthesis])
	drugInstructions, err := d.DataApi.GetDrugInstructionsForDoctor(drugName, drugForm, drugRoute, doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get drug instructions for doctor: "+err.Error())
		return
	}
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &GetDoctorDrugInstructionsResponse{SupplementalInstructions: drugInstructions, DrugInternalName: requestData.DrugInternalName})
}
