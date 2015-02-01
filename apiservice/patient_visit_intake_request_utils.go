package apiservice

import (
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
)

type doctorInfo struct {
	ShortDisplayName  string
	SmallThumbnailURL string
	Description       string
}
type VisitLayoutContext struct {
	Doctor   *doctorInfo
	Patient  *common.Patient
	CaseName string
}

type templated struct {
	Templated bool `json:"is_templated"`
}

func GetPatientLayoutForPatientVisit(
	visit *common.PatientVisit,
	languageID int64,
	dataAPI api.DataAPI,
	apiDomain string) (*info_intake.InfoIntakeLayout, error) {
	layoutVersion, err := dataAPI.GetPatientLayout(visit.LayoutVersionID.Int64(), languageID)
	if err != nil {
		return nil, err
	}

	// first lets check if the json is templated
	var isTemplated templated
	if err := json.Unmarshal(layoutVersion.Layout, &isTemplated); err != nil {
		return nil, err
	} else if isTemplated.Templated {
		var doctor *common.Doctor

		// if it is then populate the context
		doctorMember, err := dataAPI.GetActiveCareTeamMemberForCase(api.DOCTOR_ROLE, visit.PatientCaseID.Int64())
		if err == nil {
			doctor, err = dataAPI.Doctor(doctorMember.ProviderID, true)
			if err != nil {
				return nil, err
			}
		} else if !api.IsErrNotFound(err) {
			return nil, err
		}

		patient, err := dataAPI.Patient(visit.PatientID.Int64(), true)
		if err != nil {
			return nil, err
		}

		// if no doctor is found then we assume that the visit
		// will be treated by the first available doctor
		var dInfo doctorInfo
		if doctor == nil {
			dInfo.Description = "First Available Doctor"
			dInfo.ShortDisplayName = "Your doctor"
			dInfo.SmallThumbnailURL = ""
		} else {
			dInfo.ShortDisplayName = doctor.ShortDisplayName
			dInfo.Description = doctor.ShortDisplayName
			dInfo.SmallThumbnailURL = app_url.ThumbnailURL(apiDomain, api.DOCTOR_ROLE, doctor.DoctorID.Int64())
		}

		patientCase, err := dataAPI.GetPatientCaseFromID(visit.PatientCaseID.Int64())
		if err != nil {
			return nil, err
		}

		context := &VisitLayoutContext{
			Patient:  patient,
			Doctor:   &dInfo,
			CaseName: patientCase.Name,
		}

		layout, err := applyLayoutToContext(context, layoutVersion.Layout)
		if err != nil {
			return nil, err
		} else if layout != nil {
			layoutVersion.Layout = layout
		}
	}

	patientVisitLayout := &info_intake.InfoIntakeLayout{}
	if err := json.Unmarshal(layoutVersion.Layout, patientVisitLayout); err != nil {
		return nil, err
	}
	return patientVisitLayout, err
}

func GetCurrentActiveClientLayoutForPathway(dataAPI api.DataAPI, pathwayID, languageID int64, skuType string,
	appVersion *common.Version, platform common.Platform, context *VisitLayoutContext) (*info_intake.InfoIntakeLayout, int64, error) {

	data, layoutVersionID, err := dataAPI.IntakeLayoutForAppVersion(appVersion, platform, pathwayID, languageID, skuType)
	if err != nil {
		return nil, 0, err
	}

	layoutData, err := applyLayoutToContext(context, data)
	if err != nil {
		return nil, 0, err
	} else if layoutData != nil {
		data = layoutData
	}

	patientVisitLayout := &info_intake.InfoIntakeLayout{}
	if err := json.Unmarshal(data, patientVisitLayout); err != nil {
		return nil, 0, err
	}
	return patientVisitLayout, layoutVersionID, nil
}

func applyLayoutToContext(context *VisitLayoutContext, layout []byte) ([]byte, error) {
	if context == nil {
		return nil, nil
	}

	var b bytes.Buffer
	tmpl, err := template.New("Layout").Parse(string(layout))
	if err != nil {
		return nil, err
	}
	if err := tmpl.Execute(&b, context); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
