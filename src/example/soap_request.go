package main

import (
	"carefront/libs/erx"
	"fmt"
)

func main() {
	doseSpotService := erx.NewDoseSpotService("", "", "")
	medicationStrengths, err := doseSpotService.SearchForMedicationStrength("Benzoyl Peroxide Topical (topical - cream)")
	if err != nil {
		panic(err.Error())
	}

	fmt.Println(medicationStrengths)
}
