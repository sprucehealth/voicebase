package models

type City struct {
	ID                string
	Name              string
	State             string
	StateAbbreviation string
	Latitude          float64
	Longitude         float64
}

type State struct {
	FullName     string
	Abbreviation string
}

type CareRating struct {
	Title   string
	Bullets []string
	Rating  string
}
