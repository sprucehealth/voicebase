package main

import (
	"carefront/libs/erx"
	"fmt"
	"os"
)

func main() {
	doseSpotService := &erx.DoseSpotService{ClinicId: os.Getenv("DOSESPOT_CLINIC_ID"), ClinicKey: os.Getenv("DOSESPOT_CLINIC_KEY"), UserId: os.Getenv("DOSESPOT_USER_ID")}
	treatments, err := doseSpotService.GetMedicationList(1862)
	if err != nil {
		panic(err.Error())
	}
	for _, treatment := range treatments {
		fmt.Println(treatment.PrescriptionStatus)
	}

}
