package patient

import (
	"net/http"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dispatch"
)

type followupError string

func (f followupError) IsUserError() bool   { return true }
func (f followupError) UserError() string   { return string(f) }
func (f followupError) Error() string       { return string(f) }
func (f followupError) HTTPStatusCode() int { return http.StatusBadRequest }

var (
	InitialVisitNotTreated    = followupError("Cannot create a followup if the patient has not yet been treated by a doctor")
	NoInitialVisitFound       = followupError("Cannot create followup if the patient has not gone through an initial visit yet")
	FollowupNotSupportedOnApp = followupError("Followup is not supported on the patient's current app version")
	OpenFollowupExists        = followupError("There is already a pending followup for the patient")
)

// CreatePendingFollowup creates a followup visit for the patient in the PENDING status. This is a visit in the complete sense, with the
// only difference being that it will transition into the OPEN state on the first read by the patient. The reason for this is to do our best
// effort in using the patient's latest app version to pick the layout version to use for the followup. When creating the pending followup,
// we use the last seen app version for the patient to identify the layout to pick. Then, on the actual read of the followup visit by the patient
// we compare this layout version with the layout version based on the patient's actual app version and update it if the app versions are different.
func CreatePendingFollowup(
	patient *common.Patient,
	patientCase *common.PatientCase,
	dataAPI api.DataAPI,
	authAPI api.AuthAPI,
	dispatcher *dispatch.Dispatcher,
) (*common.PatientVisit, error) {

	if patientCase == nil {
		return nil, NoInitialVisitFound
	}

	// A followup visit can be created for a case if there exists an active case for the pahtway with no
	// open patient visits
	patientVisits, err := dataAPI.GetVisitsForCase(patientCase.ID.Int64(), append(common.OpenPatientVisitStates(), common.SubmittedPatientVisitStates()...))
	if err != nil {
		return nil, err
	} else if len(patientVisits) > 0 {
		return nil, InitialVisitNotTreated
	}

	pathway, err := dataAPI.PathwayForTag(patientCase.PathwayTag, api.PONone)
	if err != nil {
		return nil, err
	}

	sku, err := dataAPI.SKUForPathway(pathway.Tag, common.SCFollowup)
	if err != nil {
		return nil, err
	}

	// Using the last app version information for the patient, create a followup visit
	appInfo, err := authAPI.LatestAppInfo(patient.AccountID.Int64())
	var platform common.Platform
	var appVersion *common.Version
	if api.IsErrNotFound(err) {
		// if last app version information is not present, then create a followup visit
		// with the latest layout and assumption of iOS. this is okay because the
		// layout version will be switched over when the patient attempts to read the
		// layout for the first time
		platform = common.IOS

		appVersion, err = dataAPI.LatestAppVersionSupported(pathway.ID, &sku.ID, common.IOS, api.RolePatient, api.ConditionIntakePurpose)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		platform = appInfo.Platform
		appVersion = appInfo.Version
	}

	layoutVersionID, err := dataAPI.IntakeLayoutVersionIDForAppVersion(appVersion, platform,
		pathway.ID, api.LanguageIDEnglish, sku.Type)
	if api.IsErrNotFound(err) {
		return nil, FollowupNotSupportedOnApp
	} else if err != nil {
		return nil, err
	}

	followupVisit := &common.PatientVisit{
		PatientID:       patient.PatientID,
		PatientCaseID:   patientCase.ID,
		PathwayTag:      pathway.Tag,
		Status:          common.PVStatusPending,
		LayoutVersionID: encoding.NewObjectID(layoutVersionID),
		SKUType:         sku.Type,
		IsFollowup:      true,
	}

	// In theory a follow up visit should never create a new case. The requested doctor param is only used when creating a new case in
	// the underlying code. Leave this nil for now until care creation is broken out into a standalone construct
	_, err = dataAPI.CreatePatientVisit(followupVisit, nil)
	if err != nil {
		return nil, err
	}

	return followupVisit, nil
}

func checkLayoutVersionForFollowup(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher, visit *common.PatientVisit, r *http.Request) error {
	// if we are dealing with a followup visit in the pending state,
	// then ensure that the visit has been created with the latest version layout supported by
	// the client
	if visit.IsFollowup && visit.Status == common.PVStatusPending {
		pathway, err := dataAPI.PathwayForTag(visit.PathwayTag, api.PONone)
		if err != nil {
			return err
		}

		headers := apiservice.ExtractSpruceHeaders(r)
		var layoutVersionToUpdate *int64
		var status string
		layoutVersionID, err := dataAPI.IntakeLayoutVersionIDForAppVersion(headers.AppVersion, headers.Platform,
			pathway.ID, api.LanguageIDEnglish, visit.SKUType)
		if err != nil {
			return err
		} else if layoutVersionID != visit.LayoutVersionID.Int64() {
			layoutVersionToUpdate = &layoutVersionID
			visit.LayoutVersionID = encoding.NewObjectID(layoutVersionID)
		}

		// update the layout and the status for this visit
		status = common.PVStatusOpen
		visit.Status = common.PVStatusOpen
		if err := dataAPI.UpdatePatientVisit(visit.PatientVisitID.Int64(), &api.PatientVisitUpdate{
			Status:          &status,
			LayoutVersionID: layoutVersionToUpdate,
		}); err != nil {
			return err
		}

		dispatcher.Publish(&VisitStartedEvent{
			VisitID:       visit.PatientVisitID.Int64(),
			PatientCaseID: visit.PatientCaseID.Int64(),
		})
	}
	return nil
}
