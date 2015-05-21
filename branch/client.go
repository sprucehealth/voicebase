package branch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/errors"
)

const (
	baseAPIURL    = "https://api.branch.io"
	urlSuffix     = "/v1/url"
	branchKeyName = "branch_key"
	dataKeyName   = "data"
)

type Client interface {
	URL(linkData map[string]interface{}) (string, error)
}

type BranchClient struct {
	branchKey  string
	httpClient *http.Client
}

func NewBranchClient(branchKey string) Client {
	return &BranchClient{
		branchKey:  branchKey,
		httpClient: &http.Client{},
	}
}

type BranchURLResponse struct {
	URL string `json:"url"`
}

func (bc *BranchClient) URL(linkData map[string]interface{}) (string, error) {
	data, err := json.Marshal(struct {
		BranchKey string                 `json:"branch_key"`
		Data      map[string]interface{} `json:"data"`
	}{
		BranchKey: bc.branchKey,
		Data:      linkData,
	})
	if err != nil {
		return "", errors.Trace(err)
	}

	resp, err := bc.httpClient.Post(baseAPIURL+urlSuffix, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", errors.Trace(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Trace(fmt.Errorf("Received non 200 response %d", resp.StatusCode))
	}

	urlResp := &BranchURLResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&urlResp); err != nil {
		return "", errors.Trace(err)
	}
	return urlResp.URL, nil
}
