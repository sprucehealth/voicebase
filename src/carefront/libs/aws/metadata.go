package aws

import (
	"encoding/json"
	"log"
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
	Role            string
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
	cred := &Credentials{Role: role}
	return cred, cred.Update()
}

func (c *Credentials) Update() error {
	if c.Expiration.After(time.Now()) {
		return nil
	}

	res, err := metadataClient.Get("http://169.254.169.254/latest/meta-data/iam/security-credentials/" + c.Role)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return ErrBadStatusCode(res.StatusCode)
	}
	defer res.Body.Close()
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(c); err != nil {
		return err
	}
	c.LastUpdated, err = time.Parse(time.RFC3339, c.LastUpdatedStr)
	if err != nil {
		return err
	}
	c.Expiration, err = time.Parse(time.RFC3339, c.ExpirationStr)
	if err != nil {
		return err
	}
	return nil
}

func (c *Credentials) Keys() Keys {
	if c.Role != "" {
		if err := c.Update(); err != nil {
			log.Printf("aws: failed to refresh credentials for role %s: %s", c.Role, err.Error())
		}
	}
	return Keys{
		AccessKey: c.AccessKeyId,
		SecretKey: c.SecretAccessKey,
		Token:     c.Token,
	}
}
