package branch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sprucehealth/backend/errors"
)

const (
	baseAPIURL = "https://api.branch.io"
	urlSuffix  = "/v1/url"
)

type Client interface {
	URL(linkData map[string]interface{}) (string, error)
}

type client struct {
	branchKey  string
	httpClient *http.Client
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("branch: [%d] %s", e.Code, e.Message)
}

func NewBranchClient(branchKey string) Client {
	return &client{
		branchKey:  branchKey,
		httpClient: &http.Client{},
	}
}

type branchURLResponse struct {
	URL string `json:"url"`
}

func (bc *client) URL(linkData map[string]interface{}) (string, error) {
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
		var e struct {
			Error *Error `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			return "", errors.Trace(fmt.Errorf("batch: received non 200 response %d", resp.StatusCode))
		}
		return "", e.Error
	}

	urlResp := &branchURLResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&urlResp); err != nil {
		return "", errors.Trace(err)
	}
	return urlResp.URL, nil
}
