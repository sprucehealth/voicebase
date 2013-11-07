package apiservice

import (
	"bytes"
	"carefront/api"
	"carefront/layout_transformer"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

const (
	CAREFRONT_LAYOUT_BUCKET        = "carefront-layout"
	CAREFRONT_CLIENT_LAYOUT_BUCKET = "carefront-client-layout"
	US_EAST_REGION                 = "us-east-1"
)

type LayoutHandler struct {
	DataApi         api.DataAPI
	CloudStorageApi api.CloudStorageAPI
}

type LayoutProcessedResponse struct {
	ClientLayoutUrls []string `json:"clientLayoutUrls"`
}

type LayoutProcessingErrorResponse struct {
	PhotoUploadErrorString string `json:"error"`
}

func (l *LayoutHandler) NonAuthenticated() bool {
	return true
}

func (l *LayoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	file, handler, err := r.FormFile("layout")
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(w, PhotoUploadErrorResponse{err.Error()})
		return
	}

	// ensure that the file is a valid treatment layout, by trying to parse it
	// into the structure
	treatment := &layout_transformer.Treatment{}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(w, LayoutProcessingErrorResponse{err.Error()})
		return
	}

	err = json.Unmarshal(data, &treatment)
	if err != nil {
		fmt.Println("here")
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(w, LayoutProcessingErrorResponse{err.Error()})
		return
	}

	// determine the treatment tag so as to identify what treatment this layout belongs to
	treatmentTag := treatment.TreatmentTag
	if treatmentTag == "" {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		WriteJSONToHTTPResponseWriter(w, LayoutProcessingErrorResponse{err.Error()})
		return
	}

	// check if the current active layout is the same as the layout trying to be uploaded
	currentActiveBucket, currentActiveKey, currentActiveRegion, _ := l.DataApi.GetActiveLayoutInfoForTreatment(treatmentTag)
	if currentActiveBucket != "" {
		rawData, err := l.CloudStorageApi.GetObjectAtLocation(currentActiveBucket, currentActiveKey, currentActiveRegion)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			WriteJSONToHTTPResponseWriter(w, LayoutProcessingErrorResponse{err.Error()})
			return
		}
		res := bytes.Compare(data, rawData)
		// nothing to do if the layouts are exactly the same
		if res == 0 {
			WriteJSONToHTTPResponseWriter(w, LayoutProcessedResponse{nil})
			return
		}
	}

	// upload the layout version to S3 and get back an object storage id
	objectId, _, err := l.CloudStorageApi.PutObjectToLocation(CAREFRONT_LAYOUT_BUCKET,
		strconv.Itoa(int(time.Now().Unix())), US_EAST_REGION, handler.Header.Get("Content-Type"), data, time.Now().Add(10*time.Minute), l.DataApi)

	// once that is successful, create a record for the layout version and mark is as CREATING
	layoutId, err := l.DataApi.MarkNewLayoutVersionAsCreating(objectId, 1, 1, "automatically generated")

	// get all the supported languages
	_, supportedLanguageIds, err := l.DataApi.GetSupportedLanguages()

	// generate a client layout for each language
	clientLayouts := make(map[int64]*layout_transformer.Treatment)
	clientLayoutProcessor := &layout_transformer.TreatmentLayoutProcessor{l.DataApi}
	clientLayoutVersionIds := make([]int64, len(supportedLanguageIds))
	clientLayoutUrls := make([]string, len(supportedLanguageIds))

	for i, supportedLanguageId := range supportedLanguageIds {
		clientLayout := *treatment
		clientLayoutProcessor.TransformIntakeIntoClientLayout(&clientLayout, supportedLanguageId)
		clientLayouts[supportedLanguageId] = &clientLayout

		jsonData, err := json.MarshalIndent(&clientLayout, "", " ")
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			WriteJSONToHTTPResponseWriter(w, LayoutProcessingErrorResponse{err.Error()})
			return
		}
		// put each client layout that is generated into S3
		objectId, clientLayoutUrl, err := l.CloudStorageApi.PutObjectToLocation(CAREFRONT_CLIENT_LAYOUT_BUCKET,
			strconv.Itoa(int(time.Now().Unix())), US_EAST_REGION, handler.Header.Get("Content-Type"), jsonData, time.Now().Add(10*time.Minute), l.DataApi)
		clientLayoutUrls[i] = clientLayoutUrl
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			WriteJSONToHTTPResponseWriter(w, LayoutProcessingErrorResponse{err.Error()})
			return
		}

		// mark the client layout as creating until we have uploaded all client layouts before marking it as ACTIVE
		clientLayoutId, err := l.DataApi.MarkNewPatientLayoutVersionAsCreating(objectId, supportedLanguageId, layoutId, clientLayout.TreatmentId)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			WriteJSONToHTTPResponseWriter(w, LayoutProcessingErrorResponse{err.Error()})
			return
		}
		clientLayoutVersionIds[i] = clientLayoutId
	}

	// update the active layouts to the new current set of layouts
	l.DataApi.UpdateActiveLayouts(layoutId, clientLayoutVersionIds, 1)

	WriteJSONToHTTPResponseWriter(w, LayoutProcessedResponse{clientLayoutUrls})
}
