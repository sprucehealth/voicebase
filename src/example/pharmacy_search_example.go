package main

import (
	"carefront/libs/erx"
	"os"
)

func main() {
	doseSpotService := &erx.DoseSpotService{ClinicId: os.Getenv("DOSESPOT_CLINIC_ID"), ClinicKey: os.Getenv("DOSESPOT_CLINIC_KEY"), UserId: os.Getenv("DOSESPOT_USER_ID")}
	pharmacies, err := doseSpotService.SearchForPharmacies("", "", "35233", "", []string{"MailOrder"})
	if err != nil {
		panic(err.Error())
	}

}
