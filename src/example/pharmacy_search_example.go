package main

import (
	"carefront/libs/erx"
	"fmt"

	"os"
)

func main() {
	doseSpotService := erx.NewDoseSpotService(os.Getenv("DOSESPOT_CLINIC_ID"), os.Getenv("DOSESPOT_CLINIC_KEY"), os.Getenv("DOSESPOT_USER_ID"), nil)
	prescriptionLogs, err := doseSpotService.GetPrescriptionStatus(4874)
	if err != nil {
		panic(err.Error())
	}

	for _, prescriptionLog := range prescriptionLogs {
		fmt.Println(prescriptionLog.PrescriptionStatus, prescriptionLog.LogTimeStamp, prescriptionLog.AdditionalInfo)
	}
}
