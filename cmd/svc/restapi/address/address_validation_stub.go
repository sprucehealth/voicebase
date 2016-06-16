package address

type StubAddressValidationService struct {
	CityStateToReturn *CityState
	ErrorToReturn     error
}

func (s StubAddressValidationService) ZipcodeLookup(zipcode string) (*CityState, error) {
	return s.CityStateToReturn, s.ErrorToReturn
}
