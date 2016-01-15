package twilio

import (
	"encoding/json"
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
		json.Unmarshal(data, &exception)
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
