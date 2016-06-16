package erx

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/pharmacy"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dosespot"
	"github.com/sprucehealth/backend/libs/golog"
)

type DoseSpotService struct {
	*dosespot.Service
}

const (
	resultOk = "OK"
	prn      = "PRN"
)

type ByLogTimeStamp []*PrescriptionLog

func (a ByLogTimeStamp) Len() int      { return len(a) }
func (a ByLogTimeStamp) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLogTimeStamp) Less(i, j int) bool {
	return a[i].LogTimestamp.Before(a[j].LogTimestamp)
}

func NewDoseSpotService(clinicID, userID int64, clinicKey, soapEndpoint, apiEndpoint string, statsRegistry metrics.Registry) ERxAPI {
	return &DoseSpotService{
		Service: dosespot.New(clinicID, userID, clinicKey, soapEndpoint, apiEndpoint, statsRegistry),
	}
}

func (d *DoseSpotService) SendMultiplePrescriptions(clinicianID int64, patient *common.Patient, treatments []*common.Treatment) ([]*common.Treatment, error) {
	prescriptionIDs := make([]int64, 0, len(treatments))
	prescriptionIDToTreatmentMapping := make(map[int64]*common.Treatment, len(treatments))
	for _, treatment := range treatments {
		if treatment.ERx.PrescriptionID.Int64() == 0 {
			continue
		}
		prescriptionIDs = append(prescriptionIDs, treatment.ERx.PrescriptionID.Int64())
		prescriptionIDToTreatmentMapping[treatment.ERx.PrescriptionID.Int64()] = treatment
	}

	res, err := d.Service.SendMultiplePrescriptions(clinicianID, patient.ERxPatientID.Int64(), prescriptionIDs)
	if err != nil {
		return nil, err
	}

	var unSuccessfulTreatments []*common.Treatment
	for _, prescriptionResult := range res {
		if prescriptionResult.ResultCode != resultOk {
			unSuccessfulTreatments = append(unSuccessfulTreatments, prescriptionIDToTreatmentMapping[int64(prescriptionResult.PrescriptionID)])
			golog.Errorf("Error sending prescription with id %d : %s", prescriptionResult.PrescriptionID, prescriptionResult.ResultDescription)
		}
	}

	return unSuccessfulTreatments, nil
}

func populatePatientForDoseSpot(currentPatient *common.Patient) (*dosespot.Patient, error) {
	newPatient := &dosespot.Patient{
		PatientID:   currentPatient.ERxPatientID.Int64(),
		FirstName:   currentPatient.FirstName,
		MiddleName:  currentPatient.MiddleName,
		LastName:    currentPatient.LastName,
		Suffix:      currentPatient.Suffix,
		Prefix:      currentPatient.Prefix,
		Email:       currentPatient.Email,
		DateOfBirth: dosespot.SpecialDateTime{DateTime: currentPatient.DOB.ToTime(), DateTimeElementName: "DateOfBirth"},
		Gender:      currentPatient.Gender,
	}

	if len(currentPatient.PhoneNumbers) > 0 {
		newPatient.PrimaryPhone = currentPatient.PhoneNumbers[0].Phone.String()
		newPatient.PrimaryPhoneType = currentPatient.PhoneNumbers[0].Type.String()

		if len(currentPatient.PhoneNumbers) > 1 {
			newPatient.PhoneAdditional1 = currentPatient.PhoneNumbers[1].Phone.String()
			newPatient.PhoneAdditionalType1 = currentPatient.PhoneNumbers[1].Type.String()
		}

		if len(currentPatient.PhoneNumbers) > 2 {
			newPatient.PhoneAdditional2 = currentPatient.PhoneNumbers[2].Phone.String()
			newPatient.PhoneAdditionalType2 = currentPatient.PhoneNumbers[2].Type.String()
		}
	}

	if currentPatient.PatientAddress != nil {
		newPatient.Address1 = currentPatient.PatientAddress.AddressLine1
		newPatient.Address2 = currentPatient.PatientAddress.AddressLine2
		newPatient.City = currentPatient.PatientAddress.City
		newPatient.ZipCode = currentPatient.PatientAddress.ZipCode
		newPatient.State = currentPatient.PatientAddress.State
	}

	if currentPatient.ERxPatientID.Int64() != 0 {
		newPatient.PatientID = currentPatient.ERxPatientID.Int64()
	}

	return newPatient, nil
}

