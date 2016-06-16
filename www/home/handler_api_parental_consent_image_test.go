package home

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/mediastore"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

type mockDataAPI_parentalConsentImage struct {
	api.DataAPI
	proof    *api.ParentalConsentProof
	consent  map[common.PatientID]*common.ParentalConsent
	updated  bool
	mimeType string
}

func (a *mockDataAPI_parentalConsentImage) GetPatientIDFromAccountID(accountID int64) (common.PatientID, error) {
	return common.NewPatientID(uint64(accountID)), nil
}

func (a *mockDataAPI_parentalConsentImage) AddMedia(accountID int64, mediaURL, mimeType string) (int64, error) {
	a.mimeType = mimeType
	return 1, nil
}

func (a *mockDataAPI_parentalConsentImage) UpsertParentConsentProof(parentPatientID common.PatientID, proof *api.ParentalConsentProof) (int64, error) {
	return 1, nil
}

func (a *mockDataAPI_parentalConsentImage) GetPersonIDByRole(role string, roleID int64) (int64, error) {
	return roleID + 100, nil
}

func (a *mockDataAPI_parentalConsentImage) ParentConsentProof(parentPatientID common.PatientID) (*api.ParentalConsentProof, error) {
	if a.proof == nil {
		return nil, api.ErrNotFound("proof")
	}
	return a.proof, nil
}

func (a *mockDataAPI_parentalConsentImage) ParentalConsentCompletedForPatient(patientID common.PatientID) (bool, error) {
	a.updated = true
	return true, nil
}

func (a *mockDataAPI_parentalConsentImage) AllParentalConsent(parentPatientID common.PatientID) (map[common.PatientID]*common.ParentalConsent, error) {
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
	mediaStore := mediastore.New("XXX", signer, storage.NewTestStore(nil))

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
	body, contentType, err := multipartBody(photoTypeSelfie, testJPEG)
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
		consent: map[common.PatientID]*common.ParentalConsent{
			common.NewPatientID(2): &common.ParentalConsent{
				Consented: true,
			},
		},
	}
	body, contentType, err = multipartBody(photoTypeSelfie, testJPEG)
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", body)
	test.OK(t, err)
	r.Header.Set("Content-Type", contentType)
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, true, dataAPI.updated)
	test.Equals(t, "image/jpeg", dataAPI.mimeType)

	// Bad format

	*dataAPI = mockDataAPI_parentalConsentImage{
		proof: &api.ParentalConsentProof{
			SelfiePhotoID:       ptr.Int64(1),
			GovernmentIDPhotoID: ptr.Int64(2),
		},
		consent: map[common.PatientID]*common.ParentalConsent{
			common.NewPatientID(2): &common.ParentalConsent{
				Consented: true,
			},
		},
	}
	body, contentType, err = multipartBody(photoTypeSelfie, []byte{1, 2, 3})
	test.OK(t, err)
	r, err = http.NewRequest("POST", "/", body)
	test.OK(t, err)
	r.Header.Set("Content-Type", contentType)
	w = httptest.NewRecorder()
	h.ServeHTTP(ctx, w, r)
	test.HTTPResponseCode(t, www.HTTPStatusAPIError, w)
	test.Equals(t, "{\"error\":{\"type\":\"invalid_image\",\"message\":\"Corrupt or unsupported image format\"}}\n", w.Body.String())
}

func TestParentalConsentImageAPIHandler_GET(t *testing.T) {
	dataAPI := &mockDataAPI_parentalConsentImage{}
	signer, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)
	mediaStore := mediastore.New("XXX", signer, storage.NewTestStore(nil))

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

