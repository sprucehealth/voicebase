package storage

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
)

// awsError matches against aws-sdk-go/internal/apierr.*RequestError since it's an
// internal struct that we can't use directly.
type awsError interface {
	Error() string
	RequestID() string
	StatusCode() int
}

var sseAlgorithm = "AES256"

// S3 is a Store that uses AWS S3
type S3 struct {
	s3     *s3.S3
	bucket string
	prefix string
}

// NewS3 returns a new Store that uses S3
func NewS3(awsSession *session.Session, bucket, prefix string) *S3 {
	// Make sure the path prefix starts and ends with /
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return &S3{
		s3:     s3.New(awsSession),
		bucket: bucket,
		prefix: prefix,
	}
}

func (s *S3) Put(id string, data []byte, contentType string, meta map[string]string) (string, error) {
	return s.PutReader(id, bytes.NewReader(data), int64(len(data)), contentType, meta)
}

func (s *S3) PutReader(id string, r io.ReadSeeker, size int64, contentType string, meta map[string]string) (string, error) {
	var m map[string]*string
	if len(meta) != 0 {
		m = make(map[string]*string, len(meta))
		for k, v := range meta {
			m[k] = &v
		}
	}
	if contentType == "" {
		// TODO: could use the mime package to try to detect type based on extension
		contentType = "application/binary"
	}
	path := s.prefix + id
	_, err := s.s3.PutObject(&s3.PutObjectInput{
		Bucket:               &s.bucket,
		Key:                  &path,
		Body:                 r,
		ContentLength:        &size,
		ContentType:          &contentType,
		ServerSideEncryption: &sseAlgorithm,
		Metadata:             m,
	})
	return id, err
}

func (s *S3) Get(id string) ([]byte, http.Header, error) {
	id, err := parseAndValidateS3ID(id)
	if err != nil {
		return nil, nil, err
	}
	r, headers, err := s.GetReader(id)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, r); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), headers, nil
}

func (s *S3) GetHeader(id string) (http.Header, error) {
	id, err := parseAndValidateS3ID(id)
	if err != nil {
		return nil, err
	}
	head, err := s.s3.HeadObject(&s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    ptr.String(s.prefix + id),
	})
	if e, ok := err.(awsError); ok {
		if e.StatusCode() == http.StatusNotFound {
			return nil, errors.Wrapf(ErrNoObject, "storageID=%q", id)
		}
		return nil, err
	} else if err != nil {
		return nil, err
	}
	return s3Header(head.ContentType, head.ContentLength, head.Metadata), nil
}

func s3Header(contentType *string, contentLength *int64, metadata map[string]*string) http.Header {
	header := http.Header{}
	if contentType != nil {
		header.Set("Content-Type", *contentType)
	}
	if contentLength != nil {
		header.Set("Content-Length", strconv.FormatInt(*contentLength, 10))
	}
	for k, v := range metadata {
		if v != nil {
			header.Set(k, *v)
		}
	}
	return header
}

func (s *S3) GetReader(id string) (io.ReadCloser, http.Header, error) {
	id, err := parseAndValidateS3ID(id)
	if err != nil {
		return nil, nil, err
	}
	obj, err := s.s3.GetObject(&s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    ptr.String(s.prefix + id),
	})
	if e, ok := err.(awsError); ok {
		if e.StatusCode() == http.StatusNotFound {
			return nil, nil, errors.Wrapf(ErrNoObject, "storageID=%q", id)
		}
		return nil, nil, err
	} else if err != nil {
		return nil, nil, err
	}
	return obj.Body, s3Header(obj.ContentType, obj.ContentLength, obj.Metadata), nil
}

func (s *S3) Delete(id string) error {
	id, err := parseAndValidateS3ID(id)
	if err != nil {
		return err
	}
	_, err = s.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    ptr.String(s.prefix + id),
	})
	return err
}

func (s *S3) ExpiringURL(id string, expiration time.Duration) (string, error) {
	id, err := parseAndValidateS3ID(id)
	if err != nil {
		return "", err
	}
	req, _ := s.s3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    ptr.String(s.prefix + id),
	})
	return req.Presign(expiration)
}

func (s *S3) Copy(dstID, srcID string) error {
	dstID, err := parseAndValidateS3ID(dstID)
	if err != nil {
		return err
	}
	srcID, err = parseAndValidateS3ID(srcID)
	if err != nil {
		return err
	}
	_, err = s.s3.CopyObject(&s3.CopyObjectInput{
		Bucket:               &s.bucket,
		Key:                  ptr.String(s.prefix + dstID),
		ServerSideEncryption: &sseAlgorithm,
		CopySource:           ptr.String(url.QueryEscape(s.bucket + s.prefix + srcID)),
	})
	if e, ok := err.(awsError); ok {
		if e.StatusCode() == http.StatusNotFound {
			return errors.Wrapf(ErrNoObject, "storageID=%q", srcID)
		}
		return errors.Trace(err)
	}
	return nil
}

// parseAndValidateS3ID handles deprecated storage IDs by parsing out the final part of the path
func parseAndValidateS3ID(id string) (string, error) {
	if !strings.HasPrefix(id, "s3://") {
		return id, nil
	}
	u, err := url.Parse(id)
	if err != nil {
		return id, err
	}
	ix := strings.LastIndexByte(u.Path, '/')
	if ix >= 0 {
		return u.Path[ix+1:], nil
	}
	return u.Path, nil
}
