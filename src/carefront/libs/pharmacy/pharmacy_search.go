package pharmacy

type PharmacyData struct {
	Id              int64   `json:"id,string,omitempty"`
	Name            string  `json:"name"`
	Address         string  `json:"address"`
	City            string  `json:"city,omitempty"`
	State           string  `json:"state,omitempty"`
	Postal          string  `json:"zip_code,omitempty"`
	Latitude        string  `json:"lat"`
	Longitude       string  `json:"lng"`
	Phone           string  `json:"phone,omitempty"`
	Fax             string  `json:"fax,omitempty"`
	Url             string  `json:"url,omitempty"`
	DistanceInMiles float64 `json:"distance,string,omitempty"`
}

type PharmacySearchAPI interface {
	GetPharmaciesAroundSearchLocation(searchLocationLat, searchLocationLng, searchRadius float64, numResults int64) (pharmacies []*PharmacyData, err error)
}
