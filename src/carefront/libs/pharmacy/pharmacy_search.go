package pharmacy

const (
	PHARMACY_SOURCE_ODDITY      = "oddity"
	PHARMACY_SOURCE_GOOGLE      = "google"
	PHARMACY_SOURCE_SURESCRIPTS = "surescripts"
)

type PharmacyData struct {
	Id              string   `json:"id,omitempty"`
	PatientId       int64    `json:"-"`
	Source          string   `json:"source,omitempty"`
	Name            string   `json:"name"`
	Address         string   `json:"address"`
	City            string   `json:"city,omitempty"`
	State           string   `json:"state,omitempty"`
	Postal          string   `json:"zip_code,omitempty"`
	Country         string   `json:"country,omitempty"`
	Latitude        string   `json:"lat"`
	Longitude       string   `json:"lng"`
	Phone           string   `json:"phone,omitempty"`
	Fax             string   `json:"fax,omitempty"`
	Url             string   `json:"url,omitempty"`
	PharmacyTypes   []string `json:"pharmacy_types,omitempty"`
	DistanceInMiles float64  `json:"distance,string,omitempty"`
}

type PharmacySearchAPI interface {
	GetPharmaciesAroundSearchLocation(searchLocationLat, searchLocationLng, searchRadius float64, numResults int64) (pharmacies []*PharmacyData, err error)
	GetPharmaciesBasedOnTextSearch(textSearch, lat, lng, searchRadius string) (pharmacies []*PharmacyData, err error)
	GetPharmacyBasedOnId(pharmacyId string) (pharmacy *PharmacyData, err error)
}
