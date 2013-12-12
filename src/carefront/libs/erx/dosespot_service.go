package erx

import (
	"os"
)

type DoseSpotService struct {
	ClinicId  string
	ClinicKey string
	UserId    string
}

func NewDoseSpotService(clinicId, clinicKey, userId string) *DoseSpotService {
	d := &DoseSpotService{}
	if clinicId == "" {
		d.ClinicKey = os.Getenv("DOSESPOT_CLINIC_KEY")
		d.UserId = os.Getenv("DOSESPOT_USER_ID")
		d.ClinicId = os.Getenv("DOSESPOT_CLINIC_ID")
	} else {
		d.ClinicKey = clinicKey
		d.ClinicId = clinicId
		d.UserId = userId
	}
	return d
}

func (d *DoseSpotService) GetDrugNames(prefix string) ([]string, error) {
	medicationSearch := newMedicationQuickSearchMessage()
	medicationSearch.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)
	medicationSearch.SearchString = prefix

	searchResult := &medicationQuickSearchResult{}
	err := makeSoapRequest(medicationSearch, searchResult)

	if err != nil {
		return nil, err
	}

	return searchResult.DisplayNames, nil
}