// func ensurePatientInformationIsConsistent(currentPatient *common.Patient, patientUpdatesFromDoseSpot []*patientUpdate) error {
// 	if len(patientUpdatesFromDoseSpot) != 1 {
// 		return fmt.Errorf("Expected a single patient to be returned from dosespot instead got back %d", len(patientUpdatesFromDoseSpot))
// 	}

// 	patientFromDoseSpot := patientUpdatesFromDoseSpot[0].Patient

// 	if currentPatient.FirstName != patientFromDoseSpot.FirstName {
// 		return errors.New("PATIENT_INFO_MISMATCH: firstName")
// 	}

// 	if currentPatient.LastName != patientFromDoseSpot.LastName {
// 		return errors.New("PATIENT_INFO_MISTMATCH: lastName")
// 	}

// 	if currentPatient.MiddleName != patientFromDoseSpot.MiddleName {
// 		return errors.New("PATIENT_INFO_MISTMATCH: middleName")
// 	}

// 	if currentPatient.Suffix != patientFromDoseSpot.Suffix {
// 		return errors.New("PATIENT_INFO_MISTMATCH: suffix")
// 	}

// 	if currentPatient.Prefix != patientFromDoseSpot.Prefix {
// 		return errors.New("PATIENT_INFO_MISTMATCH: prefix")
// 	}

// 	if currentPatient.LastName != patientFromDoseSpot.LastName {
// 		return errors.New("PATIENT_INFO_MISTMATCH: lastName")
// 	}

// 	// lets compare the day, month and year components
// 	doseSpotPatientDOBYear, doseSpotPatientDOBMonth, doseSpotPatientDay := patientFromDoseSpot.DateOfBirth.DateTime.Date()

// 	if currentPatient.DOB.Day != doseSpotPatientDay || currentPatient.DOB.Month != int(doseSpotPatientDOBMonth) || currentPatient.DOB.Year != doseSpotPatientDOBYear {
// 		return fmt.Errorf("PATIENT_INFO_MISTMATCH: dob %+v %+v", currentPatient.DOB, patientFromDoseSpot.DateOfBirth.DateTime)
// 	}

// 	if strings.ToLower(currentPatient.Gender) != strings.ToLower(patientFromDoseSpot.Gender) {
// 		return errors.New("PATIENT_INFO_MISTMATCH: gender")
// 	}

// 	if currentPatient.Email != patientFromDoseSpot.Email {
// 		return errors.New("PATIENT_INFO_MISTMATCH: email")
// 	}

// 	if currentPatient.PatientAddress.AddressLine1 != patientFromDoseSpot.Address1 {
// 		return errors.New("PATIENT_INFO_MISTMATCH: address1")
// 	}

// 	if currentPatient.PatientAddress.AddressLine2 != patientFromDoseSpot.Address2 {
// 		return errors.New("PATIENT_INFO_MISTMATCH: email")
// 	}

// 	if currentPatient.PatientAddress.City != patientFromDoseSpot.City {
// 		return errors.New("PATIENT_INFO_MISTMATCH: city")
// 	}

// 	if strings.ToLower(currentPatient.PatientAddress.State) != strings.ToLower(patientFromDoseSpot.State) {
// 		return errors.New("PATIENT_INFO_MISTMATCH: state")
// 	}

// 	if currentPatient.PatientAddress.ZipCode != patientFromDoseSpot.ZipCode {
// 		return errors.New("PATIENT_INFO_MISTMATCH: zipCode")
// 	}

// 	if currentPatient.PhoneNumbers[0].Phone.String() != patientFromDoseSpot.PrimaryPhone {
// 		return errors.New("PATIENT_INFO_MISTMATCH: primaryPhone")
// 	}

// 	if currentPatient.PhoneNumbers[0].Type.String() != patientFromDoseSpot.PrimaryPhoneType {
// 		return errors.New("PATIENT_INFO_MISTMATCH: primaryPhoneType")
// 	}

// 	return nil
// }

func (d *DoseSpotService) UpdatePatientInformation(clinicianID int64, patient *common.Patient) error {
	dsPatient, err := populatePatientForDoseSpot(patient)
	if err != nil {
		return err
	}
	updates, err := d.Service.UpdatePatientInformation(clinicianID, dsPatient, patient.Pharmacy.SourceID)
	if err != nil {
		return err
	}

	// if err := ensurePatientInformationIsConsistent(currentPatient, response.PatientUpdates); err != nil {
	// 	return err
	// }

	// populate the prescription id into the patient object
	patient.ERxPatientID = encoding.DeprecatedNewObjectID(updates[0].Patient.PatientID)
	return nil
}

