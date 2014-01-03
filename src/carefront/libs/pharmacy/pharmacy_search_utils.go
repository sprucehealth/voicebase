package pharmacy

import (
	"math"
)

type point struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

func GreatCircleDistanceBetweenTwoPoints(p *point, p2 *point) float64 {
	dLat := (p2.Latitude - p.Latitude) * (math.Pi / 180.0)
	dLon := (p2.Longitude - p.Longitude) * (math.Pi / 180.0)

	lat1 := p.Latitude * (math.Pi / 180.0)
	lat2 := p2.Latitude * (math.Pi / 180.0)

	a1 := math.Sin(dLat/2) * math.Sin(dLat/2)
	a2 := math.Sin(dLon/2) * math.Sin(dLon/2) * math.Cos(lat1) * math.Cos(lat2)

	a := a1 + a2

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusInMiles * c
}
