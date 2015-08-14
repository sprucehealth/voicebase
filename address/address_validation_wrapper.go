package address

import (
	"encoding/json"

	"github.com/sprucehealth/backend/libs/golog"
	"gopkgs.com/memcache.v2"
)

const zipLookupCacheExpireSeconds = 60 * 60 * 24 * 29 // Must be less than 1 month or memcached will consider it an epoch

type addressValidationWithCacheWrapper struct {
	addressValidationService Validator
	mc                       *memcache.Client
}

// NewAddressValidationWithCacheWrapper returns an initialized instance of addressValidationWithCacheWrapper wrapping the provided validator and memcache.Client
func NewAddressValidationWithCacheWrapper(validator Validator, mc *memcache.Client) Validator {
	if mc == nil {
		return validator
	}
	return &addressValidationWithCacheWrapper{
		addressValidationService: validator,
		mc: mc,
	}
}

func (c *addressValidationWithCacheWrapper) ZipcodeLookup(zipcode string) (*CityState, error) {
	cacheKey := "zipcs:" + zipcode

	if item, err := c.mc.Get(cacheKey); err != nil {
		if err != memcache.ErrCacheMiss {
			golog.Errorf("Unable to get CityState info for zipcode '%s' from cache: %s", zipcode, err)
		}
	} else {
		var cs CityState
		if err := json.Unmarshal(item.Value, &cs); err != nil {
			golog.Errorf("Failed to unmarshal cached CityState info for zipcode '%s': %s",
				zipcode, err.Error())
		} else {
			return &cs, nil
		}
	}

	cs, err := c.addressValidationService.ZipcodeLookup(zipcode)
	if err != nil {
		return nil, err
	}

	go func() {
		if b, err := json.Marshal(cs); err != nil {
			golog.Errorf("Failed to marshal CityState info: %s", err.Error())
		} else {
			if err := c.mc.Set(&memcache.Item{
				Key:        cacheKey,
				Value:      b,
				Expiration: zipLookupCacheExpireSeconds,
			}); err != nil {
				golog.Errorf("Failed to cache CityState info: %s", err.Error())
			}
		}
	}()

	return cs, nil
}
