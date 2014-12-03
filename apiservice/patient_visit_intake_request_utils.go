package apiservice

import (
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/sku"
)

type VisitLayoutContext struct {
	Doctor  *common.Doctor
	Patient *common.Patient
}

type templated struct {
	Templated bool `json:"is_templated"`
}

func GetPatientLayoutForPatientVisit(visit *common.PatientVisit, languageId int64, dataApi api.DataAPI) (*info_intake.InfoIntakeLayout, error) {
	layoutVersion, err := dataApi.GetPatientLayout(visit.LayoutVersionId.Int64(), languageId)
	if err != nil {
		return nil, err
	}

	// first lets check if the json is templated
	var isTemplated templated
	if err := json.Unmarshal(layoutVersion.Layout, &isTemplated); err != nil {
		return nil, err
	} else if isTemplated.Templated {
		// if it is then populate the context
		doctorMember, err := dataApi.GetActiveCareTeamMemberForCase(api.DOCTOR_ROLE, visit.PatientCaseId.Int64())
		if err != api.NoRowsError && err != nil {
			return nil, err
		}
		var doctor *common.Doctor
		if doctorMember != nil {
			doctor, err = dataApi.Doctor(doctorMember.ProviderID, true)
			if err != nil {
				return nil, err
			}
		}

		patient, err := dataApi.Patient(visit.PatientId.Int64(), true)
		if err != nil {
			return nil, err
		}

		context := &VisitLayoutContext{
			Patient: patient,
			Doctor:  doctor,
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

func GetCurrentActiveClientLayoutForHealthCondition(dataAPI api.DataAPI, healthConditionId, languageId int64, skuType sku.SKU,
	appVersion *common.Version, platform common.Platform, context *VisitLayoutContext) (*info_intake.InfoIntakeLayout, int64, error) {
	data, layoutVersionId, err := dataAPI.IntakeLayoutForAppVersion(appVersion, platform, languageId, healthConditionId, skuType)
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
	return patientVisitLayout, layoutVersionId, nil
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
