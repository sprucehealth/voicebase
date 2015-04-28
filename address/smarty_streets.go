// Using http://smartystreets.com/ for Address Validation and Zipcode lookup

package address

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

var ErrPaymentRequired = errors.New("smarty streets: payment required")

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
	params := url.Values{
		"auth-id":    []string{s.AuthID},
		"auth-token": []string{s.AuthToken},
		"zipcode":    []string{zipcode},
	}
	u := smartyStreetsEndpoint + "?" + params.Encode()

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusPaymentRequired {
		return nil, ErrPaymentRequired
	}

	var zipCodes []zipcodeLookupResponseItem
	if err := json.NewDecoder(resp.Body).Decode(&zipCodes); err != nil {
		return nil, err
	}

	if len(zipCodes) == 0 {
		return nil, ErrInvalidZipcode
	}

	switch zipCodes[0].Status {
	case invalidZipcodeStatus:
		return nil, ErrInvalidZipcode
	case "":
		return nil, zipCodes[0].smartyStreetsError
	}

	if len(zipCodes[0].CityStates) == 0 {
		return nil, ErrInvalidZipcode
	}

	return &CityState{
		City:              zipCodes[0].CityStates[0].City,
		State:             zipCodes[0].CityStates[0].State,
		StateAbbreviation: zipCodes[0].CityStates[0].StateAbbreviation,
	}, nil
}
