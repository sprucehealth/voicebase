package apiservice

import (
	"bytes"
	"carefront/api"
	"github.com/gorilla/schema"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type PhotoAnswerIntakeHandler struct {
	DataApi             api.DataAPI
	CloudStorageApi     api.CloudStorageAPI
	PatientVisitBucket  string
	MaxInMemoryForPhoto int64
	accountId           int64
}

type PhotoAnswerIntakeResponse struct {
	AnswerId int64 `json:answer_id`
}

type PhotoAnswerIntakeRequestData struct {
	QuestionId        int64 `schema:"question_id,required"`
	PotentialAnswerId int64 `schema:"potential_answer_id,required"`
	PatientVisitId    int64 `schema:"patient_visit_id,required"`
}

func NewPhotoAnswerIntakeHandler(dataApi api.DataAPI, cloudStorageApi api.CloudStorageAPI,
	bucketLocation string, maxMemoryForPhotoMB int64) *PhotoAnswerIntakeHandler {
	return &PhotoAnswerIntakeHandler{dataApi, cloudStorageApi, bucketLocation, maxMemoryForPhotoMB, 0}
}

func (p *PhotoAnswerIntakeHandler) AccountIdFromAuthToken(accountId int64) {
	p.accountId = accountId
}

func (p *PhotoAnswerIntakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(p.MaxInMemoryForPhoto)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse out the form values for the request")
		return
	}

	requestData := new(PhotoAnswerIntakeRequestData)
	decoder := schema.NewDecoder()
	err = decoder.Decode(requestData, r.Form)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	patientId, err := p.DataApi.GetPatientIdFromAccountId(p.accountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError,
			"Unable to get patientId from the accountId retrieved from auth token: "+err.Error())
		return
	}

	layoutVersionId, err := p.DataApi.GetLayoutVersionIdForPatientVisit(requestData.PatientVisitId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Error getting latest layout version id: "+err.Error())
		return
	}

	questionType, err := p.DataApi.GetQuestionType(requestData.QuestionId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Error getting question type: "+err.Error())
		return
	}

	if questionType != "q_type_single_photo" && questionType != "q_type_multiple_photo" {
		WriteDeveloperError(w, http.StatusBadRequest, "This api is only for uploading pictures")
		return
	}

	file, handler, err := r.FormFile("photo")
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Missing or invalid photo in parameters: "+err.Error())
		return
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Error reading data from photo: "+err.Error())
		return
	}

	// in the event that we are dealing with a question that can only have one photo set for the potential answer,
	// mark the previously set answer to the quesiton as inactive
	if questionType == "q_type_single_photo" {
		err = p.DataApi.MakeCurrentPhotoAnswerInactive(patientId, requestData.QuestionId, requestData.PatientVisitId, requestData.PotentialAnswerId, layoutVersionId)
		if err != nil {
			WriteDeveloperError(w, http.StatusInternalServerError,
				"Error marking the current active photo answer as inactive: "+err.Error())
			return
		}
	}

	// create the record for answer input and mark it as pending upload
	patientAnswerInfoIntakeId, err := p.DataApi.CreatePhotoAnswerForQuestionRecord(patientId,
		requestData.QuestionId, requestData.PatientVisitId, requestData.PotentialAnswerId, layoutVersionId)
	var buffer bytes.Buffer
	buffer.WriteString(strconv.Itoa(int(requestData.PatientVisitId)))
	buffer.WriteString("/")
	buffer.WriteString(strconv.FormatInt(patientAnswerInfoIntakeId, 10))

	parts := strings.Split(handler.Filename, ".")
	if len(parts) > 1 {
		buffer.WriteString(".")
		buffer.WriteString(parts[1])
	}

	objectStorageId, _, err := p.CloudStorageApi.PutObjectToLocation(p.PatientVisitBucket, buffer.String(), api.US_WEST_1,
		handler.Header.Get("Content-Type"), data, time.Now().Add(10*time.Minute), p.DataApi)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Error uploading image to patient-visit bucket in s3: "+err.Error())
		return
	}

	// once the upload is complete, go ahead and mark the record as active with the object storage id linked
	err = p.DataApi.UpdatePhotoAnswerRecordWithObjectStorageId(patientAnswerInfoIntakeId, objectStorageId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, `Unable to update photo answer record with 
			object storage id after uploading picture: `+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, PhotoAnswerIntakeResponse{patientAnswerInfoIntakeId})
}
