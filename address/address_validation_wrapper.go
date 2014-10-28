package address

import (
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-cache/cache"
)

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