func (d *DoseSpotService) StartPrescribingPatient(clinicianID int64, patient *common.Patient, treatments []*common.Treatment, pharmacySourceID int64) error {
	dsPatient, err := populatePatientForDoseSpot(patient)
	if err != nil {
		return err
	}

	prescriptions := make([]*dosespot.Prescription, len(treatments))
	for i, treatment := range treatments {
		lexiDrugSynIDInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiDrugSynID], 0, 64)
		lexiGenProductIDInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiGenProductID], 0, 64)
		lexiSynonymTypeIDInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiSynonymTypeID], 0, 64)

		patientPrescription := &dosespot.Prescription{
			LexiDrugSynID:     lexiDrugSynIDInt,
			LexiGenProductID:  lexiGenProductIDInt,
			LexiSynonymTypeID: lexiSynonymTypeIDInt,
			Refills:           treatment.NumberRefills.Int64(),
			Dispense:          treatment.DispenseValue.String(),
			DaysSupply:        treatment.DaysSupply.Int64(),
			DispenseUnitID:    treatment.DispenseUnitID.Int64(),
			Instructions:      treatment.PatientInstructions,
			NoSubstitutions:   !treatment.SubstitutionsAllowed,
			PharmacyNotes:     treatment.PharmacyNotes,
			PharmacyID:        pharmacySourceID,
		}

		if treatment.ERx != nil && treatment.ERx.ErxReferenceNumber != "" {
			patientPrescription.RxReferenceNumber = treatment.ERx.ErxReferenceNumber
		}

		prescriptions[i] = patientPrescription
	}

	updates, err := d.Service.StartPrescribingPatient(clinicianID, dsPatient, prescriptions, pharmacySourceID)
	if err != nil {
		return err
	}

	// if err := ensurePatientInformationIsConsistent(currentPatient, response.PatientUpdates); err != nil {
	// 	return err
	// }

	// populate the prescription id into the patient object
	patient.ERxPatientID = encoding.DeprecatedNewObjectID(updates[0].Patient.PatientID)

	// go through and assign medication ids to all prescriptions
	for _, patientUpdate := range updates {
		for _, medication := range patientUpdate.Medications {
			for _, treatment := range treatments {
				lexiDrugSynIDInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiDrugSynID], 0, 64)
				lexiGenProductIDInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiGenProductID], 0, 64)
				lexiSynonymTypeIDInt, _ := strconv.ParseInt(treatment.DrugDBIDs[LexiSynonymTypeID], 0, 64)
				if medication.LexiDrugSynID == lexiDrugSynIDInt &&
					medication.LexiGenProductID == lexiGenProductIDInt &&
					medication.LexiSynonymTypeID == lexiSynonymTypeIDInt {
					if treatment.ERx == nil {
						treatment.ERx = &common.ERxData{}
					}
					treatment.ERx.PrescriptionID = encoding.DeprecatedNewObjectID(medication.DoseSpotPrescriptionID)
				}
			}
		}
	}

	return nil
}

func (d *DoseSpotService) SearchForPharmacies(clinicianID int64, city, state, zipcode, name string, pharmacyTypes []string) ([]*pharmacy.PharmacyData, error) {
	pharmacies, err := d.Service.SearchForPharmacies(clinicianID, city, state, zipcode, name, pharmacyTypes)
	if err != nil {
		return nil, err
	}
	pharms := make([]*pharmacy.PharmacyData, len(pharmacies))
	for i, p := range pharmacies {
		pharms[i] = &pharmacy.PharmacyData{
			SourceID:      p.PharmacyID,
			AddressLine1:  p.Address1,
			AddressLine2:  p.Address2,
			City:          p.City,
			State:         p.State,
			Name:          p.StoreName,
			Fax:           p.PrimaryFax,
			Phone:         p.PrimaryPhone,
			Postal:        p.ZipCode,
			Source:        pharmacy.PharmacySourceSurescripts,
			PharmacyTypes: p.Specialties,
		}
	}
	return pharms, nil
}

func (d *DoseSpotService) GetPrescriptionStatus(clincianID int64, prescriptionID int64) ([]*PrescriptionLog, error) {
	resLog, err := d.Service.GetPrescriptionStatus(clincianID, prescriptionID)
	if err != nil {
		return nil, err
	}
	prescriptionLogs := make([]*PrescriptionLog, len(resLog))
	for i, logDetails := range resLog {
		prescriptionLogs[i] = &PrescriptionLog{
			LogTimestamp:       logDetails.DateTimeStamp.DateTime,
			PrescriptionStatus: logDetails.Status,
			AdditionalInfo:     logDetails.AdditionalInfo,
		}
	}
	sort.Sort(sort.Reverse(ByLogTimeStamp(prescriptionLogs)))
	return prescriptionLogs, nil
}

