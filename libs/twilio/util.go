package twilio

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
)

func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	exception := new(Exception)
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		if err := json.Unmarshal(data, &exception); err != nil {
			// Might be XML exception for REST requests
			exc := struct {
				RestException *Exception
			}{
				RestException: exception,
			}
			if err := xml.Unmarshal(data, &exc); err != nil {
				return errors.New("twilio: unparseable error response: " + string(data))
			}
		}
	}

	return exception
}

type Metadata struct {
	SID         string    `json:"sid"`
	AccountSID  string    `json:"account_sid"`
	ParentSID   string    `json:"parent_sid"`
	ContentType string    `json:"content-type"`
	DateCreated Timestamp `json:"date_created"`
	DateUpdated Timestamp `json:"date_updated"`
	URI         string    `json:"uri"`
}
