package aws

// TODO: retries

import (
	"bytes"
	"crypto/md5"
	"io"
	"io/ioutil"
	"net/http"
)

type S3 struct {
	Region
	Client *Client
}

func (s3 *S3) buildPath(bucket, path string) string {
	if len(path) == 0 || path[0] != '/' {
		path = "/" + path
	}
	return "/" + bucket + path
}

func (s3 *S3) buildUrl(bucket, path string) string {
	return s3.S3Endpoint + s3.buildPath(bucket, path)
}

func (s3 *S3) Do(req *http.Request) (*http.Response, error) {
	s3.sign(req)
	res, err := s3.Client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		defer res.Body.Close()
		return res, ParseErrorResponse(res.Body)
	}
	return res, nil
}

// http://docs.aws.amazon.com/AmazonS3/latest/API/RESTObjectGET.html
func (s3 *S3) Get(bucket, path string) ([]byte, error) {
	rd, err := s3.GetReader(bucket, path)
	if err != nil {
		return nil, err
	}
	defer rd.Close()
	return ioutil.ReadAll(rd)
}

// http://docs.aws.amazon.com/AmazonS3/latest/API/RESTObjectGET.html
func (s3 *S3) GetReader(bucket, path string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", s3.buildUrl(bucket, path), nil)
	if err != nil {
		return nil, err
	}
	res, err := s3.Do(req)
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}

// http://docs.aws.amazon.com/AmazonS3/latest/API/RESTObjectPUT.html
func (s3 *S3) Put(bucket, path string, data []byte, contType string, perm ACL, additionalHeaders map[string][]string) error {
	h := md5.New()
	h.Write(data)
	md5Sum := base64Std.EncodeToString(h.Sum(nil))

	dataReader := bytes.NewReader(data)
	req, err := http.NewRequest("PUT", s3.buildUrl(bucket, path), dataReader)
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(data))
	req.Header.Set("Content-MD5", md5Sum)
	if contType != "" {
		req.Header.Set("Content-Type", contType)
	}
	if perm != "" {
		req.Header.Set(HeaderACL, string(perm))
	}
	for key, values := range additionalHeaders {
		req.Header[key] = values
	}
	res, err := s3.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusOK {
		res.Body.Close()
	}
	return nil
}

// http://docs.aws.amazon.com/AmazonS3/latest/API/RESTObjectDELETE.html
func (s3 *S3) Delete(bucket, path string) error {
	req, err := http.NewRequest("DELETE", s3.buildUrl(bucket, path), nil)
	if err != nil {
		return err
	}
	res, err := s3.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusOK {
		res.Body.Close()
	}
	return nil
}
