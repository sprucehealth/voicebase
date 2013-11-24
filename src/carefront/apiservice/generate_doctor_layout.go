package apiservice

import (
	"bytes"
	"carefront/api"
	"carefront/info_intake"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

type GenerateDoctorLayoutHandler struct {
	DataApi                  api.DataAPI
	CloudStorageApi          api.CloudStorageAPI
	DoctorLayoutBucket       string
	DoctorVisualLayoutBucket string
	MaxInMemoryForPhoto      int64
	AWSRegion                string
}

type DoctorLayoutGeneratedResponse struct {
	DoctorLayoutUrls []string `json:"doctor_layout_urls"`
}

func (d *GenerateDoctorLayoutHandler) NonAuthenticated() bool {
	return true
}

func (d *GenerateDoctorLayoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(d.MaxInMemoryForPhoto)
	file, handler, err := r.FormFile("layout")
	if err != nil {
		log.Println(err)
		WriteDeveloperError(w, http.StatusBadRequest, "No layout file or invalid layout file specified")
		return
	}

	doctorLayout := &info_intake.PatientVisitOverview{}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to read in layout file: "+err.Error())
		return
	}

	err = json.Unmarshal(data, &doctorLayout)
	if err != nil {
		log.Println(err)
		WriteDeveloperError(w, http.StatusBadRequest, "Error parsing layout file: "+err.Error())
		return
	}

	healthConditionTag := doctorLayout.HealthConditionTag
	if healthConditionTag == "" {
		WriteDeveloperError(w, http.StatusBadRequest, "health condition not specified or invalid in layout")
		return
	}

	currentActiveBucket, currentActiveKey, currentActiveRegion, err := d.DataApi.GetActiveLayoutInfoForHealthCondition(healthConditionTag, api.DOCTOR_ROLE)
	if currentActiveBucket != "" {
		rawData, err := d.CloudStorageApi.GetObjectAtLocation(currentActiveBucket, currentActiveKey, currentActiveRegion)
		if err != nil {
			log.Println(err)
			WriteDeveloperError(w, http.StatusInternalServerError, "Error getting current active doctor layout from S3: "+err.Error())
			return
		}
		res := bytes.Compare(data, rawData)
		// nothing to do if the layouts are exactly the same
		if res == 0 {
			WriteJSONToHTTPResponseWriter(w, http.StatusOK, DoctorLayoutGeneratedResponse{})
			return
		}
	}

	objectId, objectUrl, err := d.CloudStorageApi.PutObjectToLocation(d.DoctorVisualLayoutBucket,
		strconv.Itoa(int(time.Now().Unix())), d.AWSRegion, handler.Header.Get("Content-Type"), data, time.Now().Add(10*time.Minute), d.DataApi)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to upload file to cloud: "+err.Error())
		return
	}

	healthConditionId, err := d.DataApi.GetHealthConditionInfo(healthConditionTag)
	// once that is successful, create a record for the layout version and mark it as CREATING
	modelId, err := d.DataApi.MarkNewLayoutVersionAsCreating(objectId, layout_syntax_version, healthConditionId, api.DOCTOR_ROLE, "automatically generated")
	if err != nil {
		log.Println(err)
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create record for layout : "+err.Error())
		return
	}

	// fill in the questions from the database
	for _, patientVisitSection := range doctorLayout.Sections {
		for _, subSection := range patientVisitSection.SubSections {
			for _, question := range subSection.Questions {
				question.FillFromDatabase(d.DataApi, api.EN_LANGUAGE_ID, question.ShowPotentialResponses)
			}
		}
	}

	jsonData, err := json.Marshal(doctorLayout)
	objectId, objectUrl, err = d.CloudStorageApi.PutObjectToLocation(d.DoctorLayoutBucket,
		strconv.Itoa(int(time.Now().Unix())), d.AWSRegion, handler.Header.Get("Content-Type"), jsonData, time.Now().Add(10*time.Minute), d.DataApi)
	if err != nil {
		log.Println(err)
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to upload doctor layout after filling in questions to it "+err.Error())
		return
	}

	doctorLayoutId, err := d.DataApi.MarkNewDoctorLayoutAsCreating(objectId, modelId, healthConditionId)
	if err != nil {
		log.Println(err)
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to record for doctor layout: "+err.Error())
		return
	}

	err = d.DataApi.UpdateDoctorActiveLayouts(modelId, doctorLayoutId, healthConditionId)
	if err != nil {
		log.Println(err)
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to mark record as active: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, DoctorLayoutGeneratedResponse{[]string{objectUrl}})
}
