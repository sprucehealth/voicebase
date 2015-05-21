package branch

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/test"
)

type recordingTransport struct {
	req  *http.Request
	resp *http.Response
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.req = req
	return t.resp, nil
}

func TestGenerateBranchURL(t *testing.T) {
	expectedURL := "branch.url.com"
	data, err := json.Marshal(&BranchURLResponse{
		URL: expectedURL,
	})
	cli := &BranchClient{
		branchKey: "key",
		httpClient: &http.Client{
			Transport: &recordingTransport{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader(data)),
				},
			},
		},
	}

	url, err := cli.URL(map[string]interface{}{})
	test.OK(t, err)
	test.Equals(t, expectedURL, url)
}

func TestGenerateBranchURLNotOK(t *testing.T) {
	cli := &BranchClient{
		branchKey: "key",
		httpClient: &http.Client{
			Transport: &recordingTransport{
				resp: &http.Response{
					StatusCode: http.StatusBadRequest,
				},
			},
		},
	}

	url, err := cli.URL(map[string]interface{}{})
	test.Assert(t, err != nil, "Expected a non nil err from the client")
	test.Equals(t, "", url)
}
