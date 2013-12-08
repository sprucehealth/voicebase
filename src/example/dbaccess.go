package main

import (
	"carefront/libs/maps"
	"fmt"
)

func main() {
	googleMapsService := maps.GoogleMapsService(0)
	cityStateInfo, err := googleMapsService.ConvertZipcodeToCityState("90210")
	if err != nil {
		panic(err.Error())
	}

	fmt.Println(cityStateInfo.LongCityName)
	fmt.Println(cityStateInfo.LongStateName)

}
