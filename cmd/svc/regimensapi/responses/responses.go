package responses

import "github.com/sprucehealth/backend/svc/regimens"

// Product represents an individual product associated with the products service
type Product struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ImageURLs  []string `json:"image_urls"`
	ProductURL string   `json:"product_url,omitempty"`
	Prefetched *bool    `json:"prefetched,omitempty"`
}

// ProductList represents a list of products returned by the products service
type ProductList struct {
	Products []*Product `json:"products"`
}

// ProductGETResponse is the response for an API that returns a single product.
type ProductGETResponse struct {
	Product *Product `json:"product"`
}

// RegimenGETRequest represents the data expected to be associated with a successful GET request for the regimen endpoint
type RegimenGETRequest struct {
	AuthToken string `schema:"token"`
}

// RegimenGETResponse represents the data expected to be returned from a successful GET request for the regimen endpoint
type RegimenGETResponse regimens.Regimen

// RegimenPUTRequest represents the data expected to be associated with a successful PUT request for the regimen endpoint
type RegimenPUTRequest struct {
	Regimen         *regimens.Regimen `json:"regimen"`
	Publish         bool              `json:"publish"`
	AllowRestricted bool              `json:"allow_restricted"` // TODO: Remove this if we figure out a better way or move to accounts
}

// RegimenPUTResponse represents the data expected to be returned from a successful PUT call to the regimen endpoint
type RegimenPUTResponse struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	AuthToken string `json:"auth_token"`
}

// RegimenPOSTRequest represents the data expected to be associated with a successful POST request for the regimen endpoint
type RegimenPOSTRequest RegimenPUTRequest

// RegimenPOSTResponse represents the data expected to be returned from a successful POST call to the regimen endpoint
type RegimenPOSTResponse RegimenPUTResponse

// RegimensGETRequest represents the data expected to be associated with a successful GET request for the regimens endpoint
type RegimensGETRequest struct {
	Query string `schema:"q,required"`
}

// RegimensGETResponse represents the data expected to be returned from a successful GET call to the regimens endpoint
type RegimensGETResponse struct {
	Regimens []*regimens.Regimen `json:"regimens"`
}

// MediaPOSTResponse represents the data expected to be returned from a successful POST call to the regimens endpoint
type MediaPOSTResponse struct {
	MediaID  uint64 `json:"id,string"`
	MediaURL string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// MediaGETRequest represents the data expected to be associated with a successful GET request for the media endpoint
type MediaGETRequest struct {
	Width        int  `schema:"width"`
	Height       int  `schema:"height"`
	Crop         bool `schema:"crop"`
	AllowScaleUp bool `schema:"allow_scale_up"`
}

// FoundationGETRequest represents the data excpected to be associated with a successful GET request to the foundation endpoint
type FoundationGETRequest struct {
	MaxResults int `schema:"max_results"`
}

// FoundationGETResponse represents the data excpected to be returned from a successful GET request to the foundation endpoint
type FoundationGETResponse struct {
	FoundationOf []*regimens.Regimen `json:"foundation_of"`
}
