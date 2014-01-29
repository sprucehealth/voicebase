package apiservice

import (
	"carefront/api"
	"carefront/common"
	"encoding/json"
	"net/http"
)

type DoctorFavoriteTreatmentsHandler struct {
	DataApi api.DataAPI
}

type DoctorFavoriteTreatmentsResponse struct {
	FavoritedTreatments []*common.DoctorFavoriteTreatment `json:"favorite_treatments"`
}

func (t *DoctorFavoriteTreatmentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		t.getFavoriteTreatments(w, r)
	case "POST":
		t.addFavoriteTreatments(w, r)
	case "DELETE":
	}
}

func (t *DoctorFavoriteTreatmentsHandler) getFavoriteTreatments(w http.ResponseWriter, r *http.Request) {
	doctorId, err := t.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	favoriteTreatments, err := t.DataApi.GetFavoriteTreatments(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorite treatments for doctor: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentsResponse{FavoritedTreatments: favoriteTreatments})
}

func (t *DoctorFavoriteTreatmentsHandler) addFavoriteTreatments(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	favoriteTreatment := &common.DoctorFavoriteTreatment{}

	err := jsonDecoder.Decode(favoriteTreatment)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatment body: "+err.Error())
		return
	}

	err = validateTreatment(favoriteTreatment.FavoritedTreatment)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, err.Error())
		return
	}

	doctorId, err := t.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	err = t.DataApi.AddFavoriteTreatment(favoriteTreatment, doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to favorite treatment: "+err.Error())
		return
	}

	favoriteTreatments, err := t.DataApi.GetFavoriteTreatments(doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get favorited treatments for doctor: "+err.Error())
		return
	}

	WriteJSONToHTTPResponseWriter(w, http.StatusOK, &DoctorFavoriteTreatmentsResponse{FavoritedTreatments: favoriteTreatments})
}
