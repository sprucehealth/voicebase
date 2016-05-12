package patient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/cmd/svc/restapi/erx"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/treatment_plan"
	"github.com/sprucehealth/backend/views"
	"golang.org/x/net/context"
)

type rxReminderHandlerReminderService struct {
	remindersForPatientParam         common.PatientID
	remindersForPatientErr           error
	remindersForPatient              map[int64]*common.RXReminder
	deleteRXReminderParam            int64
	deleteRXReminderErr              error
	createRXReminderParam            *common.RXReminder
	createRXReminderErr              error
	updateRXReminderTreatmentIDParam int64
	updateRXReminderRxReminderParam  *common.RXReminder
	updateRXReminderErr              error
}

func (s *rxReminderHandlerReminderService) RemindersForPatient(patientID common.PatientID) (map[int64]*common.RXReminder, error) {
	s.remindersForPatientParam = patientID
	return s.remindersForPatient, s.remindersForPatientErr
}

func (s *rxReminderHandlerReminderService) DeleteRXReminder(treatmentID int64) error {
	s.deleteRXReminderParam = treatmentID
	return s.deleteRXReminderErr
}

func (s *rxReminderHandlerReminderService) CreateRXReminder(reminder *common.RXReminder) error {
	s.createRXReminderParam = reminder
	return s.createRXReminderErr
}

func (s *rxReminderHandlerReminderService) UpdateRXReminder(treatmentID int64, reminder *common.RXReminder) error {
	s.updateRXReminderTreatmentIDParam = treatmentID
	s.updateRXReminderRxReminderParam = reminder
	return s.updateRXReminderErr
}

type rxReminderHandlerTreatmentPlanService struct {
	patientCanAccessTreatmentPatientIDParam   common.PatientID
	patientCanAccessTreatmentTreatmentIDParam int64
	patientCanAccessTreatmentErr              error
	patientCanAccessTreatment                 bool
	treatmentsForPatientParam                 common.PatientID
	treatmentsForPatientErr                   error
	treatmentsForPatient                      []*common.Treatment
}

func (s *rxReminderHandlerTreatmentPlanService) PatientCanAccessTreatment(patientID common.PatientID, treatmentID int64) (bool, error) {
	s.patientCanAccessTreatmentPatientIDParam = patientID
	s.patientCanAccessTreatmentTreatmentIDParam = treatmentID
	return s.patientCanAccessTreatment, s.patientCanAccessTreatmentErr
}

func (s *rxReminderHandlerTreatmentPlanService) TreatmentsForPatient(patientID common.PatientID) ([]*common.Treatment, error) {
	s.treatmentsForPatientParam = patientID
	return s.treatmentsForPatient, s.treatmentsForPatientErr
}

type rxReminderHandlerDrugService struct {
	multiQueryDrugDetailIDsParam []*api.DrugDetailsQuery
	multiQueryDrugDetailIDsErr   error
	multiQueryDrugDetailIDs      []int64
}

func (s *rxReminderHandlerDrugService) MultiQueryDrugDetailIDs(queries []*api.DrugDetailsQuery) ([]int64, error) {
	s.multiQueryDrugDetailIDsParam = queries
	return s.multiQueryDrugDetailIDs, s.multiQueryDrugDetailIDsErr
}