var testJPEG = []byte{
	0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x48,
	0x00, 0x48, 0x00, 0x00, 0xff, 0xe1, 0x00, 0x94, 0x45, 0x78, 0x69, 0x66, 0x00, 0x00, 0x4d, 0x4d,
	0x00, 0x2a, 0x00, 0x00, 0x00, 0x08, 0x00, 0x04, 0x01, 0x1a, 0x00, 0x05, 0x00, 0x00, 0x00, 0x01,
	0x00, 0x00, 0x00, 0x3e, 0x01, 0x1b, 0x00, 0x05, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x46,
	0x01, 0x31, 0x00, 0x02, 0x00, 0x00, 0x00, 0x14, 0x00, 0x00, 0x00, 0x4e, 0x87, 0x69, 0x00, 0x04,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x62, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x48,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x48, 0x00, 0x00, 0x00, 0x01, 0x41, 0x63, 0x6f, 0x72,
	0x6e, 0x20, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x20, 0x34, 0x2e, 0x35, 0x2e, 0x35, 0x00,
	0x00, 0x03, 0xa0, 0x01, 0x00, 0x03, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xa0, 0x02,
	0x00, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0xa0, 0x03, 0x00, 0x04, 0x00, 0x00,
	0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0xff, 0xed, 0x00, 0x38, 0x50, 0x68,
	0x6f, 0x74, 0x6f, 0x73, 0x68, 0x6f, 0x70, 0x20, 0x33, 0x2e, 0x30, 0x00, 0x38, 0x42, 0x49, 0x4d,
	0x04, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x38, 0x42, 0x49, 0x4d, 0x04, 0x25, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x10, 0xd4, 0x1d, 0x8c, 0xd9, 0x8f, 0x00, 0xb2, 0x04, 0xe9, 0x80, 0x09, 0x98,
	0xec, 0xf8, 0x42, 0x7e, 0xff, 0xc0, 0x00, 0x11, 0x08, 0x00, 0x01, 0x00, 0x01, 0x03, 0x01, 0x22,
	0x00, 0x02, 0x11, 0x01, 0x03, 0x11, 0x01, 0xff, 0xc4, 0x00, 0x1f, 0x00, 0x00, 0x01, 0x05, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03,
	0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0xff, 0xc4, 0x00, 0xb5, 0x10, 0x00, 0x02, 0x01,
	0x03, 0x03, 0x02, 0x04, 0x03, 0x05, 0x05, 0x04, 0x04, 0x00, 0x00, 0x01, 0x7d, 0x01, 0x02, 0x03,
	0x00, 0x04, 0x11, 0x05, 0x12, 0x21, 0x31, 0x41, 0x06, 0x13, 0x51, 0x61, 0x07, 0x22, 0x71, 0x14,
	0x32, 0x81, 0x91, 0xa1, 0x08, 0x23, 0x42, 0xb1, 0xc1, 0x15, 0x52, 0xd1, 0xf0, 0x24, 0x33, 0x62,
	0x72, 0x82, 0x09, 0x0a, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x34,
	0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x53, 0x54,
	0x55, 0x56, 0x57, 0x58, 0x59, 0x5a, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6a, 0x73, 0x74,
	0x75, 0x76, 0x77, 0x78, 0x79, 0x7a, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89, 0x8a, 0x92, 0x93,
	0x94, 0x95, 0x96, 0x97, 0x98, 0x99, 0x9a, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9, 0xaa,
	0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7, 0xb8, 0xb9, 0xba, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7, 0xc8,
	0xc9, 0xca, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7, 0xd8, 0xd9, 0xda, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5,
	0xe6, 0xe7, 0xe8, 0xe9, 0xea, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8, 0xf9, 0xfa, 0xff,
	0xc4, 0x00, 0x1f, 0x01, 0x00, 0x03, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b,
	0xff, 0xc4, 0x00, 0xb5, 0x11, 0x00, 0x02, 0x01, 0x02, 0x04, 0x04, 0x03, 0x04, 0x07, 0x05, 0x04,
	0x04, 0x00, 0x01, 0x02, 0x77, 0x00, 0x01, 0x02, 0x03, 0x11, 0x04, 0x05, 0x21, 0x31, 0x06, 0x12,
	0x41, 0x51, 0x07, 0x61, 0x71, 0x13, 0x22, 0x32, 0x81, 0x08, 0x14, 0x42, 0x91, 0xa1, 0xb1, 0xc1,
	0x09, 0x23, 0x33, 0x52, 0xf0, 0x15, 0x62, 0x72, 0xd1, 0x0a, 0x16, 0x24, 0x34, 0xe1, 0x25, 0xf1,
	0x17, 0x18, 0x19, 0x1a, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x43,
	0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5a, 0x63,
	0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6a, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79, 0x7a, 0x82,
	0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89, 0x8a, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97, 0x98, 0x99,
	0x9a, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9, 0xaa, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7,
	0xb8, 0xb9, 0xba, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7, 0xc8, 0xc9, 0xca, 0xd2, 0xd3, 0xd4, 0xd5,
	0xd6, 0xd7, 0xd8, 0xd9, 0xda, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7, 0xe8, 0xe9, 0xea, 0xf2, 0xf3,
	0xf4, 0xf5, 0xf6, 0xf7, 0xf8, 0xf9, 0xfa, 0xff, 0xdb, 0x00, 0x43, 0x00, 0x02, 0x02, 0x02, 0x02,
	0x02, 0x02, 0x03, 0x02, 0x02, 0x03, 0x04, 0x03, 0x03, 0x03, 0x04, 0x05, 0x04, 0x04, 0x04, 0x04,
	0x05, 0x06, 0x05, 0x05, 0x05, 0x05, 0x05, 0x06, 0x08, 0x06, 0x06, 0x06, 0x06, 0x06, 0x06, 0x08,
	0x08, 0x08, 0x08, 0x08, 0x08, 0x08, 0x08, 0x09, 0x09, 0x09, 0x09, 0x09, 0x09, 0x0b, 0x0b, 0x0b,
	0x0b, 0x0b, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c, 0xff, 0xdb, 0x00, 0x43,
	0x01, 0x02, 0x02, 0x02, 0x03, 0x03, 0x03, 0x05, 0x03, 0x03, 0x05, 0x0d, 0x09, 0x07, 0x09, 0x0d,
	0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d,
	0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d,
	0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d, 0x0d,
	0x0d, 0xff, 0xdd, 0x00, 0x04, 0x00, 0x01, 0xff, 0xda, 0x00, 0x0c, 0x03, 0x01, 0x00, 0x02, 0x11,
	0x03, 0x11, 0x00, 0x3f, 0x00, 0xfd, 0xfc, 0xa2, 0x8a, 0x28, 0x03, 0xff, 0xd9,
}
