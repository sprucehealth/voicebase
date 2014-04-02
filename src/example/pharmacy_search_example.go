package main

import (
	"carefront/libs/erx"
	"os"
)

func main() {

	clinicId := int64(124)
	userId := int64(228)

	doseSpotService := erx.NewDoseSpotService(clinicId, userId, os.Getenv("DOSESPOT_CLINIC_KEY"), nil)
	_, err := doseSpotService.GetPatientDetails(3137)

	if err != nil {
		panic(err.Error())
	}

}
