package worker

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/audioutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/storage"
)

type Uploader interface {
	Upload(contentType, url string) (*models.Media, error)
}

type twilioToS3Uploader struct {
	store            storage.Store
	twilioAccountSID string
	twilioAuthToken  string
}

func newTwilioToS3Uploader(store storage.Store, twilioAccountSID, twilioAuthToken string) Uploader {
	return &twilioToS3Uploader{
		store:            store,
		twilioAccountSID: twilioAccountSID,
		twilioAuthToken:  twilioAuthToken,
	}
}

type errMediaNotFound string

func (e errMediaNotFound) Error() string {
	return string(e)
}

func (t *twilioToS3Uploader) Upload(contentType, url string) (*models.Media, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create GET request for url %q", url)
	}
	req.SetBasicAuth(t.twilioAccountSID, t.twilioAuthToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "GET failed on url %q", url)
	}
	defer res.Body.Close()

	// Note: have to read all the data into memory here because
	// there is no way to know the size of the data when working with a reader
	// via the response body
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {

		if res.StatusCode == 404 {
			return nil, errors.Trace(errMediaNotFound(fmt.Sprintf("twilio media %s not found", url)))
		}

		// Avoid flooding the log
		if len(data) > 1000 {
			data = data[:1000]
		}
		dataStr := string(data)
		if !strings.HasPrefix(res.Header.Get("Content-Type"), "text/") {
			// Avoid non-valid characters from breaking anything in case we get back binary
			dataStr = strconv.Quote(string(data))
		}
		return nil, errors.Trace(fmt.Errorf("Expected status code 2xx when pulling media, got %d: %s", res.StatusCode, dataStr))
	}

	duration, err := audioutil.Duration(bytes.NewReader(data), contentType)
	if err != nil {
		golog.Errorf("Failed to calculate duration of audio: %s", err)
	}

	id, err := media.NewID()
	if err != nil {
		return nil, errors.Trace(err)
	}

	_, err = t.store.Put(id, data, contentType, map[string]string{
		"x-amz-meta-duration-ns": strconv.FormatInt(duration.Nanoseconds(), 10),
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &models.Media{
		ID:       id,
		Type:     contentType,
		Duration: duration,
	}, nil
}
