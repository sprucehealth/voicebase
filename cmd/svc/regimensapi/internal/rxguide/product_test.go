package rxguide

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/media"
	rxtest "github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/rxguide/test"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/products"
	"github.com/sprucehealth/backend/test"
)

func TestProductQueryProducts(t *testing.T) {
	q := "myQuery"
	limit := 90
	mediaEndpoint := "http://test.media"
	webEndpoint := "http://test.web"
	rxGuides := []*responses.RXGuide{&responses.RXGuide{GenericName: q}, &responses.RXGuide{GenericName: q + "2"}}
	imageURLs := []string{media.ResizeURL(mediaEndpoint, RXPlaceholderMediaID, 100, 100)}
	svc := &rxtest.RXGuideService{Expector: &mock.Expector{T: t}}
	svc.Expect(mock.NewExpectation(svc.QueryRXGuides, q, limit))
	svc.QueryRXGuidesOutput = [][]*responses.RXGuide{rxGuides}
	dal := AsProductDAL(svc, mediaEndpoint, webEndpoint)
	prods, err := dal.QueryProducts(q, limit)
	test.OK(t, err)
	test.Equals(t, []*products.Product{
		transformGuide(imageURLs, webEndpoint, rxGuides[0]),
		transformGuide(imageURLs, webEndpoint, rxGuides[1]),
	}, prods)
	svc.Finish()
}

func TestProductQueryProductsNoGuidesErr(t *testing.T) {
	q := "myQuery"
	limit := 90
	mediaEndpoint := "http://test.media"
	webEndpoint := "http://test.web"
	svc := &rxtest.RXGuideService{Expector: &mock.Expector{T: t}}
	svc.Expect(mock.NewExpectation(svc.QueryRXGuides, q, limit))
	svc.QueryRXGuidesOutput = [][]*responses.RXGuide{nil}
	svc.QueryRXGuidesErrs = append(svc.QueryRXGuidesErrs, ErrNoGuidesFound)
	dal := AsProductDAL(svc, mediaEndpoint, webEndpoint)
	prods, err := dal.QueryProducts(q, limit)
	test.OK(t, err)
	test.Equals(t, 0, len(prods))
	svc.Finish()
}

func TestProductProduct(t *testing.T) {
	id := "rxID"
	mediaEndpoint := "http://test.media"
	webEndpoint := "http://test.web"
	imageURLs := []string{media.ResizeURL(mediaEndpoint, RXPlaceholderMediaID, 100, 100)}
	rxGuide := &responses.RXGuide{GenericName: id}
	svc := &rxtest.RXGuideService{Expector: &mock.Expector{T: t}}
	svc.Expect(mock.NewExpectation(svc.RXGuide, id))
	svc.RXGuideOutput = append(svc.RXGuideOutput, rxGuide)
	dal := AsProductDAL(svc, mediaEndpoint, webEndpoint)
	prod, err := dal.Product(id)
	test.OK(t, err)
	test.Equals(t, prod, transformGuide(imageURLs, webEndpoint, rxGuide))
	svc.Finish()
}

func TestProductProductNoGuideErr(t *testing.T) {
	id := "rxID"
	mediaEndpoint := "http://test.media"
	webEndpoint := "http://test.web"
	svc := &rxtest.RXGuideService{Expector: &mock.Expector{T: t}}
	svc.Expect(mock.NewExpectation(svc.RXGuide, id))
	svc.RXGuideOutput = append(svc.RXGuideOutput, nil)
	svc.RXGuideErrs = append(svc.RXGuideErrs, ErrNoGuidesFound)
	dal := AsProductDAL(svc, mediaEndpoint, webEndpoint)
	_, err := dal.Product(id)
	test.Equals(t, products.ErrNotFound, err)
	svc.Finish()
}
