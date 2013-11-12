package apiservice

import (
	"bytes"
	"carefront/api"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type PhotoAnswerIntakeHandler struct {
	DataApi            api.DataAPI
	CloudStorageApi    api.CloudStorageAPI
	PatientVisitBucket string
	accountId          int64
}

type PhotoAnswerIntakeErrorResponse struct {
	ErrorString string `json:"error"`
}

type PhotoAnswerIntakeResponse struct {
	AnswerId int64 `json:answer_id`
}

func NewPhotoAnswerIntakeHandler(dataApi api.DataAPI, cloudStorageApi api.CloudStorageAPI, bucketLocation string) *PhotoAnswerIntakeHandler {
	return &PhotoAnswerIntakeHandler{dataApi, cloudStorageApi, bucketLocation, 0}
}

func (p *PhotoAnswerIntakeHandler) AccountIdFromAuthToken(accountId int64) {
	p.accountId = accountId
}

func convertStringToInt64(stringToConvert string) (convertedInt int64, err error) {
	return strconv.ParseInt(stringToConvert, 0, 64)
}

func (p *PhotoAnswerIntakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(5 * 1024 * 1024)
	sectionId := r.FormValue("section_id")
	questionId := r.FormValue("question_id")
	potentialAnswerId := r.FormValue("potential_answer_id")
	patientVisitId := r.FormValue("patient_visit_id")

	if sectionId == "" || questionId == "" || potentialAnswerId == "" || patientVisitId == "" {
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(w, PhotoUploadErrorResponse{"Bad input parameters"})
		return
	}

	patientVisitIdInt, err := convertStringToInt64(patientVisitId)
	if err != nil {
		WriteJSONToHTTPResponseWriter(w, PhotoUploadErrorResponse{"patient_visit_id is not a int when it should be"})
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	questionIdInt, err := convertStringToInt64(questionId)
	if err != nil {
		WriteJSONToHTTPResponseWriter(w, PhotoUploadErrorResponse{"question_id is not a int when it should be"})
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	potentialAnswerIdInt, err := convertStringToInt64(potentialAnswerId)
	if err != nil {
		WriteJSONToHTTPResponseWriter(w, PhotoUploadErrorResponse{"potential_answer_id is not a int when it should be"})

		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sectionIdInt, err := convertStringToInt64(sectionId)
	if err != nil {
		WriteJSONToHTTPResponseWriter(w, PhotoUploadErrorResponse{"section_id is not a int when it should be"})
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	patientId, err := p.DataApi.GetPatientIdFromAccountId(p.accountId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	layoutVersionId, err := p.DataApi.GetLayoutVersionIdForPatientVisit(patientVisitIdInt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		WriteJSONToHTTPResponseWriter(w, PhotoAnswerIntakeErrorResponse{"Error getting latest layout version id"})
		return
	}

	questionType, err := p.DataApi.GetQuestionType(questionIdInt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		WriteJSONToHTTPResponseWriter(w, PhotoAnswerIntakeErrorResponse{"Error getting question type"})
		return
	}

	if questionType != "q_type_photo" {
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(w, PhotoAnswerIntakeErrorResponse{"This api is only for uploading pictures"})
		return
	}

	file, handler, err := r.FormFile("photo")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(w, PhotoAnswerIntakeErrorResponse{err.Error()})
		return
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(w, PhotoAnswerIntakeErrorResponse{err.Error()})
		return
	}

	// create the record for answer input and mark it as pending upload
	patientAnswerInfoIntakeId, err := p.DataApi.CreatePhotoAnswerForQuestionRecord(patientId, questionIdInt, sectionIdInt, patientVisitIdInt, potentialAnswerIdInt, layoutVersionId)
	var buffer bytes.Buffer
	buffer.WriteString(patientVisitId)
	buffer.WriteString("/")
	buffer.WriteString(strconv.FormatInt(patientAnswerInfoIntakeId, 10))

	parts := strings.Split(handler.Filename, ".")
	if len(parts) > 1 {
		buffer.WriteString(".")
		buffer.WriteString(parts[1])
	}

	objectStorageId, _, err := p.CloudStorageApi.PutObjectToLocation(p.PatientVisitBucket, buffer.String(), api.US_EAST_1,
		handler.Header.Get("Content-Type"), data, time.Now().Add(10*time.Minute), p.DataApi)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		WriteJSONToHTTPResponseWriter(w, PhotoAnswerIntakeErrorResponse{err.Error()})
		return
	}

	// once the upload is complete, go ahead and mark the record as active with the object storage id linked
	err = p.DataApi.UpdatePhotoAnswerRecordWithObjectStorageId(patientAnswerInfoIntakeId, objectStorageId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		WriteJSONToHTTPResponseWriter(w, PhotoAnswerIntakeErrorResponse{err.Error()})
		return
	}

	WriteJSONToHTTPResponseWriter(w, PhotoAnswerIntakeResponse{patientAnswerInfoIntakeId})
}
