package api

const (
	AttrAmericanBoardCertified        = "AmericanBoardCertified"
	AttrContinuedEducation            = "ContinuedEducation"
	AttrContinuedEducationCreditHours = "ContinuedEducationCreditHours"
	AttrCVFile                        = "CVFile"
	AttrDriversLicenseFile            = "DriversLicenseFile"
	AttrExcitedAboutSpruce            = "ExcitedAboutSpruce"
	AttrHoursUsingSprucePerWeek       = "HoursUsingSprucePerWeek"
	AttrJacketSize                    = "JacketSize"
	AttrMostRecentCertificationDate   = "MostRecentCertificationDate"
	AttrRiskManagementCourse          = "RiskManagementCourse"
	AttrSpecialtyBoard                = "SpecialtyBoard"
	AttrTimesActiveOnSpruce           = "TimesActiveOnSpruce"
)

func BoolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func StringToBool(s string) bool {
	if s == "true" {
		return true
	}
	return false
}
