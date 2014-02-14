package main

import (
	"carefront/libs/erx"

	"os"
)

func main() {
	doseSpotService := erx.NewDoseSpotService(os.Getenv("DOSESPOT_CLINIC_ID"), os.Getenv("DOSESPOT_CLINIC_KEY"), os.Getenv("DOSESPOT_USER_ID"), nil)
	err := doseSpotService.IgnoreAlert(5033)
	if err != nil {
		panic(err.Error())
	}
}
