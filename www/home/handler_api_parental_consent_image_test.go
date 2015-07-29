package home

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/www"
)

type mockDataAPI_parentalConsentImage struct {
	api.DataAPI
	proof   *api.ParentalConsentProof
	consent map[int64]*common.ParentalConsent
	updated bool
}

func (a *mockDataAPI_parentalConsentImage) GetPatientIDFromAccountID(accountID int64) (int64, error) {
	return accountID, nil
}

func (a *mockDataAPI_parentalConsentImage) AddMedia(accountID int64, mediaURL, mimeType string) (int64, error) {
	return 1, nil
}

func (a *mockDataAPI_parentalConsentImage) UpsertParentConsentProof(parentPatientID int64, proof *api.ParentalConsentProof) (int64, error) {
	return 1, nil
}

func (a *mockDataAPI_parentalConsentImage) ParentConsentProof(parentPatientID int64) (*api.ParentalConsentProof, error) {
	if a.proof == nil {
		return nil, api.ErrNotFound("proof")
	}
	return a.proof, nil
}

func (a *mockDataAPI_parentalConsentImage) ParentalConsentCompletedForPatient(patientID int64) error {
	a.updated = true
	return nil
}

func (a *mockDataAPI_parentalConsentImage) AllParentalConsent(parentPatientID int64) (map[int64]*common.ParentalConsent, error) {
	return a.consent, nil
}

func multipartBody(imageType string, data []byte) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "example.jpg")
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(data); err != nil {
		return nil, "", err
	}
	if err := writer.WriteField("type", imageType); err != nil {
		return nil, "", err
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return body, writer.FormDataContentType(), nil
}

func TestParentalConsentImageAPIHandler_POST(t *testing.T) {
	dataAPI := &mockDataAPI_parentalConsentImage{}
	signer, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	mediaStore := media.NewStore("XXX", signer, storage.NewTestStore(nil))

	h := newParentalConsentImageAPIHAndler(dataAPI, dispatch.New(), mediaStore)

	account := &common.Account{ID: 1, Role: api.RolePatient}
	ctx := www.CtxWithAccount(context.Background(), account)

	// No consent granted (should not update patient and visits)

	*dataAPI = mockDataAPI_parentalConsentImage{
		proof: &api.ParentalConsentProof{
			SelfiePhotoID:       ptr.Int64(1),
			GovernmentIDPhotoID: ptr.Int64(2),
		},
	}
	body, contentType, err := multipartBody(photoTypeSelfie, []byte("boo"))
	test.OK(t, err)
	r, err := http.NewRequest("POST", "/", body)
	test.OK(t, err)
	r.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, false, dataAPI.updated)

	// Consent granted (should update patient and visits)

	*dataAPI = mockDataAPI_parentalConsentImage{
		proof: &api.ParentalConsentProof{
			SelfiePhotoID:       ptr.Int64(1),
			GovernmentIDPhotoID: ptr.Int64(2),
		},
		consent: map[int64]*common.ParentalConsent{
			2: &common.ParentalConsent{
				Consented: true,
			},
		},
	}
	body, contentType, err = multipartBody(photoTypeSelfie, []byte("boo"))
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", body)
	test.OK(t, err)
	r.Header.Set("Content-Type", contentType)
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, true, dataAPI.updated)
}

func TestParentalConsentImageAPIHandler_GET(t *testing.T) {
	dataAPI := &mockDataAPI_parentalConsentImage{}
	signer, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	mediaStore := media.NewStore("XXX", signer, storage.NewTestStore(nil))

	h := newParentalConsentImageAPIHAndler(dataAPI, dispatch.New(), mediaStore)

	account := &common.Account{ID: 1, Role: api.RolePatient}
	ctx := www.CtxWithAccount(context.Background(), account)

	*dataAPI = mockDataAPI_parentalConsentImage{
		proof: &api.ParentalConsentProof{
			SelfiePhotoID:       ptr.Int64(1),
			GovernmentIDPhotoID: ptr.Int64(2),
		},
	}
	r, err := http.NewRequest("GET", "/", nil)
	test.OK(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	var res parentalConsentImageAPIGETResponse
	test.OK(t, json.NewDecoder(w.Body).Decode(&res))
	test.Assert(t, res.Types != nil, "res.Types must not be nil")
	test.Assert(t, res.Types[photoTypeSelfie] != nil, "selfie image type missing")
	test.Assert(t, res.Types[photoTypeGovernmentID] != nil, "governmentid image type missing")
}
