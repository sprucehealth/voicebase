// Package mediaproxy implements a caching proxy for 3rd party media.
package mediaproxy

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/validate"
)

const (
	fetchRetryInterval = time.Second
	fetchMaxAttempts   = 10
)

// ErrFetchFailed is returned when fetching the remote image fails
type ErrFetchFailed struct {
	ID        string
	Permanent bool
	Reason    string
	URL       string
}

func (e ErrFetchFailed) Error() string {
	return fmt.Sprintf("mediaproxy: fetch failed id=%s permanent=%t url='%s': %s", e.ID, e.Permanent, e.URL, e.Reason)
}

// Service is a media proxy
type Service struct {
	mediaSvc   *media.ImageService
	dal        DAL
	httpClient *http.Client
}

// DAL is the media metadata storage data layer interface
type DAL interface {
	Get(ids []string) ([]*Media, error)
	Put([]*Media) error
}

// Media represents the metadata for an image
type Media struct {
	ID            string
	URL           string
	Status        Status
	LastFetch     time.Time
	FetchAttempts uint
	FailReason    string
	Width, Height int
	MimeType      string
	Size          int
}

// Status is the status for fetched media
type Status string

const (
	// StatusNotFetched means there hasn't been an attempt to fetch the media yet
	StatusNotFetched Status = ""
	// StatusStored means the remote image has been fetched and stored
	StatusStored Status = "STORED"
	// StatusFailedTemp means the last fetch failed with a temporary error that may be retries (e.g. 500, timeout)
	StatusFailedTemp Status = "FAILED_TEMP"
	// StatusFailedPerm means the last fetch failed with a permanent error (e.g. 404, 403)
	StatusFailedPerm Status = "FAILED_PERM"
)

// New returns a new instance of the media proxy service. If httpClient is nil
// then http.DefaultClient will be used.
func New(mediaSvc *media.ImageService, dal DAL, httpClient *http.Client) *Service {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Service{
		mediaSvc:   mediaSvc,
		dal:        dal,
		httpClient: httpClient,
	}
}

// LookupByURL returns a map of url -> media ID. If a URL is invalid it will
// be filtered out from the returned map.
func (s *Service) LookupByURL(urls []string) (map[string]*Media, error) {
	// Validate the URLs to make sure we only fetch remote
	// resources (not local addresses which could be a security risk).
	for i := 0; i < len(urls); i++ {
		earl := urls[i]
		ok := true
		u, err := url.Parse(earl)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			ok = false
		}
		if ok {
			// Don'to resolve the DNS here as it's expensive. We'll recheck when fetching.
			// But still want to make sure the host is well formed and looks valid (existing TLD).
			_, ok = validate.RemoteHost(u.Host, false)
		}
		if !ok {
			// Remove the bad URL
			urls[i] = urls[len(urls)-1]
			urls = urls[:len(urls)-1]
			i--
			golog.Infof("mediaproxy: filtering out bad url: %s", earl)
		}
	}

	// Use a collision resistant hash to deterministically transform URL to ID
	urlMap := make(map[string]string, len(urls))
	h := md5.New()
	var b []byte
	for _, u := range urls {
		h.Reset()
		h.Write([]byte(u))
		b = h.Sum(b[:0])
		urlMap[u] = strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
	}

	// Lookup existing media
	ids := make([]string, 0, len(urlMap))
	for _, id := range urlMap {
		ids = append(ids, id)
	}
	media, err := s.dal.Get(ids)
	if err != nil {
		return nil, errors.Trace(err)
	}
	mp := make(map[string]*Media, len(media))
	for _, m := range media {
		mp[m.URL] = m
		delete(urlMap, m.URL)
	}

	// Anything left over hasn't yet been seen so store an initial metadata
	if len(urlMap) != 0 {
		toPut := make([]*Media, 0, len(urlMap))
		for url, id := range urlMap {
			m := &Media{
				ID:     id,
				URL:    url,
				Status: StatusNotFetched,
			}
			toPut = append(toPut, m)
			mp[m.URL] = m
		}
		// NOTE: there's a timing issue if who calls for the same URL comes
		// in at the same time but it's likely not worth worrying about.
		if err := s.dal.Put(toPut); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return mp, nil
}

