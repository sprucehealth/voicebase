package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/svc/regimens"
)

type foundationSvc struct {
	foundationOfIDParam         string
	foundationOfMaxResultsParam int
	foundationOfErr             error
	foundationOf                []*regimens.Regimen
}

func (v *foundationSvc) FoundationOf(id string, maxResults int) ([]*regimens.Regimen, error) {
	v.foundationOfMaxResultsParam = maxResults
	v.foundationOfIDParam = id
	return v.foundationOf, v.foundationOfErr
}

func TestFoundationGET(t *testing.T) {
	regRefs := []*regimens.Regimen{
		{
			ID: "test1",
		},
		{
			ID: "test2",
		},
	}
	svc := &foundationSvc{
		foundationOf: regRefs,
	}
	h := NewFoundation(svc)

	// No search results
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/foundation?max_results=5", nil)
	test.OK(t, err)
	ctx := mux.SetVars(context.Background(), map[string]string{"id": "foo"})
	h.ServeHTTP(ctx, w, r)
	expectedResp := &responses.FoundationGETResponse{FoundationOf: regRefs}
	data, err := json.Marshal(expectedResp)
	test.OK(t, err)
	test.HTTPResponseCode(t, http.StatusOK, w)
	test.Equals(t, "foo", svc.foundationOfIDParam)
	test.Equals(t, 5, svc.foundationOfMaxResultsParam)
	test.Equals(t, string(data)+"\n", w.Body.String())
}
