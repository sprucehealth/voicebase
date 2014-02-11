package aws

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// Metadata available for an EC2 instance.
const (
	MetadataAvailabilityZone = "placement/availability-zone"
	MetadataHostname         = "hostname"
	MetadataInstanceID       = "instance-id"
	MetadataInstanceType     = "instance-type"
	MetadataLocalIPv4        = "local-ipv4"
)

// TODO: The locking on Credentials is pretty inefficient. The request for keys
// should never block, and all updates should happen in the background.

type Credentials struct {
	Code            string
	LastUpdatedStr  string    `json:"LastUpdated"`
	LastUpdated     time.Time `json:"-"`
	Type            string
	AccessKeyID     string
	SecretAccessKey string
	Token           string
	ExpirationStr   string    `json:"Expiration"`
	Expiration      time.Time `json:"-"`
	Role            string

	mu sync.RWMutex
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

func GetMetadataReader(path string) (io.ReadCloser, error) {
	res, err := metadataClient.Get("http://169.254.169.254/latest/meta-data/" + path)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, ErrBadStatusCode(res.StatusCode)
	}
	return res.Body, nil
}

func GetMetadata(path string) (string, error) {
	rd, err := GetMetadataReader(path)
	if err != nil {
		return "", err
	}
	defer rd.Close()
	by, err := ioutil.ReadAll(rd)
	if err != nil {
		return "", err
	}
	return string(by), nil
}

var (
	defaultRole     string
	defaultRoleOnce sync.Once
)

func CredentialsForRole(role string) (*Credentials, error) {
	if role == "" {
		defaultRoleOnce.Do(func() {
			rl, err := GetMetadata("iam/security-credentials/")
			if err != nil {
				return
			}
			defaultRole = rl
		})
		role = defaultRole
		if role == "" {
			return nil, errors.New("aws: unable to get default role")
		}
	}
	cred := &Credentials{Role: role}
	return cred, cred.Update()
}

func (c *Credentials) Update() error {
	c.mu.RLock()
	if c.Expiration.After(time.Now()) {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	rd, err := GetMetadataReader("iam/security-credentials/" + c.Role)
	if err != nil {
		return err
	}
	defer rd.Close()
	dec := json.NewDecoder(rd)
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
		AccessKey: c.AccessKeyID,
		SecretKey: c.SecretAccessKey,
		Token:     c.Token,
	}
}
