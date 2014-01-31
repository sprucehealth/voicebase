package main

import (
	"carefront/common"
	"carefront/libs/erx"
	"carefront/libs/pharmacy"
	"os"
	"time"
)

func main() {

	patient := &common.Patient{}
	patient.FirstName = "Kunal"
	patient.LastName = "Jham"
	patient.Gender = "Male"
	patient.ZipCode = "94115"
	patient.City = "San Francisco"
	patient.State = "CA"
	patient.Phone = "2068773590"
	patient.PhoneType = "Home"
	patient.PatientAddress = &common.Address{}
	patient.PatientAddress.AddressLine1 = "1510 Eddy Street"
	patient.PatientAddress.AddressLine2 = "Apt 1112"
	patient.PatientAddress.City = "San Francisco"
	patient.PatientAddress.State = "CA"
	patient.PatientAddress.ZipCode = "94115"
	patient.Dob = time.Date(2013, 11, 01, 0, 0, 0, 0, time.Local)
	patient.Pharmacy = &pharmacy.PharmacyData{}
	patient.Pharmacy.Id = "39203"

	treatment := &common.Treatment{}
	treatment.DrugDBIds = make(map[string]string)
	treatment.DrugDBIds[erx.LexiDrugSynId] = "93147"
	treatment.DrugDBIds[erx.LexiGenProductId] = "19014"
	treatment.DrugDBIds[erx.LexiSynonymTypeId] = "59"
	treatment.NumberRefills = 2
	treatment.DispenseUnitId = 19
	treatment.DispenseValue = 10
	treatment.DrugInternalName = "Benzoyl Peroxide Topical (topical - kit)"
	treatment.DosageStrength = "8%"
	treatment.DaysSupply = 90
	treatment.PatientInstructions = "Take twice a day"

	treatment2 := &common.Treatment{}
	treatment2.DrugDBIds = make(map[string]string)
	treatment2.DrugDBIds[erx.LexiDrugSynId] = "19810"
	treatment2.DrugDBIds[erx.LexiGenProductId] = "4976"
	treatment2.DrugDBIds[erx.LexiSynonymTypeId] = "59"
	treatment2.NumberRefills = 2
	treatment2.DispenseUnitId = 19
	treatment2.DispenseValue = 10
	treatment2.DrugInternalName = "Benzoyl Peroxide Topical (topical - gel)"
	treatment2.DosageStrength = "6%"
	treatment2.DaysSupply = 90
	treatment2.PatientInstructions = "#2 Take twice a day"

	doseSpotService := &erx.DoseSpotService{ClinicId: os.Getenv("DOSESPOT_CLINIC_ID"), ClinicKey: os.Getenv("DOSESPOT_CLINIC_KEY"), UserId: os.Getenv("DOSESPOT_USER_ID")}
	err := doseSpotService.StartPrescribingPatient(patient, []*common.Treatment{treatment, treatment2})

	if err != nil {
		panic(err.Error())
	}

	// now send the prescriptions to the pharmacy
	err = doseSpotService.SendMultiplePrescriptions(patient, []*common.Treatment{treatment, treatment2})
	if err != nil {
		panic(err.Error())
	}
}
