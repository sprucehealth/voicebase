package branch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"

	"github.com/rainycape/memcache"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	baseAPIURL                        = "https://api.branch.io"
	urlSuffix                         = "/v1/url"
	retryAttempts                     = 5
	urlLookupCacheExpireSeconds int32 = 60 * 60 * 24 * 15 // 15 days
)

// Link control parameters keys : https://dev.branch.io/link_configuration/
const (
	// FallbackURL changes the redirect endpoint all platforms - so you don’t have to enable it by platform.
	FallbackURL = "$fallback_url"
	// DesktopURL changes the redirect endpoint on desktops. Default is set to a Branch hosted SMS to download page.
	DesktopURL = "$desktop_url"
	// IOSURL changes the redirect endpoint for iOS. Default is set to the App Store page for your app.
	IOSURL = "$ios_url"
	// AndroidURL changes the redirect endpoint for Android. Default is set to the Play Store page for your app.
	AndroidURL = "$android_url"
)

// MemcacheClient is the interface implemented by a memcached client
type MemcacheClient interface {
	Get(key string) (item *memcache.Item, err error)
	Set(item *memcache.Item) error
}

// Client represents the interface exposed by the branch client
type Client interface {
	URL(linkData map[string]interface{}) (string, error)
}

type client struct {
	key        string
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

// NewClient returns an initialized instance of client
func NewClient(key string) Client {
	return &client{
		key:        key,
		httpClient: &http.Client{},
	}
}

type urlRequest struct {
	BranchKey string                 `json:"branch_key"`
	Data      map[string]interface{} `json:"data"`
}

type urlResponse struct {
	URL string `json:"url"`
}

type errorResponse struct {
	Error *Error `json:"error"`
}

func (bc *client) URL(linkData map[string]interface{}) (string, error) {
	data, err := json.Marshal(&urlRequest{
		BranchKey: bc.key,
		Data:      linkData,
	})
	if err != nil {
		return "", errors.Trace(err)
	}

	var e errorResponse
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
			var urlResp urlResponse
			if err := json.NewDecoder(resp.Body).Decode(&urlResp); err != nil {
				return "", errors.Trace(err)
			}
			return urlResp.URL, nil
		}
	}
	return "", errors.Trace(fmt.Errorf("branch: received non 200 response %s", e.Error.Error()))
}

type memcachedBranchClient struct {
	key    string
	client Client
	mc     MemcacheClient
}

// NewMemcachedClient returns an initialized instance of memcachedBranchClient intended to place a memcache layer infront of the client
func NewMemcachedClient(key string, mc MemcacheClient) Client {
	bc := NewClient(key)
	if mc == nil {
		return bc
	}
	return &memcachedBranchClient{
		key:    key,
		client: bc,
		mc:     mc,
	}
}

func (bc *memcachedBranchClient) URL(linkData map[string]interface{}) (string, error) {
	cacheKey, err := bc.requestHash(linkData)
	if err != nil {
		return "", errors.Trace(err)
	}

	key := applyPrefix(string(cacheKey))
	cachedURL, err := bc.mc.Get(key)
	if err != nil {
		if err != memcache.ErrCacheMiss {
			golog.Errorf("Unable to get cached url for request - %s", err.Error())
		}

		earl, err := bc.client.URL(linkData)
		if err != nil {
			return "", err
		}

		if err := bc.mc.Set(&memcache.Item{
			Key:        key,
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
	data, err := json.Marshal(&urlRequest{
		BranchKey: bc.key,
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
