package api

const (
	AttrAmericanBoardCertified        = "AmericanBoardCertified"
	AttrBackgroundCheckAgreement      = "BackgroundCheckAgreement"
	AttrClaimsHistoryAgreement        = "ClaimsHistoryAgreement"
	AttrClaimsHistoryFile             = "ClaimsHistoryFile"
	AttrContinuedEducation            = "ContinuedEducation"
	AttrContinuedEducationCreditHours = "ContinuedEducationCreditHours"
	AttrCurrentLiabilityInsurer       = "CurrentLiabilityInsurer"
	AttrCVFile                        = "CVFile"
	AttrDoctorClaims                  = "DoctorClaims"
	AttrDoctorImpairments             = "DoctorImpairments"
	AttrDoctorIncidents               = "DoctorIncidents"
	AttrDoctorViolations              = "DoctorViolations"
	AttrDriversLicenseFile            = "DriversLicenseFile"
	AttrEBusinessAgreement            = "EBusinessAgreement"
	AttrExcitedAboutSpruce            = "ExcitedAboutSpruce"
	AttrHoursUsingSprucePerWeek       = "HoursUsingSprucePerWeek"
	AttrInsuranceDeclines             = "InsuranceDeclines"
	AttrJacketSize                    = "JacketSize"
	AttrMostRecentCertificationDate   = "MostRecentCertificationDate"
	AttrPreviousLiabilityInsurers     = "PreviousLiabilityInsurers"
	AttrRiskManagementCourse          = "RiskManagementCourse"
	AttrSexualMisconduct              = "SexualMisconduct"
	AttrSpecialtyBoard                = "SpecialtyBoard"
	AttrSocialSecurityNumber          = "SocialSecurityNumber"
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
