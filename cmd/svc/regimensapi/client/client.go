package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/regimens"
)

// Client defines the methods required to interact with the regimensapi
type Client interface {
	InsertRegimen(r *regimens.Regimen, publish bool) (*responses.RegimenPOSTResponse, error)
	Regimen(regimenID string) (*responses.RegimenGETResponse, error)
}

type regimensAPIClient struct {
	endpoint   string
	httpClient *http.Client
}

// New returns an initialized instance of regimensAPIClient
func New(endpoint string) Client {
	return &regimensAPIClient{endpoint: endpoint, httpClient: http.DefaultClient}
}

func (c *regimensAPIClient) InsertRegimen(r *regimens.Regimen, publish bool) (*responses.RegimenPOSTResponse, error) {
	req := &responses.RegimenPUTRequest{Publish: publish, Regimen: r, AllowRestricted: true}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Trace(err)
	}
	resp, err := c.httpClient.Post(formURL(c.endpoint, "regimen"), "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer resp.Body.Close()

	if err := checkOK(resp); err != nil {
		return nil, errors.Trace(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	postResp := &responses.RegimenPOSTResponse{}
	return postResp, errors.Trace(json.Unmarshal(body, postResp))
}

func (c *regimensAPIClient) Regimen(regimenID string) (*responses.RegimenGETResponse, error) {
	resp, err := c.httpClient.Get(formURL(c.endpoint, "regimen/"+regimenID))
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer resp.Body.Close()

	if err := checkOK(resp); err != nil {
		return nil, errors.Trace(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	getResp := &responses.RegimenGETResponse{}
	return getResp, errors.Trace(json.Unmarshal(body, getResp))
}

func (c *regimensAPIClient) IncrementViewCount(regimenID string) error {
	resp, err := c.httpClient.Get(formURL(c.endpoint, "regimen/"+regimenID+"/view"))
	if err != nil {
		return errors.Trace(err)
	}
	defer resp.Body.Close()

	if err := checkOK(resp); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func formURL(baseEndpoint, path string) string {
	return strings.TrimRight(baseEndpoint, "/") + "/" + path
}

func checkOK(r *http.Response) error {
	if r.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return errors.Trace(fmt.Errorf("Unexpected errors code %d - could not read body: %s", r.StatusCode, err))
		}
		return errors.Trace(fmt.Errorf("Unexpected errors code %d: %s", r.StatusCode, string(body)))
	}
	return nil
}
