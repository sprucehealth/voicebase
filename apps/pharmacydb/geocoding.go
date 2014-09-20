package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/libs/golog"
)

const (
	tokenExpirationMinutes = 60 * 24
)

// arcgisClient is a client for  the ArcGIS World Geocoding Service
// https://developers.arcgis.com/rest/geocode/api-reference/overview-world-geocoding-service.htm
type arcgisClient struct {
	clientID     string
	clientSecret string
	accessToken  string
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

type addressItem struct {
	Attributes address `json:"attributes"`
}

type address struct {
	ObjectID int64  `json:"OBJECTID"`
	Address  string `json:"Address"`
	City     string `json:"City"`
	Region   string `json:"Region"`
	Postal   string `json:"Postal"`
}

type addressResultItem struct {
	Location struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"location"`
	Score      float64 `json:"score"`
	Attributes struct {
		ResultID int64   `json:"ResultID"`
		DisplayX float64 `json:"DisplayX"`
		DisplayY float64 `json:"DisplayY"`
	} `json:"attributes"`
}

type addressResult struct {
	Locations []*addressResultItem `json:"locations"`
}

func (a *arcgisClient) getAccessToken() error {
	params := url.Values{}
	params.Set("client_secret", a.clientSecret)
	params.Set("client_id", a.clientID)
	params.Set("grant_type", "client_credentials")
	params.Set("f", "json")
	params.Set("expiration", strconv.FormatInt(tokenExpirationMinutes, 10))
	resp, err := http.Get("https://www.arcgis.com/sharing/oauth2/token?" + params.Encode())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200 but got %d", resp.StatusCode)
	}

	responseData := &accessTokenResponse{}
	if err := json.NewDecoder(resp.Body).Decode(responseData); err != nil {
		return err
	}

	a.accessToken = responseData.AccessToken
	return nil
}

func (a *arcgisClient) geocodeAddresses(addresses []*address) (*addressResult, error) {
	// wrap each address into the address item to prepare the request
	addressItems := make([]*addressItem, len(addresses))
	for i, ad := range addresses {
		addressItems[i] = &addressItem{
			Attributes: *ad,
		}
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"records": addressItems,
	})
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("token", a.accessToken)
	params.Set("f", "json")
	params.Set("addresses", string(jsonData))
	params.Set("sourceCountry", "USA")

	res, err := http.Post("https://geocode.arcgis.com/arcgis/rest/services/World/GeocodeServer/geocodeAddresses",
		"application/x-www-form-urlencoded",
		strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	} else if res.StatusCode != http.StatusOK {
		respData, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		} else {
			golog.Errorf(string(respData))
		}

		// TODO better error reporting
		return nil, fmt.Errorf("Expected 200 but got %d", res.StatusCode)
	}

	responseData := &addressResult{}
	if err := json.NewDecoder(res.Body).Decode(responseData); err != nil {
		return nil, err
	}
	res.Body.Close()

	return responseData, nil
}
