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
	refillRequestQueue, err := doseSpotService.GetRefillRequestQueueForClinic()
	if err != nil {
		panic(err.Error())
	}

	for _, refillRequestQueueItem := range refillRequestQueue {
		fmt.Printf("%+v\n", refillRequestQueueItem.RequestedPrescription)
	}
}
