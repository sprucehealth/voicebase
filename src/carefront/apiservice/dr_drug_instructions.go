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

type DeleteDrugInstructionsResponse struct {
	Result string `json:"result"`
}

type DoctorDrugInstructionsRequestResponse struct {
	SupplementalInstructions []*common.DoctorSupplementalInstruction `json:"supplemental_instructions"`
	DrugInternalName         string                                  `json:"drug_internal_name"`
	TreatmentId              int64                                   `json:"treatment_id,string,omitempty"`
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
	case "DELETE":
		d.deleteDrugInstructions(w, r)
	}

}

func (d *DoctorDrugInstructionsHandler) deleteDrugInstructions(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	deleteInstructionsRequestBody := &DoctorDrugInstructionsRequestResponse{}

	err := jsonDecoder.Decode(deleteInstructionsRequestBody)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse json request body for adding instructions: "+err.Error())
		return
	}

	doctorId, err := d.DataApi.GetDoctorIdFromAccountId(d.accountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get the doctor id from the account id "+err.Error())
		return
	}

	for _, instructionItem := range deleteInstructionsRequestBody.SupplementalInstructions {
		err := d.DataApi.DeleteDrugInstructionForDoctor(instructionItem, doctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add instruction for doctor: "+err.Error())
			return
		}
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DeleteDrugInstructionsResponse{Result: "success"})
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

	// this means that the intent is to add the instructions to the treatment id specified
	if addInstructionsRequestBody.TreatmentId != 0 {
		err = d.DataApi.AddDrugInstructionsToTreatment(drugName, drugForm, drugRoute, addInstructionsRequestBody.SupplementalInstructions, addInstructionsRequestBody.TreatmentId, doctorId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError, "Unable to add instructions to treatment: "+err.Error())
			return
		}
		WriteJSONToHTTPResponseWriter(w, http.StatusOK, nil)
		return
	}

	drugInstructions := make([]*common.DoctorSupplementalInstruction, 0)
	for _, instructionItem := range addInstructionsRequestBody.SupplementalInstructions {
		if instructionItem.Text == "" {
			WriteDeveloperError(w, http.StatusBadRequest, "The text for the instruction is empty so nothing to add or update: "+err.Error())
			return
		}
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
