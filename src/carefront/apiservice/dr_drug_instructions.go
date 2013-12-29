package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
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

type DoctorDrugInstructionsRequestResponse struct {
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
	case "POST":
		d.addDrugInstructions(w, r)
	}
}

func (d *DoctorDrugInstructionsHandler) addDrugInstructions(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	addInstructionsRequestBody := &DoctorDrugInstructionsRequestResponse{}

	err := jsonDecoder.Decode(addInstructionsRequestBody)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse json request body for adding instructions: "+err.Error())
		return
	}

	doctorId, err := d.DataApi.GetDoctorIdFromAccountId(d.accountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the doctor id from the account id "+err.Error())
		return
	}

	drugName, drugForm, drugRoute := breakDrugInternalNameIntoComponents(addInstructionsRequestBody.DrugInternalName)

	drugInstructions := make([]*common.DoctorSupplementalInstruction, 0)
	for _, instructionItem := range addInstructionsRequestBody.SupplementalInstructions {
		drugInstruction, err := d.DataApi.AddOrUpdateDrugInstructionForDoctor(drugName, drugForm, drugRoute, instructionItem, doctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add instruction for doctor: "+err.Error())
			return
		}
		drugInstructions = append(drugInstructions, drugInstruction)
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorDrugInstructionsRequestResponse{DrugInternalName: addInstructionsRequestBody.DrugInternalName, SupplementalInstructions: drugInstructions})
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

	drugName, drugForm, drugRoute := breakDrugInternalNameIntoComponents(requestData.DrugInternalName)
	// break up drug name into its components
	drugInstructions, err := d.DataApi.GetDrugInstructionsForDoctor(drugName, drugForm, drugRoute, doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get drug instructions for doctor: "+err.Error())
		return
	}
	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorDrugInstructionsRequestResponse{SupplementalInstructions: drugInstructions, DrugInternalName: requestData.DrugInternalName})
}

func breakDrugInternalNameIntoComponents(drugInternalName string) (drugName, drugForm, drugRoute string) {
	indexOfParanthesis := strings.Index(drugInternalName, "(")
	indexOfClosingParanthesis := strings.Index(drugInternalName, ")")
	indexOfHyphen := strings.Index(drugInternalName, "-")
	drugName = strings.TrimSpace(drugInternalName[:indexOfParanthesis])
	drugRoute = strings.TrimSpace(drugInternalName[indexOfParanthesis+1 : indexOfHyphen])
	drugForm = strings.TrimSpace(drugInternalName[indexOfHyphen+1 : indexOfClosingParanthesis])
	return
}
