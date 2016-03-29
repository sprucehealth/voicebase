package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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

// IDFromName returns a deterministic ID for a name.
func (s *S3) IDFromName(name string) string {
	return fmt.Sprintf("s3://%s/%s%s%s", *s.s3.Config.Region, s.bucket, s.prefix, name)
}

func (s *S3) Put(name string, data []byte, contentType string, meta map[string]string) (string, error) {
	return s.PutReader(name, bytes.NewReader(data), int64(len(data)), contentType, meta)
}

func (s *S3) PutReader(name string, r io.ReadSeeker, size int64, contentType string, meta map[string]string) (string, error) {
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
	path := s.prefix + name
	_, err := s.s3.PutObject(&s3.PutObjectInput{
		Bucket:               &s.bucket,
		Key:                  &path,
		Body:                 r,
		ContentLength:        &size,
		ContentType:          &contentType,
		ServerSideEncryption: &sseAlgorithm,
		Metadata:             m,
	})
	if err != nil {
		return "", err
	}
	return s.IDFromName(name), nil
}

func (s *S3) Get(id string) ([]byte, http.Header, error) {
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

func (s *S3) GetReader(id string) (io.ReadCloser, http.Header, error) {
	region, bkt, path, err := s.parseURI(id)
	if err != nil {
		return nil, nil, err
	}
	// TODO(samuel): Support different regions
	_ = region
	obj, err := s.s3.GetObject(&s3.GetObjectInput{
		Bucket: &bkt,
		Key:    &path,
	})
	if e, ok := err.(awsError); ok {
		if e.StatusCode() == http.StatusNotFound {
			return nil, nil, ErrNoObject
		}
		return nil, nil, err
	} else if err != nil {
		return nil, nil, err
	}
	header := http.Header{}
	if obj.ContentType != nil {
		header.Set("Content-Type", *obj.ContentType)
	}
	if obj.ContentLength != nil {
		header.Set("Content-Length", strconv.FormatInt(*obj.ContentLength, 10))
	}
	for k, v := range obj.Metadata {
		if v != nil {
			header.Set(k, *v)
		}
	}
	return obj.Body, header, nil
}

func (s *S3) Delete(id string) error {
	region, bkt, path, err := s.parseURI(id)
	if err != nil {
		return err
	}
	// TODO(samuel): Support different regions
	_ = region
	_, err = s.s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &bkt,
		Key:    &path,
	})
	return err
}

func (s *S3) ExpiringURL(id string, expiration time.Duration) (string, error) {
	_, bkt, path, err := s.parseURI(id)
	if err != nil {
		return "", err
	}

	req, _ := s.s3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: &bkt,
		Key:    &path,
	})

	return req.Presign(expiration)
}

func (s *S3) parseURI(uri string) (region string, bucket string, key string, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", "", "", err
	}
	p := strings.SplitN(u.Path, "/", 3)
	if len(p) < 3 {
		return "", "", "", fmt.Errorf("storage: bad S3 path %s", u.Path)
	}
	return u.Host, p[1], "/" + p[2], nil
}
