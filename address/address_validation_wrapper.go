package address

import (
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-cache/cache"
)

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

func (h *hackAddressValidationWrapper) ZipcodeLookup(zipcode string) (*CityState, error) {
	if cityState := h.zipCodeToCityStateMapper[zipcode]; cityState != nil {
		return cityState, nil
	}

	return h.addressValidationService.ZipcodeLookup(zipcode)
}

type addressValidationWithCacheWrapper struct {
	addressValidationService AddressValidationAPI
	cache                    cache.Cache
}

func NewAddressValidationWithCacheWrapper(addressValidationAPI AddressValidationAPI, maxCachedItems int) AddressValidationAPI {
	if maxCachedItems == 0 {
		return addressValidationAPI
	}
	return &addressValidationWithCacheWrapper{
		addressValidationService: addressValidationAPI,
		cache: cache.NewLRUCache(maxCachedItems),
	}
}

func (c *addressValidationWithCacheWrapper) ZipcodeLookup(zipcode string) (*CityState, error) {
	var cityStateInfo *CityState
	cs, err := c.cache.Get(zipcode)
	if err != nil {
		golog.Errorf("Unable to get cityState info from cache: %s", err)
	}

	if err != nil || cs == nil {
		cityStateInfo, err = c.addressValidationService.ZipcodeLookup(zipcode)
		if err != nil {
			return nil, err
		}
		c.cache.Set(zipcode, cityStateInfo)
	} else {
		cityStateInfo = cs.(*CityState)
	}

	return cityStateInfo, nil
}
