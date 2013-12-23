package erx

import (
	"os"
)

type DoseSpotService struct {
	ClinicId  string
	ClinicKey string
	UserId    string
}

const (
	doseSpotAPIEndPoint                = "http://www.dosespot.com/API/11/"
	doseSpotSOAPEndPoint               = "http://i.dosespot.com/api/11/apifull.asmx"
	medicationQuickSearchAction        = "MedicationQuickSearchMessage"
	selfReportedMedicationSearchAction = "SelfReportedMedicationSearch"
	medicationStrengthSearchAction     = "MedicationStrengthSearchMessage"
)

var (
	doseSpotClient = soapClient{SoapAPIEndPoint: doseSpotSOAPEndPoint, APIEndpoint: doseSpotAPIEndPoint}
)

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

func (d *DoseSpotService) GetDrugNamesForDoctor(prefix string) ([]string, error) {
	medicationSearch := &medicationQuickSearchRequest{}
	medicationSearch.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)
	medicationSearch.SearchString = prefix

	searchResult := &medicationQuickSearchResponse{}
	err := doseSpotClient.makeSoapRequest(medicationQuickSearchAction, medicationSearch, searchResult)

	if err != nil {
		return nil, err
	}

	return searchResult.DisplayNames, nil
}

func (d *DoseSpotService) GetDrugNamesForPatient(prefix string) ([]string, error) {
	selfReportedDrugsSearch := &selfReportedMedicationSearchRequest{}
	selfReportedDrugsSearch.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)
	selfReportedDrugsSearch.SearchTerm = prefix

	searchResult := &selfReportedMedicationSearchResponse{}
	err := doseSpotClient.makeSoapRequest(selfReportedMedicationSearchAction, selfReportedDrugsSearch, searchResult)

	if err != nil {
		return nil, err
	}

	drugNames := make([]string, len(searchResult.SearchResults))
	for i, searchResultItem := range searchResult.SearchResults {
		drugNames[i] = searchResultItem.DisplayName
	}

	return drugNames, nil
}

func (d *DoseSpotService) SearchForMedicationStrength(medicationName string) ([]string, error) {
	medicationStrengthSearch := &medicationStrengthSearchRequest{}
	medicationStrengthSearch.SSO = generateSingleSignOn(d.ClinicKey, d.UserId, d.ClinicId)
	medicationStrengthSearch.SearchString = medicationName

	searchResult := &medicationStrengthSearchResponse{}
	err := doseSpotClient.makeSoapRequest(medicationStrengthSearchAction, medicationStrengthSearch, searchResult)

	if err != nil {
		return nil, err
	}

	return searchResult.DisplayStrengths, nil
}