func (d *DoseSpotService) GetTransmissionErrorDetails(clinicianID int64) ([]*common.Treatment, error) {
	details, err := d.Service.GetTransmissionErrorDetails(clinicianID)

	medicationsWithErrors := make([]*common.Treatment, len(details))
	for i, transmissionError := range details {
		dispenseValueFloat, _ := strconv.ParseFloat(transmissionError.Medication.Dispense, 64)
		medicationsWithErrors[i] = &common.Treatment{
			ERx: &common.ERxData{
				ErxMedicationID:       encoding.DeprecatedNewObjectID(transmissionError.Medication.MedicationID),
				PrescriptionID:        encoding.DeprecatedNewObjectID(transmissionError.Medication.DoseSpotPrescriptionID),
				PrescriptionStatus:    transmissionError.Medication.Status,
				ErxSentDate:           &transmissionError.Medication.DatePrescribed.DateTime,
				TransmissionErrorDate: &transmissionError.ErrorDateTimeStamp.DateTime,
				ErxReferenceNumber:    transmissionError.Medication.RxReferenceNumber,
				ErxPharmacyID:         transmissionError.Medication.PharmacyID,
			},
			DrugDBIDs: map[string]string{
				LexiGenProductID:  strconv.FormatInt(transmissionError.Medication.LexiGenProductID, 10),
				LexiSynonymTypeID: strconv.FormatInt(transmissionError.Medication.LexiSynonymTypeID, 10),
				LexiDrugSynID:     strconv.FormatInt(transmissionError.Medication.LexiDrugSynID, 10),
			},
			DispenseUnitID:       encoding.DeprecatedNewObjectID(transmissionError.Medication.DispenseUnitID),
			StatusDetails:        transmissionError.ErrorDetails,
			DrugName:             transmissionError.Medication.DrugName,
			DosageStrength:       transmissionError.Medication.Strength,
			DaysSupply:           transmissionError.Medication.DaysSupply,
			DispenseValue:        encoding.HighPrecisionFloat64(dispenseValueFloat),
			PatientInstructions:  transmissionError.Medication.Instructions,
			PharmacyNotes:        transmissionError.Medication.PharmacyNotes,
			SubstitutionsAllowed: !transmissionError.Medication.NoSubstitutions,
		}

		// we expect the refills to be a number so if it errors out this is expected
		medicationsWithErrors[i].NumberRefills, err = encoding.NullInt64FromString(transmissionError.Medication.Refills)
		if err != nil {
			return nil, err
		}
	}

	return medicationsWithErrors, nil
}

