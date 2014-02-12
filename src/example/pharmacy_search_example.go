package main

import (
	"carefront/libs/erx"
	"fmt"

	"os"
)

func main() {
	doseSpotService := erx.NewDoseSpotService(os.Getenv("DOSESPOT_CLINIC_ID"), os.Getenv("DOSESPOT_CLINIC_KEY"), os.Getenv("DOSESPOT_USER_ID"), nil)
	refillRequests, transactionErrors, err := doseSpotService.GetTransmissionErrorRefillRequestsCount()
	if err != nil {
		panic(err.Error())
	}

	fmt.Println(refillRequests, transactionErrors)
}
