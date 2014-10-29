package patient

import (
	"errors"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/sku"
)

var (
	InitialVisitNotTreated = errors.New("Cannot create a followup if the patient has not yet been treated by a doctor")
	NoInitialVisitFound    = errors.New("Cannot create followup if the patient has not gone through a visit yet")
)

// CreatePendingFollowup creates a followup visit for the patient in the PENDING status. This is a visit in the complete sense, with the
// only difference being that it will transition into the OPEN state on the first read by the patient. The reason for this is to do our best
// effort in using the patient's latest app version to pick the layout version to use for the followup. When creating the pending followup,
// we use the last seen app version for the patient to identify the layout to pick. Then, on the actual read of the followup visit by the patient
// we compare this layout version with the layout version based on the patient's actual app version and update it if the app versions are different.
func CreatePendingFollowup(patient *common.Patient, dataAPI api.DataAPI, authAPI api.AuthAPI,
	dispatcher *dispatch.Dispatcher, store storage.Store, expirationDuration time.Duration) (*common.PatientVisit, error) {

	// Ensure that a patient has gone through a regular visit before creating a followup
	patientVisit, err := dataAPI.GetPatientVisitForSKU(patient.PatientId.Int64(), sku.AcneVisit)
	if err == api.NoRowsError {
		return nil, NoInitialVisitFound
	} else if err != nil {
		return nil, err
	} else if patientVisit.Status != common.PVStatusTreated {
		return nil, InitialVisitNotTreated
	}

	// Ensure that there isn't already an open followup before creating yet another one
	patientVisit, err = dataAPI.GetLastCreatedPatientVisit(patient.PatientId.Int64())
	if err != nil && err != api.NoRowsError {
		return nil, err
	} else if patientVisit.Status == common.PVStatusOpen || patientVisit.Status == common.PVStatusPending {
		// nothing to do since there already exists an open followup visit
		return nil, err
	}

	// Using the last app version information for the patient, create a followup visit
	platform, appVersion, err := authAPI.LatestAppPlatformVersion(patient.AccountId.Int64())
	if err != nil && err != api.NoRowsError {
		return nil, err
	} else if err == api.NoRowsError {
		// if last app version information is not present, then create a followup visit
		// with the latest layout and assumption of iOS. this is okay because the
		// layout version will be switched over when the patient attempts to read the
		// layout for the first time
		ios := common.IOS
		platform = &ios

		skuID, err := dataAPI.SKUID(sku.AcneFollowup)
		if err != nil {
			return nil, err
		}
		appVersion, err = dataAPI.LatestAppVersionSupported(api.HEALTH_CONDITION_ACNE_ID, &skuID, ios, api.PATIENT_ROLE, api.ConditionIntakePurpose)
		if err != nil {
			return nil, err
		}
	}

	layoutVersionID, err := dataAPI.IntakeLayoutVersionIDForAppVersion(appVersion, *platform,
		api.HEALTH_CONDITION_ACNE_ID, api.EN_LANGUAGE_ID, sku.AcneFollowup)
	if err != nil {
		return nil, err
	}

	followupVisit := &common.PatientVisit{
		PatientId:         patient.PatientId,
		PatientCaseId:     patientVisit.PatientCaseId,
		HealthConditionId: encoding.NewObjectId(api.HEALTH_CONDITION_ACNE_ID),
		Status:            common.PVStatusPending,
		LayoutVersionId:   encoding.NewObjectId(layoutVersionID),
		SKU:               sku.AcneFollowup,
	}

	_, err = dataAPI.CreatePatientVisit(followupVisit)
	if err != nil {
		return nil, err
	}

	return followupVisit, nil
}

func checkLayoutVersionForFollowup(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, visit *common.PatientVisit, r *http.Request) error {
	// if we are dealing with a followup visit in the pending state,
	// then ensure that the visit has been created with the latest version layout supported by
	// the client
	isFollowup, err := dataAPI.IsFollowupVisit(visit.PatientVisitId.Int64())
	if err != nil {
		return err
	}

	if isFollowup && visit.Status == common.PVStatusPending {
		headers := apiservice.ExtractSpruceHeaders(r)
		var layoutVersionToUpdate *int64
		var status string
		layoutVersionID, err := dataAPI.IntakeLayoutVersionIDForAppVersion(headers.AppVersion, headers.Platform,
			visit.HealthConditionId.Int64(), api.EN_LANGUAGE_ID, visit.SKU)
		if err != nil {
			return err
		} else if layoutVersionID != visit.LayoutVersionId.Int64() {
			layoutVersionToUpdate = &layoutVersionID
			visit.LayoutVersionId = encoding.NewObjectId(layoutVersionID)
		}

		// update the layout and the status for this visit
		status = common.PVStatusOpen
		visit.Status = common.PVStatusOpen
		if err := dataAPI.UpdatePatientVisit(visit.PatientVisitId.Int64(), &api.PatientVisitUpdate{
			Status:          &status,
			LayoutVersionID: layoutVersionToUpdate,
		}); err != nil {
			return err
		}

		dispatcher.Publish(&VisitStartedEvent{
			VisitId:       visit.PatientVisitId.Int64(),
			PatientCaseId: visit.PatientCaseId.Int64(),
		})
	}
	return nil
}
