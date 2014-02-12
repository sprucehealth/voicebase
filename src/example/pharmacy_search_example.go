package main

import (
	"carefront/libs/erx"
	"fmt"

	"os"
)

func main() {
	doseSpotService := erx.NewDoseSpotService(os.Getenv("DOSESPOT_CLINIC_ID"), os.Getenv("DOSESPOT_CLINIC_KEY"), os.Getenv("DOSESPOT_USER_ID"), nil)
	transmissionErrors, err := doseSpotService.GetTransmissionErrorDetails()
	if err != nil {
		panic(err.Error())
	}
	for _, transmissionError := range transmissionErrors {
		fmt.Printf("%d\n", transmissionError.DoseSpotPrescriptionId)
	}

}
