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
		t.deleteFavoriteTreatment(w, r)
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

func (t *DoctorFavoriteTreatmentsHandler) deleteFavoriteTreatment(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	favoriteTreatment := &common.DoctorFavoriteTreatment{}

	doctorId, err := t.DataApi.GetDoctorIdFromAccountId(GetContext(r).AccountId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get doctor from account id: "+err.Error())
		return
	}

	err = jsonDecoder.Decode(favoriteTreatment)
	if err != nil {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse treatment body: "+err.Error())
		return
	}

	if favoriteTreatment.Id == 0 {
		WriteDeveloperError(w, http.StatusBadRequest, "Unable to delete a treatment that does not have an id associated with it")
		return
	}

	err = t.DataApi.DeleteFavoriteTreatment(favoriteTreatment, doctorId)
	if err != nil {
		WriteDeveloperError(w, http.StatusInternalServerError, "Unable to delete favorited treatment: "+err.Error())
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

	// break up the name into its components so that it can be saved into the database as its components
	drugName, drugForm, drugRoute := breakDrugInternalNameIntoComponents(favoriteTreatment.FavoritedTreatment.DrugInternalName)
	favoriteTreatment.FavoritedTreatment.DrugName = drugName
	// only break down name into route and form if the route and form are non-empty strings
	if drugForm != "" && drugRoute != "" {
		favoriteTreatment.FavoritedTreatment.DrugForm = drugForm
		favoriteTreatment.FavoritedTreatment.DrugRoute = drugRoute
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
