package awsutil

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

// ErrBadStatusCode is the error when an API endpoint returns a non-successful status code.
type ErrBadStatusCode int

func (e ErrBadStatusCode) Error() string {
	return fmt.Sprintf("bad status code %d", int(e))
}

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
