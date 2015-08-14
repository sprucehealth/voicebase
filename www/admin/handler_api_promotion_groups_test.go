package admin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type mockedDataAPIPromotionGroupsHandler struct {
	api.DataAPI
	promotionGroups    []*common.PromotionGroup
	promotionGroupsErr error
}

func (m *mockedDataAPIPromotionGroupsHandler) PromotionGroups() ([]*common.PromotionGroup, error) {
	return m.promotionGroups, m.promotionGroupsErr
}

func TestPromotionGroupsHandlerGETPromotionGroupsErr(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?", nil)
	test.OK(t, err)
	handler := newPromotionGroupsHandler(&mockedDataAPIPromotionGroupsHandler{
		promotionGroupsErr: errors.New("Foo"),
	})
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusInternalServerError, responseWriter.Code)
}

func TestPromotionGroupsHandlerGETPromotionGroups(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?", nil)
	test.OK(t, err)
	handler := newPromotionGroupsHandler(&mockedDataAPIPromotionGroupsHandler{
		promotionGroups: []*common.PromotionGroup{
			&common.PromotionGroup{
				ID:               1,
				Name:             "Foo",
				MaxAllowedPromos: 5,
			},
			&common.PromotionGroup{
				ID:               2,
				Name:             "Bar",
				MaxAllowedPromos: 1,
			},
		},
	})
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, &PromotionGroupsGETResponse{PromotionGroups: []*responses.PromotionGroup{
		&responses.PromotionGroup{
			ID:               1,
			Name:             "Foo",
			MaxAllowedPromos: 5,
		},
		&responses.PromotionGroup{
			ID:               2,
			Name:             "Bar",
			MaxAllowedPromos: 1,
		},
	}})
	handler.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, http.StatusOK, responseWriter.Code)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}
