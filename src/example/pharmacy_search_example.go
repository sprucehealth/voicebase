package main

import (
	"carefront/libs/erx"
	"fmt"
	"strconv"

	"os"
)

func main() {

	clinicId, _ := strconv.ParseInt(os.Getenv("DOSESPOT_CLINIC_ID"), 10, 64)
	userId, _ := strconv.ParseInt(os.Getenv("DOSESPOT_USER_ID"), 10, 64)

	doseSpotService := erx.NewDoseSpotService(clinicId, userId, os.Getenv("DOSESPOT_CLINIC_KEY"), nil)
	transmissionErrors, err := doseSpotService.GetTransmissionErrorDetails(userId)

	for _, transmissionError := range transmissionErrors {
		err = doseSpotService.IgnoreAlert(userId, transmissionError.PrescriptionId.Int64())
		fmt.Printf("Error resolved for prescriptionId %d", transmissionError.PrescriptionId.Int64())
		if err != nil {
			panic(err.Error())
		}
	}
	if err != nil {
		panic(err.Error())
	}

}
