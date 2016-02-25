package twilio

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type RecordingFormat int

const (
	RecordingFormatMP3 RecordingFormat = iota
	RecordingFormatWAV
)

type RecordingService struct {
	client *Client
}

type RecordingIFace interface {
	GetMetadata(sid string) (*Metadata, *Response, error)
	Delete(sid string) (*Response, error)
}

// ParseRecordingSID expects the url to be of the form /2010-04-01/Accounts/{AccountSid}/Recordings/{RecordingSid}
// to be able to parse out the recordingSID.
func ParseRecordingSID(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	parts := strings.Split(parsedURL.Path, "/")
	if len(parts) != 6 {
		return "", fmt.Errorf("Expected URI of the form /2010-04-01/Accounts/{AccountSid}/Recordings/{RecordingSid}, but got %s", parsedURL.Path)
	}

	return strings.Split(parts[5], ".")[0], nil
}

func (r *RecordingService) GetMetadata(sid string) (*Metadata, *Response, error) {
	u := r.client.EndPoint("Recordings", sid)

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	met := new(Metadata)
	res, err := r.client.Do(req, met)
	if err != nil {
		return nil, nil, err
	}
	return met, res, nil
}

func (r *RecordingService) Delete(sid string) (*Response, error) {
	u := r.client.EndPoint("Recordings", sid)

	req, err := r.client.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req, nil)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
