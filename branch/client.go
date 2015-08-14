package branch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"

	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"gopkgs.com/memcache.v2"
)

const (
	baseAPIURL                        = "https://api.branch.io"
	urlSuffix                         = "/v1/url"
	retryAttempts                     = 5
	urlLookupCacheExpireSeconds int32 = 60 * 60 * 24 * 15 // 15 days
)

type memcacheClient interface {
	Get(key string) (item *memcache.Item, err error)
	Set(item *memcache.Item) error
}

// Client represents the interface exposed by the branch client
type Client interface {
	URL(linkData map[string]interface{}) (string, error)
}

type client struct {
	branchKey  string
	httpClient *http.Client
}

// Error represents the branch error type to wrap and annotate
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("branch: [%d] %s", e.Code, e.Message)
}

// NewBranchClient returns an initialized instance of client
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
	var err error
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

	var e struct {
		Error *Error `json:"error"`
	}
	for i := 0; i < retryAttempts; i++ {
		resp, err := bc.httpClient.Post(baseAPIURL+urlSuffix, "application/json", bytes.NewReader(data))
		if err != nil {
			return "", errors.Trace(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
				return "", errors.Trace(fmt.Errorf("branch: received non 200 response %d", resp.StatusCode))
			}
			golog.Errorf("branch: received non 200 response %d - %d retries remaining - %s", resp.StatusCode, retryAttempts, e.Error.Error())
		} else {
			urlResp := &branchURLResponse{}
			if err := json.NewDecoder(resp.Body).Decode(&urlResp); err != nil {
				return "", errors.Trace(err)
			}
			return urlResp.URL, nil
		}
	}
	return "", errors.Trace(fmt.Errorf("branch: received non 200 response %s", e.Error.Error()))
}

type memcachedBranchClient struct {
	branchKey string
	client    Client
	mc        memcacheClient
}

// NewMemcachedBranchClient returns an initialized instance of memcachedBranchClient intended to place a memcache layer infront of the client
func NewMemcachedBranchClient(branchKey string, mc memcacheClient) Client {
	bc := NewBranchClient(branchKey)
	if mc == nil {
		return bc
	}
	return &memcachedBranchClient{
		branchKey: branchKey,
		client:    bc,
		mc:        mc,
	}
}

func (bc *memcachedBranchClient) URL(linkData map[string]interface{}) (string, error) {
	cacheKey, err := bc.requestHash(linkData)
	if err != nil {
		return "", errors.Trace(err)
	}

	cachedURL, err := bc.mc.Get(applyPrefix(string(cacheKey)))
	if err != nil {
		if err != memcache.ErrCacheMiss {
			golog.Errorf("Unable to get cached url for request - %s", err.Error())
		}

		earl, err := bc.client.URL(linkData)
		if err != nil {
			return "", err
		}

		if err := bc.mc.Set(&memcache.Item{
			Key:        applyPrefix(string(cacheKey)),
			Value:      []byte(earl),
			Expiration: urlLookupCacheExpireSeconds,
		}); err != nil {
			golog.Errorf("Failed to cache url info: %s", err.Error())
		}
		return earl, nil
	}
	return string(cachedURL.Value), nil
}

// requestHash returns a hash of the provided map using fnv
func (bc *memcachedBranchClient) requestHash(linkData map[string]interface{}) ([]byte, error) {
	data, err := json.Marshal(struct {
		BranchKey string                 `json:"branch_key"`
		Data      map[string]interface{} `json:"data"`
	}{
		BranchKey: bc.branchKey,
		Data:      linkData,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	hash := fnv.New32()
	io.WriteString(hash, string(data))
	return hash.Sum(nil), nil
}

func applyPrefix(key string) string {
	return `branchURL:` + key
}
