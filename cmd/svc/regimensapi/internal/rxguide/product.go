package rxguide

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/sprucehealth/backend/cmd/svc/regimensapi/internal/media"
	"github.com/sprucehealth/backend/cmd/svc/regimensapi/responses"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/products"
)

const (
	// RXPlaceholderMediaID represents the media ID to associate with images using the RX product placeholder
	RXPlaceholderMediaID          = "rx_placeholder.png"
	rxGuideProductURLFormatString = "/rxguide/%s"
)

// ProductDAL is a utility claass that allows the RXGuide service to conform to the products.Service DAL needs
type productDAL struct {
	svc           Service
	mediaEndpoint string
	webEndpoint   string
	imageURLs     []string
}

// AsProductDAL wraps the provided rxGuide service for use in the products system
func AsProductDAL(svc Service, mediaEndpoint, webEndpoint string) products.DAL {
	return &productDAL{
		svc:           svc,
		mediaEndpoint: mediaEndpoint,
		webEndpoint:   webEndpoint,
		imageURLs:     []string{media.ResizeURL(mediaEndpoint, RXPlaceholderMediaID, 100, 100)},
	}
}

// QueryProducts wraps the QueryRXGuides functionality
func (d *productDAL) QueryProducts(query string, limit int) ([]*products.Product, error) {
	rxGuides, err := d.svc.QueryRXGuides(query, limit)
	if err == ErrNoGuidesFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	i := 0
	prods := make([]*products.Product, len(rxGuides))
	for name, rxg := range rxGuides {
		prods[i] = transformGuide(d.imageURLs, d.webEndpoint, name, rxg)
		i++
	}
	return prods, nil
}

// Product wraps the RXGuide functionality
func (d *productDAL) Product(id string) (*products.Product, error) {
	rxGuide, err := d.svc.RXGuide(id)
	if err == ErrNoGuidesFound {
		return nil, products.ErrNotFound
	}

	return transformGuide(d.imageURLs, d.webEndpoint, rxGuide.GenericName, rxGuide), errors.Trace(err)
}

func transformGuide(imageURLs []string, webEndpoint, name string, r *responses.RXGuide) *products.Product {
	if r == nil {
		return nil
	}
	url, _ := url.Parse(webEndpoint)
	url.Path += fmt.Sprintf(rxGuideProductURLFormatString, strings.ToLower(r.GenericName))
	return &products.Product{
		ID:         r.GenericName,
		Name:       name,
		ImageURLs:  imageURLs,
		ProductURL: url.String(),
	}
}
