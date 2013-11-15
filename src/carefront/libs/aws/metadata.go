package aws

import (
	"encoding/json"
	"net"
	"net/http"
	"time"
)

type Credentials struct {
	Code            string
	LastUpdatedStr  string    `json:"LastUpdated"`
	LastUpdated     time.Time `json:"-"`
	Type            string
	AccessKeyId     string
	SecretAccessKey string
	Token           string
	ExpirationStr   string    `json:"Expiration"`
	Expiration      time.Time `json:"-"`
}

const metadataTimeout = time.Second

var (
	metadataTransport = &http.Transport{
		Dial: (&net.Dialer{Timeout: metadataTimeout}).Dial,
		ResponseHeaderTimeout: metadataTimeout,
	}
	metadataClient = &http.Client{
		Transport: metadataTransport,
	}
)

func CredentialsForRole(role string) (*Credentials, error) {
	res, err := metadataClient.Get("http://169.254.169.254/latest/meta-data/iam/security-credentials/" + role)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, ErrBadStatusCode(res.StatusCode)
	}
	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	var cred Credentials
	if err := dec.Decode(&cred); err != nil {
		return nil, err
	}
	cred.LastUpdated, err = time.Parse(time.RFC3339, cred.LastUpdatedStr)
	if err != nil {
		return nil, err
	}
	cred.Expiration, err = time.Parse(time.RFC3339, cred.ExpirationStr)
	if err != nil {
		return nil, err
	}
	return &cred, nil
}