func (d *DoseSpotService) GetPatientDetails(erxPatientID int64) (*common.Patient, error) {
	patientUpdate, err := d.Service.GetPatientDetails(erxPatientID)
	if err != nil {
		return nil, err
	}
	if patientUpdate == nil {
		return nil, nil
	}

	// not worrying about suffix/prefix for now
	patientPhoneNumberType, err := common.ParsePhoneNumberType(patientUpdate.Patient.PrimaryPhoneType)
	if err != nil {
		return nil, err
	}
	newPatient := &common.Patient{
		ERxPatientID: encoding.DeprecatedNewObjectID(patientUpdate.Patient.PatientID),
		FirstName:    patientUpdate.Patient.FirstName,
		LastName:     patientUpdate.Patient.LastName,
		Gender:       patientUpdate.Patient.Gender,
		PatientAddress: &common.Address{
			AddressLine1: patientUpdate.Patient.Address1,
			AddressLine2: patientUpdate.Patient.Address2,
			City:         patientUpdate.Patient.City,
			ZipCode:      patientUpdate.Patient.ZipCode,
			State:        patientUpdate.Patient.State,
		},
		Email:   patientUpdate.Patient.Email,
		ZipCode: patientUpdate.Patient.ZipCode,
		DOB:     encoding.NewDateFromTime(patientUpdate.Patient.DateOfBirth.DateTime),
		PhoneNumbers: []*common.PhoneNumber{
			{
				Phone: parsePhone(patientUpdate.Patient.PrimaryPhone),
				Type:  patientPhoneNumberType,
			},
		},
	}

	if patientUpdate.Patient.PhoneAdditional1 != "" {
		patientAdditionalPhoneNumberType1, err := common.ParsePhoneNumberType(patientUpdate.Patient.PrimaryPhoneType)
		if err != nil {
			return nil, err
		}
		newPatient.PhoneNumbers = append(newPatient.PhoneNumbers, &common.PhoneNumber{
			Phone: parsePhone(patientUpdate.Patient.PhoneAdditional1),
			Type:  patientAdditionalPhoneNumberType1,
		})
	}

	if patientUpdate.Patient.PhoneAdditional2 != "" {
		patientAdditionalPhoneNumberType2, err := common.ParsePhoneNumberType(patientUpdate.Patient.PrimaryPhoneType)
		if err != nil {
			return nil, err
		}
		newPatient.PhoneNumbers = append(newPatient.PhoneNumbers, &common.PhoneNumber{
			Phone: parsePhone(patientUpdate.Patient.PhoneAdditional2),
			Type:  patientAdditionalPhoneNumberType2,
		})
	}

	if len(patientUpdate.Pharmacies) > 0 {
		newPatient.Pharmacy = &pharmacy.PharmacyData{
			Source:       pharmacy.PharmacySourceSurescripts,
			SourceID:     patientUpdate.Pharmacies[0].PharmacyID,
			Name:         patientUpdate.Pharmacies[0].StoreName,
			AddressLine1: patientUpdate.Pharmacies[0].Address1,
			AddressLine2: patientUpdate.Pharmacies[0].Address2,
			City:         patientUpdate.Pharmacies[0].City,
			State:        patientUpdate.Pharmacies[0].State,
			Postal:       patientUpdate.Pharmacies[0].ZipCode,
			Phone:        patientUpdate.Pharmacies[0].PrimaryPhone,
			Fax:          patientUpdate.Pharmacies[0].PrimaryFax,
		}
	}

	return newPatient, nil
}

func parsePhone(phoneNumber string) common.Phone {
	p, err := common.ParsePhone(phoneNumber)
	if err != nil {
		golog.Errorf("Unable to parse phone number for dosespot patient from string: %s", err)
	}
	return p
}

func (d *DoseSpotService) GetRefillRequestQueueForClinic(clinicianID int64) ([]*common.RefillRequestItem, error) {
	queue, err := d.Service.GetRefillRequestQueueForClinic(clinicianID)
	if err != nil {
		return nil, err
	}

	refillRequestQueue := make([]*common.RefillRequestItem, len(queue))
	// translate each of the request queue items into the object to return
	for i, refillRequest := range queue {
		refillRequestQueue[i] = &common.RefillRequestItem{
			RxRequestQueueItemID:      refillRequest.RxRequestQueueItemID,
			ReferenceNumber:           refillRequest.ReferenceNumber,
			PharmacyRxReferenceNumber: refillRequest.PharmacyRxReferenceNumber,
			RequestedRefillAmount:     refillRequest.RequestedRefillAmount,
			ErxPatientID:              refillRequest.PatientID,
			PatientAddedForRequest:    refillRequest.PatientAddedForRequest,
			RequestDateStamp:          refillRequest.RequestDateStamp.DateTime,
			ClinicianID:               refillRequest.Clinician.ClinicianID,
			RequestedPrescription:     convertMedicationIntoTreatment(refillRequest.RequestedPrescription),
			DispensedPrescription:     convertMedicationIntoTreatment(refillRequest.DispensedPrescription),
		}

		// FIX: We will read the number refill values from RefillRequest.RequestedRefillAmount for now
		// due to the discrepancy on Dosespot's end with this value and the RefillRequested.RequestedPrescription.Refills value.
		// In theory, these two values should be the same. Also, its possible for RequestedRefillAmount to indicate "PRN"
		// or a number. In the event it indicates "PRN" we will handle it via a -1 on our end to indicate this. If its any value other than a number
		// or "PRN", we error out given that the number is not parseable.
		if refillRequest.RequestedRefillAmount == prn {
			refillRequestQueue[i].RequestedPrescription.NumberRefills = encoding.NullInt64{IsValid: true, Int64Value: -1}
		} else {
			refillRequestQueue[i].RequestedPrescription.NumberRefills, err = encoding.NullInt64FromString(refillRequest.RequestedRefillAmount)
			if err != nil {
				return nil, err
			}
		}

		// FIX: the refill quantity for dispensed and requested prescription are expected to be the same. So enforcing that until Dosespot
		// has a fix to ensure that all three (RequestedRefillAmount, RequestedPrescription.Refills, DispensedPrescription.Refills) are the same
		refillRequestQueue[i].DispensedPrescription.NumberRefills = refillRequestQueue[i].RequestedPrescription.NumberRefills

	}

	return refillRequestQueue, nil
}