// ImageReader returns a reader and metadata for the requested media image. It either returns
// a stored image if available or attempts to fetch the remote image if not.
func (s *Service) ImageReader(id string, size *media.ImageSize) (io.ReadCloser, *Media, error) {
	// Lookup the media metadata
	med, err := s.dal.Get([]string{id})
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	if len(med) == 0 {
		return nil, nil, errors.Trace(media.ErrNotFound)
	}
	m := med[0]

	// Check the store for the image
	rc, meta, err := s.mediaSvc.GetReader(id, size)
	if err == nil {
		m.Width = meta.Width
		m.Height = meta.Height
		m.Size = int(meta.Size)
		m.MimeType = meta.MimeType
		return rc, m, nil
	} else if errors.Cause(err) != media.ErrNotFound {
		return nil, nil, errors.Trace(err)
	}

	// TODO: possibility for multiple requests for the same media causing redundant
	// fetches. Could avoid this with a global lock, but for now it's not worth the
	// complexity.

	// Attempt to feth the image if it hasn't been flagged as permanently failed
	if m.Status == StatusFailedPerm {
		return nil, nil, errors.Trace(ErrFetchFailed{Reason: m.FailReason, URL: m.URL, Permanent: true})
	}
	if m.Status == StatusFailedTemp {
		// For temporary failures check the last fetch time to rate limit requests
		// Back off the retry by the number of attempts
		iv := fetchRetryInterval * time.Duration(1<<m.FetchAttempts)
		if time.Since(m.LastFetch) < iv {
			return nil, nil, errors.Trace(ErrFetchFailed{Reason: m.FailReason, URL: m.URL, Permanent: false})
		}
	}

	m.LastFetch = time.Now()

	// Validate the host to make sure it doesn't resolve to a local IP
	ur, err := url.Parse(m.URL)
	if err != nil {
		return nil, m, s.fetchFailed(m, fmt.Sprintf("cannot parse URL: %s", err), true)
	}
	if reason, ok := validate.RemoteHost(ur.Host, true); !ok {
		return nil, m, s.fetchFailed(m, fmt.Sprintf("invalid host: %s", reason), true)
	}

	res, err := s.httpClient.Get(m.URL)
	if err != nil {
		// Unknown but likely temporary failure
		return nil, m, s.fetchFailed(m, fmt.Sprintf("request failed: %s", err), false)
	} else if res.StatusCode != http.StatusOK {
		switch res.StatusCode {
		case http.StatusNotFound, http.StatusForbidden, http.StatusUnauthorized:
			// Permanent failure
			return nil, m, s.fetchFailed(m, http.StatusText(res.StatusCode), true)
		}
		// Temporary failure
		return nil, m, s.fetchFailed(m, http.StatusText(res.StatusCode), false)
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		// Temporary failure
		return nil, m, s.fetchFailed(m, fmt.Sprintf("read failed: %s", err), false)
	}
	meta, err = s.mediaSvc.PutReader(m.ID, bytes.NewReader(b))
	if _, ok := errors.Cause(err).(media.ErrInvalidImage); ok {
		// Permanent failure
		return nil, m, s.fetchFailed(m, fmt.Sprintf("bad image: %s", err), true)
	} else if err != nil {
		// Temporary failure
		return nil, m, s.fetchFailed(m, fmt.Sprintf("failed to store: %s", err), false)
	}

	m.Status = StatusStored
	m.FailReason = ""
	m.Size = int(meta.Size)
	m.Width = meta.Width
	m.Height = meta.Height
	m.MimeType = meta.MimeType
	if err := s.dal.Put([]*Media{m}); err != nil {
		golog.Errorf("mediaproxy: failed to update media on success: %s", err)
	}

	// Need to repull the image even though we had it since it could have been processed / resized.
	rc, meta, err = s.mediaSvc.GetReader(id, size)
	if err != nil {
		return nil, m, errors.Trace(err)
	}
	m.Size = int(meta.Size)
	m.Width = meta.Width
	m.Height = meta.Height
	m.MimeType = meta.MimeType
	return rc, m, nil
}

func (s *Service) fetchFailed(m *Media, reason string, perm bool) error {
	m.FailReason = reason
	m.FetchAttempts++
	if m.FetchAttempts >= fetchMaxAttempts || perm {
		m.Status = StatusFailedPerm
	} else {
		m.Status = StatusFailedTemp
	}
	if err := s.dal.Put([]*Media{m}); err != nil {
		golog.Errorf("mediaproxy: failed to update media on failure: %s", err)
	}
	return ErrFetchFailed{ID: m.ID, Reason: reason, URL: m.URL, Permanent: m.Status == StatusFailedPerm}
}
