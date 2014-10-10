package s3

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

var ErrNoETag = errors.New("s3: no etag in response")

type Multi struct {
	s3       *S3
	key      string
	uploadID string
}

type Part struct {
	PartNumber int
	ETag       string
}

func (m *Multi) PutPartFrom(partNumber int, r io.Reader, size int64) (Part, error) {
	params := (url.Values{
		"partNumber": []string{strconv.Itoa(partNumber)},
		"uploadId":   []string{m.uploadID},
	}).Encode()
	req, err := http.NewRequest("PUT", m.key+"?"+params, r)
	if err != nil {
		return Part{}, err
	}
	req.ContentLength = size
	res, err := m.s3.Do(req, nil)
	if err != nil {
		return Part{}, err
	}
	res.Body.Close()
	etag := res.Header.Get("ETag")
	if etag == "" {
		return Part{}, ErrNoETag
	}
	return Part{
		PartNumber: partNumber,
		ETag:       etag,
	}, nil
}

func (m *Multi) Complete(parts []Part) error {
	params := (url.Values{
		"uploadId": []string{m.uploadID},
	}).Encode()

	body := &bytes.Buffer{}
	if err := xml.NewEncoder(body).Encode(&struct {
		XMLName xml.Name `xml:"CompleteMultipartUpload"`
		Parts   []Part   `xml:"Part"`
	}{
		Parts: parts,
	}); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", m.key+"?"+params, body)
	if err != nil {
		return err
	}
	req.ContentLength = int64(body.Len())
	res, err := m.s3.Do(req, nil)
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

func (m *Multi) Abort() error {
	params := (url.Values{
		"uploadId": []string{m.uploadID},
	}).Encode()
	req, err := http.NewRequest("DELETE", m.key+"?"+params, nil)
	if err != nil {
		return err
	}
	res, err := m.s3.Do(req, nil)
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}