func (d *DoseSpotService) GetPharmacyDetails(pharmacyID int64) (*pharmacy.PharmacyData, error) {
	details, err := d.Service.GetPharmacyDetails(pharmacyID)
	if err != nil {
		return nil, err
	}

	return &pharmacy.PharmacyData{
		SourceID:     details.PharmacyID,
		AddressLine1: details.Address1,
		AddressLine2: details.Address2,
		City:         details.City,
		Postal:       details.ZipCode,
		State:        details.State,
		Phone:        details.PrimaryPhone,
		Name:         details.StoreName,
		Source:       pharmacy.PharmacySourceSurescripts,
	}, nil
}

func convertMedicationIntoTreatment(medicationItem *dosespot.Medication) *common.Treatment {
	if medicationItem == nil {
		return nil
	}
	scheduleInt, err := strconv.Atoi(medicationItem.Schedule)
	dispenseValue, _ := strconv.ParseFloat(medicationItem.Dispense, 64)
	treatment := &common.Treatment{
		DrugDBIDs: map[string]string{
			LexiDrugSynID:     strconv.FormatInt(medicationItem.LexiDrugSynID, 10),
			LexiGenProductID:  strconv.FormatInt(medicationItem.LexiGenProductID, 10),
			LexiSynonymTypeID: strconv.FormatInt(medicationItem.LexiSynonymTypeID, 10),
			NDC:               medicationItem.NDC,
		},
		DrugName:                medicationItem.DrugName,
		IsControlledSubstance:   err == nil && scheduleInt > 0,
		DaysSupply:              medicationItem.DaysSupply,
		DispenseValue:           encoding.HighPrecisionFloat64(dispenseValue),
		DispenseUnitID:          encoding.DeprecatedNewObjectID(medicationItem.DispenseUnitID),
		DispenseUnitDescription: medicationItem.DispenseUnitDescription,
		PatientInstructions:     medicationItem.Instructions,
		SubstitutionsAllowed:    !medicationItem.NoSubstitutions,
		PharmacyNotes:           medicationItem.PharmacyNotes,
		DrugRoute:               medicationItem.Route,
		DosageStrength:          medicationItem.Strength,
		ERx: &common.ERxData{
			PrescriptionID:      encoding.DeprecatedNewObjectID(medicationItem.DoseSpotPrescriptionID),
			ErxPharmacyID:       medicationItem.PharmacyID,
			PrescriptionStatus:  medicationItem.PrescriptionStatus,
			ErxMedicationID:     encoding.DeprecatedNewObjectID(medicationItem.MedicationID),
			DoseSpotClinicianID: medicationItem.ClinicianID,
		},
	}

	treatment.NumberRefills, _ = encoding.NullInt64FromString(medicationItem.Refills)

	if medicationItem.DatePrescribed != nil {
		treatment.ERx.ErxSentDate = &medicationItem.DatePrescribed.DateTime
	}

	if medicationItem.LastDateFilled != nil {
		treatment.ERx.ErxLastDateFilled = &medicationItem.LastDateFilled.DateTime
	}
	return treatment

}

// ParseGenericName parses and returns the generic drug name from a medication select
// response. The .GenericName in that struct seems to almost always be in the format
// "name strength route form".. except for odd cases. (for examples search for
// "PruClair", "Pruet", "Tums", "Pepcid")
func ParseGenericName(m *dosespot.MedicationSelectResponse) (string, error) {
	trimFn := func(r rune) bool {
		switch r {
		case ' ':
			return true
		case ',':
			return true
		}
		return false
	}
	name := m.GenericProductName
	// The generic name is at the beginning of the string so find the lowest index in
	// the string for route, form, and strength and truncate the string to it.
	ix := strings.Index(name, m.DoseFormDescription)
	if i := strings.Index(name, m.RouteDescription); ix < 0 || (i >= 0 && i < ix) {
		ix = i
	}
	if i := strings.Index(name, m.StrengthDescription); ix < 0 || (i >= 0 && i < ix) {
		ix = i
	}
	// If none were found then something is terribly wrong
	if ix <= 0 {
		return "", fmt.Errorf("dosespot: no route, form, or strength found for '%s'", name)
	}
	return strings.TrimRightFunc(name[:ix], trimFn), nil
}
