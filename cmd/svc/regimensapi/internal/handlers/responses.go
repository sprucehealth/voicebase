package handlers

import "github.com/sprucehealth/backend/svc/regimens"

type product struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ImageURLs  []string `json:"image_urls"`
	ProductURL string   `json:"product_url,omitempty"`
}

type productList struct {
	Products []*product `json:"products"`
}

type regimenGETResponse regimens.Regimen

type regimenPUTRequest struct {
	Regimen *regimens.Regimen `json:"regimen"`
	Publish bool              `json:"publish"`
}

type regimenPUTResponse struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	AuthToken string `json:"auth_token"`
}

type regimenPOSTRequest regimenPUTRequest

type regimenPOSTResponse regimenPUTResponse
