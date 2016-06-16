package pharmacy

const (
	PharmacySourceSurescripts = "surescripts"
)

type PharmacyData struct {
	LocalID       int64    `json:"-"`
	SourceID      int64    `json:"id,omitempty,string"`
	PatientID     int64    `json:"-"`
	Source        string   `json:"source,omitempty"`
	NCPDPID       string   `json:"-"`
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
	URL           string   `json:"url,omitpempty"`
	PharmacyTypes []string `json:"pharmacy_types,omitempty"`
}

type PharmacySearchAPI interface {
	GetPharmaciesAroundSearchLocation(searchLocationLat, searchLocationLng, searchRadius float64, numResults int64) (pharmacies []*PharmacyData, err error)
	GetPharmacyFromID(pharmacyID int64) (pharmacy *PharmacyData, err error)
}
