package main

import (
	"carefront/libs/erx"
	"fmt"
)

func main() {
	doseSpotService := erx.NewDoseSpotService("", "", "")
	medication, err := doseSpotService.SelectMedication("Amoxicillin (oral - powder for reconstitution)", "125 mg/5 mL")
	if err != nil {
		panic(err.Error())
	}

	fmt.Println(medication.AdditionalDrugDBIds)
}
