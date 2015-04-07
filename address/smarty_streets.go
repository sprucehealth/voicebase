// Using http://smartystreets.com/ for Address Validation and Zipcode lookup

package address

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type SmartyStreetsService struct {
	AuthID    string
	AuthToken string
}

const (
	smartyStreetsEndpoint = "https://api.smartystreets.com/zipcode"
	invalidZipcodeStatus  = "invalid_zipcode"
)

type smartyStreetsError struct {
	Reason string `json:"reason"`
	Status string `json:"status"`
}

func (s smartyStreetsError) Error() string {
	return fmt.Sprintf("Error from smarty streets service. Reason = %s, Status = %s", s.Reason, s.Status)
}

type smartyStreetsCityState struct {
	City              string `json:"city"`
	State             string `json:"state"`
	StateAbbreviation string `json:"state_abbreviation"`
}

type zipcodeLookupResponseItem struct {
	smartyStreetsError
	CityStates []smartyStreetsCityState `json:"city_states"`
}

func (s *SmartyStreetsService) ZipcodeLookup(zipcode string) (*CityState, error) {
	cityState := &CityState{}
	endPoint, err := url.Parse(smartyStreetsEndpoint)
	if err != nil {
		return cityState, err
	}

	params := url.Values{}
	params.Set("auth-id", s.AuthID)
	params.Set("auth-token", s.AuthToken)
	params.Set("zipcode", zipcode)
	endPoint.RawQuery = params.Encode()

	httpRequest, err := http.NewRequest("GET", endPoint.String(), nil)
	if err != nil {
		return cityState, err
	}

	resp, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return cityState, err
	}

	zipCodes := make([]zipcodeLookupResponseItem, 1)
	if err := json.NewDecoder(resp.Body).Decode(&zipCodes); err != nil {
		return cityState, err
	}

	if len(zipCodes) == 0 {
		return cityState, ErrInvalidZipcode
	}

	if zipCodes[0].Status == invalidZipcodeStatus {
		return cityState, ErrInvalidZipcode
	}

	if zipCodes[0].Status != "" {
		return cityState, zipCodes[0].smartyStreetsError
	}

	if len(zipCodes[0].CityStates) == 0 {
		return cityState, ErrInvalidZipcode
	}

	cityState.City = zipCodes[0].CityStates[0].City
	cityState.State = zipCodes[0].CityStates[0].State
	cityState.StateAbbreviation = zipCodes[0].CityStates[0].StateAbbreviation

	return cityState, nil
}
