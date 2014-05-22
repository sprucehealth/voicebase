package patient

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/encoding"
	"carefront/libs/dispatch"
	"carefront/libs/golog"
	thriftapi "carefront/thrift/api"
	"net/http"
	"strings"

	"github.com/gorilla/schema"
)

type SignupHandler struct {
	dataApi api.DataAPI
	authApi thriftapi.Auth
}

type PatientSignedupResponse struct {
	Token   string          `json:"token"`
	Patient *common.Patient `json:"patient,omitempty"`
}

func (s *SignupHandler) NonAuthenticated() bool {
	return true
}

type SignupPatientRequestData struct {
	Email      string `schema:"email,required"`
	Password   string `schema:"password,required"`
	FirstName  string `schema:"first_name,required"`
	LastName   string `schema:"last_name,required"`
	Dob        string `schema:"dob,required"`
	Gender     string `schema:"gender,required"`
	Zipcode    string `schema:"zip_code,required"`
	Phone      string `schema:"phone,required"`
	Agreements string `schema:"agreements"`
	DoctorId   int64  `schema:"doctor_id"`
}

func NewSignupHandler(dataApi api.DataAPI, authApi thriftapi.Auth) *SignupHandler {
	return &SignupHandler{
		dataApi: dataApi,
		authApi: authApi,
	}
}

func (s *SignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var requestData SignupPatientRequestData
	if err := schema.NewDecoder().Decode(&requestData, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	// ensure that the date of birth can be correctly parsed
	// Note that the date will be returned as MM/DD/YYYY
	dobParts := strings.Split(requestData.Dob, encoding.DOB_SEPARATOR)

	if len(dobParts) < 3 {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse dob. Format should be "+encoding.DOB_FORMAT)
		return
	}

	// first, create an account for the user
	res, err := s.authApi.SignUp(requestData.Email, requestData.Password, api.PATIENT_ROLE)
	if _, ok := err.(*thriftapi.LoginAlreadyExists); ok {
		apiservice.WriteUserError(w, http.StatusBadRequest, "An account with the specified email address already exists.")
		return
	}

	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Internal Server Error. Unable to register patient")
		return
	}

	newPatient := &common.Patient{
		AccountId: encoding.NewObjectId(res.AccountId),
		FirstName: requestData.FirstName,
		LastName:  requestData.LastName,
		Gender:    requestData.Gender,
		ZipCode:   requestData.Zipcode,
		PhoneNumbers: []*common.PhoneInformation{&common.PhoneInformation{
			Phone:     requestData.Phone,
			PhoneType: api.PHONE_CELL,
		},
		},
	}

	newPatient.Dob, err = encoding.NewDobFromComponents(dobParts[0], dobParts[1], dobParts[2])
	if err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse date of birth. Required format + "+encoding.DOB_FORMAT)
		return
	}

	// then, register the signed up user as a patient
	err = s.dataApi.RegisterPatient(newPatient)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to register patient: "+err.Error())
		return
	}

	// track patient agreements
	if requestData.Agreements != "" {
		patientAgreements := make(map[string]bool)
		for _, agreement := range strings.Split(requestData.Agreements, ",") {
			patientAgreements[strings.TrimSpace(agreement)] = true
		}

		err = s.dataApi.TrackPatientAgreements(newPatient.PatientId.Int64(), patientAgreements)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to track patient agreements: "+err.Error())
			return
		}
	}

	// create care team for patient
	var careProviderGroup *common.PatientCareProviderGroup
	if requestData.DoctorId != 0 {
		careProviderGroup, err = s.dataApi.CreateCareTeamForPatientWithPrimaryDoctor(newPatient.PatientId.Int64(), requestData.DoctorId)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create care team with specified doctor for patient: "+err.Error())
			return
		}
	} else {
		careProviderGroup, err = s.dataApi.CreateCareTeamForPatient(newPatient.PatientId.Int64())
		if err != nil {
			golog.Errorf(err.Error())
			apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to create care team for patient :"+err.Error())
			return
		}
	}

	dispatch.Default.PublishAsync(&CareTeamAssingmentEvent{
		PatientId:   newPatient.PatientId.Int64(),
		Assignments: careProviderGroup.Assignments,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, PatientSignedupResponse{Token: res.Token, Patient: newPatient})
}