func TestRxReminderHandlerDELETETreatmentIDRequired(t *testing.T) {
	r, err := http.NewRequest("DELETE", "mock.api.request?", nil)
	test.OK(t, err)
	handler := NewRXReminderHandlerHandler(&rxReminderHandlerReminderService{}, &rxReminderHandlerTreatmentPlanService{}, &rxReminderHandlerDrugService{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestRxReminderHandlerDELETETreatmentIDAccessRequired(t *testing.T) {
	r, err := http.NewRequest("DELETE", "mock.api.request?treatment_id=1", nil)
	test.OK(t, err)
	handler := NewRXReminderHandlerHandler(&rxReminderHandlerReminderService{}, &rxReminderHandlerTreatmentPlanService{
		patientCanAccessTreatment: false,
	}, &rxReminderHandlerDrugService{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(apiservice.CtxWithPatient(context.Background(), &common.Patient{}), responseWriter, r)
	test.Equals(t, http.StatusForbidden, responseWriter.Code)
}

func TestRxReminderHandlerDELETETreatmentIDAccessValidationError(t *testing.T) {
	r, err := http.NewRequest("DELETE", "mock.api.request?treatment_id=1", nil)
	test.OK(t, err)
	handler := NewRXReminderHandlerHandler(&rxReminderHandlerReminderService{}, &rxReminderHandlerTreatmentPlanService{
		patientCanAccessTreatmentErr: errors.New("Foo"),
	}, &rxReminderHandlerDrugService{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(apiservice.CtxWithPatient(context.Background(), &common.Patient{}), responseWriter, r)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestRxReminderHandlerDELETEDALDeleteError(t *testing.T) {
	var treatmentID int64 = 1
	var patientID uint64 = 2
	r, err := http.NewRequest("DELETE", fmt.Sprintf("mock.api.request?treatment_id=%d", treatmentID), nil)
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{
		deleteRXReminderErr: nil,
	}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{
		patientCanAccessTreatment: true,
	}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteJSONSuccess(expectedWriter)
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: common.NewPatientID(patientID)}),
		responseWriter, r)
	test.Equals(t, http.StatusOK, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, treatmentID, reminderSvc.deleteRXReminderParam)
}

func TestRxReminderHandlerGETRemindersForPatientError(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	handler := NewRXReminderHandlerHandler(&rxReminderHandlerReminderService{
		remindersForPatientErr: errors.New("Foo"),
	}, &rxReminderHandlerTreatmentPlanService{}, &rxReminderHandlerDrugService{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(apiservice.CtxWithPatient(context.Background(), &common.Patient{}), responseWriter, r)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestRxReminderHandlerGETTreatmentsForPatientError(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?include_views=1", nil)
	test.OK(t, err)
	handler := NewRXReminderHandlerHandler(&rxReminderHandlerReminderService{
		remindersForPatient: make(map[int64]*common.RXReminder),
	}, &rxReminderHandlerTreatmentPlanService{
		treatmentsForPatientErr: errors.New("Foo"),
	}, &rxReminderHandlerDrugService{})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(apiservice.CtxWithPatient(context.Background(), &common.Patient{}), responseWriter, r)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestRxReminderHandlerGETRxRemindersNoView(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	reminder := &common.RXReminder{TreatmentID: treatmentID}
	reminders := map[int64]*common.RXReminder{treatmentID: reminder}
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{
		remindersForPatient: reminders,
	}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(
		expectedWriter,
		http.StatusOK,
		&rxRemindersGETResponse{
			Reminders: map[string]*responses.RXReminder{
				strconv.FormatInt(treatmentID, 10): responses.TransformRXReminder(reminder),
			},
		},
	)
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusOK, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, patientID, reminderSvc.remindersForPatientParam)
}

func TestRxReminderHandlerGETRxRemindersWithView(t *testing.T) {
	var treatmentID int64 = 1
	treatment := &common.Treatment{
		ID:              encoding.NewObjectID(uint64(treatmentID)),
		DrugDBIDs:       map[string]string{erx.NDC: "NDC"},
		GenericDrugName: "GenericName",
		DrugRoute:       "topical",
		DrugForm:        "DrugForm",
	}
	patientID := common.NewPatientID(2)
	reminder := &common.RXReminder{TreatmentID: treatmentID}
	reminders := map[int64]*common.RXReminder{treatmentID: reminder}
	r, err := http.NewRequest("GET", "mock.api.request?include_views=true", nil)
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{
		remindersForPatient: reminders,
	}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{
		treatmentsForPatient: []*common.Treatment{treatment},
	}
	drugSvc := &rxReminderHandlerDrugService{
		multiQueryDrugDetailIDs: []int64{1},
	}
	view := treatment_plan.NewPrescriptionView(
		treatment,
		"Topical Prescription",
		app_url.IconRXLarge,
		[]views.View{
			treatment_plan.NewPrescriptionButtonView("Prescription Guide", app_url.IconRXGuide, app_url.ViewTreatmentGuideAction(treatmentID)),
		})
	test.OK(t, view.Validate("treatment"))
	views := map[string]views.View{
		strconv.FormatInt(treatmentID, 10): view,
	}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(
		expectedWriter,
		http.StatusOK,
		&rxRemindersGETResponse{
			TreatmentViews: views,
			Reminders: map[string]*responses.RXReminder{
				strconv.FormatInt(treatmentID, 10): responses.TransformRXReminder(reminder),
			},
		},
	)
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusOK, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}

func TestRxReminderHandlerPOSTRxReminderTreatmentIDRequired(t *testing.T) {
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		ReminderText: "foo",
		Times:        []string{"12:30"},
		Interval:     "EVERY_OTHER_DAY",
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestRxReminderHandlerPOSTRxReminderReminderTextRequired(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID: treatmentID,
		Times:       []string{"12:30"},
		Interval:    "EVERY_OTHER_DAY",
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestRxReminderHandlerPOSTRxReminderTimesRequired(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Times:        []string{},
		Interval:     "EVERY_OTHER_DAY",
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestRxReminderHandlerPOSTRxIntervalRequired(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Times:        []string{"12:30"},
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestRxReminderHandlerPOSTRxDaysWithCUSTOMRequired(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Interval:     "CUSTOM",
		Times:        []string{"12:30"},
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestRxReminderHandlerPOSTAccessRequired(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Interval:     "EVERY_DAY",
		Times:        []string{"12:30"},
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{
		patientCanAccessTreatment: false,
	}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusForbidden, responseWriter.Code)
	test.Equals(t, treatmentSvc.patientCanAccessTreatmentPatientIDParam, patientID)
	test.Equals(t, treatmentSvc.patientCanAccessTreatmentTreatmentIDParam, treatmentID)
}

func TestRxReminderHandlerPOSTCreateRXReminderErr(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Interval:     "CUSTOM",
		Days:         []string{"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"},
		Times:        []string{"12:30"},
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{
		createRXReminderErr: errors.New("Foo"),
	}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{
		patientCanAccessTreatment: true,
	}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestRxReminderHandlerPOSTRxReminder(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	reminder := &common.RXReminder{TreatmentID: treatmentID}
	reminders := map[int64]*common.RXReminder{treatmentID: reminder}
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Interval:     "CUSTOM",
		Days:         []string{"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"},
		Times:        []string{"12:30"},
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{
		createRXReminderErr: nil,
		remindersForPatient: reminders,
	}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{
		patientCanAccessTreatment: true,
	}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(
		expectedWriter,
		http.StatusOK,
		&rxRemindersGETResponse{
			Reminders: map[string]*responses.RXReminder{
				strconv.FormatInt(treatmentID, 10): responses.TransformRXReminder(reminder),
			},
		},
	)
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusOK, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, patientID, reminderSvc.remindersForPatientParam)
}

func TestRxReminderHandlerPUTRxReminderTreatmentIDRequired(t *testing.T) {
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		ReminderText: "foo",
		Times:        []string{"12:30"},
		Interval:     "EVERY_OTHER_DAY",
	})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestRxReminderHandlerPUTRxReminderReminderTextRequired(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID: treatmentID,
		Times:       []string{"12:30"},
		Interval:    "EVERY_OTHER_DAY",
	})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestRxReminderHandlerPUTRxReminderTimesRequired(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Times:        []string{},
		Interval:     "EVERY_OTHER_DAY",
	})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestRxReminderHandlerPUTRxIntervalRequired(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Times:        []string{"12:30"},
	})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestRxReminderHandlerPUTRxDaysWithCUSTOMRequired(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Interval:     "CUSTOM",
		Times:        []string{"12:30"},
	})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestRxReminderHandlerPUTAccessRequired(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Interval:     "EVERY_DAY",
		Times:        []string{"12:30"},
	})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{
		patientCanAccessTreatment: false,
	}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusForbidden, responseWriter.Code)
	test.Equals(t, treatmentSvc.patientCanAccessTreatmentPatientIDParam, patientID)
	test.Equals(t, treatmentSvc.patientCanAccessTreatmentTreatmentIDParam, treatmentID)
}

func TestRxReminderHandlerPUTCreateRXReminderErr(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Interval:     "CUSTOM",
		Days:         []string{"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"},
		Times:        []string{"12:30"},
	})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{
		updateRXReminderErr: errors.New("Foo"),
	}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{
		patientCanAccessTreatment: true,
	}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestRxReminderHandlerPUTRxReminder(t *testing.T) {
	var treatmentID int64 = 1
	patientID := common.NewPatientID(2)
	reminder := &common.RXReminder{TreatmentID: treatmentID}
	reminders := map[int64]*common.RXReminder{treatmentID: reminder}
	data, err := json.Marshal(&rxRemindersPOSTRequest{
		TreatmentID:  treatmentID,
		ReminderText: "foo",
		Interval:     "CUSTOM",
		Days:         []string{"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"},
		Times:        []string{"12:30"},
	})
	test.OK(t, err)
	r, err := http.NewRequest("PUT", "mock.api.request", bytes.NewReader(data))
	test.OK(t, err)
	reminderSvc := &rxReminderHandlerReminderService{
		updateRXReminderErr: nil,
		remindersForPatient: reminders,
	}
	treatmentSvc := &rxReminderHandlerTreatmentPlanService{
		patientCanAccessTreatment: true,
	}
	drugSvc := &rxReminderHandlerDrugService{}
	handler := NewRXReminderHandlerHandler(reminderSvc, treatmentSvc, drugSvc)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(
		expectedWriter,
		http.StatusOK,
		&rxRemindersGETResponse{
			Reminders: map[string]*responses.RXReminder{
				strconv.FormatInt(treatmentID, 10): responses.TransformRXReminder(reminder),
			},
		},
	)
	handler.ServeHTTP(
		apiservice.CtxWithPatient(context.Background(), &common.Patient{ID: patientID}),
		responseWriter, r,
	)
	test.Equals(t, http.StatusOK, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
	test.Equals(t, treatmentID, reminderSvc.updateRXReminderTreatmentIDParam)
}
