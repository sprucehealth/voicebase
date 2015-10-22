package models

import "time"

// Doctor represents the information for a particular doctor in carefinder.
type Doctor struct {
	ID                 string
	NPI                string
	IsSpruceDoctor     bool
	FirstName          string
	LastName           string
	Gender             string
	GraduationYear     string
	MedicalSchool      string
	Residency          string
	ProfileImageID     string
	Description        string
	YelpURL            string
	YelpBusinessID     string
	ReviewCount        int
	AverageRating      float64
	ReferralCode       string
	ReferralLink       string
	SpruceProviderID   int64
	InsurancesAccepted []string
	Specialties        []string
	Address            *Address
}

// Address represents the information pertaining to a particular physical address.
type Address struct {
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	Zipcode      string
	Latitude     float64
	Longitude    float64
	Phone        string
}

// Review represents a review for a given doctor
type Review struct {
	DoctorID    string
	Text        string
	Rating      float64
	CreatedDate time.Time
}
