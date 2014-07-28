package address

type hackAddressValidationWrapper struct {
	addressValidationService AddressValidationAPI
	zipCodeToCityStateMapper map[string]*CityState
}

// NewHackAddressValidationWrapper returns an addressValidationAPI that shortcuits the zipcode lookup for certain registered zipcodes
// to return the specified cityState information
func NewHackAddressValidationWrapper(addressValidationAPI AddressValidationAPI, zipCodeToCityStateMapper map[string]*CityState) AddressValidationAPI {
	return &hackAddressValidationWrapper{
		addressValidationService: addressValidationAPI,
		zipCodeToCityStateMapper: zipCodeToCityStateMapper,
	}
}

func (h *hackAddressValidationWrapper) ZipcodeLookup(zipcode string) (CityState, error) {
	if cityState := h.zipCodeToCityStateMapper[zipcode]; cityState != nil {
		return *cityState, nil
	}

	return h.addressValidationService.ZipcodeLookup(zipcode)
}
