package branch

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/gopkgs.com/memcache.v2"
	"github.com/sprucehealth/backend/test"
)

type recordingTransport struct {
	reqs      []*http.Request
	resps     []*http.Response
	callCount int
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.callCount++
	t.reqs = append(t.reqs, req)
	retIndex := t.callCount - 1
	if len(t.resps) >= retIndex {
		retIndex = len(t.resps) - 1
	}
	return t.resps[retIndex], nil
}

type mcClient struct {
	setParam *memcache.Item
	setErr   error
	getParam string
	getErr   error
	get      *memcache.Item
}

func (mc *mcClient) Get(key string) (item *memcache.Item, err error) {
	mc.getParam = key
	return mc.get, mc.getErr
}

func (mc *mcClient) Set(item *memcache.Item) error {
	mc.setParam = item
	return mc.setErr
}

func TestGenerateBranchURL(t *testing.T) {
	expectedURL := "branch.url.com"
	data, err := json.Marshal(&branchURLResponse{
		URL: expectedURL,
	})
	cli := &client{
		branchKey: "key",
		httpClient: &http.Client{
			Transport: &recordingTransport{
				resps: []*http.Response{
					&http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(data)),
					},
				},
			},
		},
	}

	url, err := cli.URL(map[string]interface{}{})
	test.OK(t, err)
	test.Equals(t, expectedURL, url)
}

func TestGenerateBranchURLNotOK(t *testing.T) {
	cli := &client{
		branchKey: "key",
		httpClient: &http.Client{
			Transport: &recordingTransport{
				resps: []*http.Response{
					&http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       ioutil.NopCloser(bytes.NewReader(nil)),
					},
				},
			},
		},
	}

	url, err := cli.URL(map[string]interface{}{})
	test.Assert(t, err != nil, "Expected a non nil err from the client")
	test.Equals(t, "", url)
}

func TestGenerateBranchURLOK1Retry(t *testing.T) {
	expectedURL := "branch.url.com"
	data, err := json.Marshal(&branchURLResponse{
		URL: expectedURL,
	})
	cli := &client{
		branchKey: "key",
		httpClient: &http.Client{
			Transport: &recordingTransport{
				resps: []*http.Response{
					&http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       ioutil.NopCloser(bytes.NewReader(nil)),
					},
					&http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(data)),
					},
				},
			},
		},
	}

	url, err := cli.URL(map[string]interface{}{})
	test.OK(t, err)
	test.Equals(t, expectedURL, url)
}

func TestGenerateBranchURL6RetryNotOK(t *testing.T) {
	cli := &client{
		branchKey: "key",
		httpClient: &http.Client{
			Transport: &recordingTransport{
				resps: []*http.Response{
					&http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       ioutil.NopCloser(bytes.NewReader(nil)),
					},
					&http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       ioutil.NopCloser(bytes.NewReader(nil)),
					},
					&http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       ioutil.NopCloser(bytes.NewReader(nil)),
					},
					&http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       ioutil.NopCloser(bytes.NewReader(nil)),
					},
					&http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       ioutil.NopCloser(bytes.NewReader(nil)),
					},
				},
			},
		},
	}

	url, err := cli.URL(map[string]interface{}{})
	test.Assert(t, err != nil, "Expected a non nil err from the client")
	test.Equals(t, "", url)
}

func TestGenerateMemcacheBranchURLCacheHit(t *testing.T) {
	expectedURL := "branch.url.com"
	cli := &memcachedBranchClient{
		branchKey: "key",
		mc: &mcClient{
			get: &memcache.Item{Value: []byte(expectedURL)},
		},
	}

	url, err := cli.URL(map[string]interface{}{})
	test.OK(t, err)
	test.Equals(t, expectedURL, url)
}

func TestGenerateMemcacheBranchURLCacheMiss(t *testing.T) {
	expectedURL := "branch.url.com"
	data, err := json.Marshal(&branchURLResponse{
		URL: expectedURL,
	})
	mc := &mcClient{
		getErr: errors.New("Something bad!"),
	}
	cli := &memcachedBranchClient{
		branchKey: "key",
		client: &client{
			branchKey: "key",
			httpClient: &http.Client{
				Transport: &recordingTransport{
					resps: []*http.Response{
						&http.Response{
							StatusCode: http.StatusOK,
							Body:       ioutil.NopCloser(bytes.NewReader(data)),
						},
					},
				},
			},
		},
		mc: mc,
	}

	url, err := cli.URL(map[string]interface{}{"foo": "bar"})
	test.OK(t, err)
	hash, err := cli.requestHash(map[string]interface{}{"foo": "bar"})
	test.OK(t, err)
	test.Equals(t, mc.getParam, `branchURL:`+string(hash))
	test.Equals(t, mc.setParam.Key, `branchURL:`+string(hash))
	test.Equals(t, mc.setParam.Value, []byte(expectedURL))
	test.Equals(t, mc.setParam.Expiration, urlLookupCacheExpireSeconds)
	test.Equals(t, expectedURL, url)
}
