package pharmacy

const (
	PHARMACY_SOURCE_GOOGLE      = "google"
	PHARMACY_SOURCE_SURESCRIPTS = "surescripts"
)

type PharmacyData struct {
	LocalId       int64    `json:"-"`
	SourceId      int64    `json:"id,omitempty,string"`
	PatientId     int64    `json:"-"`
	Source        string   `json:"source,omitempty"`
	Name          string   `json:"name"`
	AddressLine1  string   `json:"address_line_1,omitempty"`
	AddressLine2  string   `json:"address_line_2,omitempty"`
	City          string   `json:"city,omitempty"`
	State         string   `json:"state,omitempty"`
	Postal        string   `json:"zip_code,omitempty"`
	Country       string   `json:"country,omitempty"`
	Latitude      float64  `json:"lat,omitempty"`
	Longitude     float64  `json:"lng,omitempty"`
	Phone         string   `json:"phone,omitempty"`
	Fax           string   `json:"fax,omitempty"`
	Url           string   `json:"url,omitempty"`
	PharmacyTypes []string `json:"pharmacy_types,omitempty"`
}

type PharmacySearchAPI interface {
	GetPharmaciesAroundSearchLocation(searchLocationLat, searchLocationLng, searchRadius float64, numResults int64) (pharmacies []*PharmacyData, err error)
	GetPharmacyFromId(pharmacyId int64) (pharmacy *PharmacyData, err error)
}
