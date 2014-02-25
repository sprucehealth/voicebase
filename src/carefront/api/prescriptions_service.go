package api

import (
	"carefront/common"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (d *DataService) AddRefillRequestStatusEvent(rxRefillRequestId int64, status string, statusDate time.Time) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update rx_refill_status_events set status = ? where status = ? and rx_refill_request_id = ?`, status_inactive, status_active, rxRefillRequestId)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into rx_refill_status_events (rx_refill_request_id, rx_refill_status, rx_refill_status_date, status) values (?,?,?,?)`, rxRefillRequestId, status, statusDate, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetPendingRefillRequestStatusEventsForClinic() ([]RefillRequestStatus, error) {
	rows, err := d.DB.Query(`select rx_refill_request_id, rx_refill_request.erx_request_queue_item_id, rx_refill_status, rx_refill_status_date   
								from rx_refill_status_events 
									inner join rx_refill_request on rx_refill_request_id = rx_refill_request.id
									where rx_refill_status_events.status = ? and rx_refill_status = ?`, status_active, RX_REFILL_STATUS_REQUESTED)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	refillRequestStatuses := make([]RefillRequestStatus, 0)
	for rows.Next() {
		var refillRequestStatus RefillRequestStatus
		err = rows.Scan(&refillRequestStatus.ErxRefillRequestId, &refillRequestStatus.RxRequestQueueItemId, &refillRequestStatus.Status, &refillRequestStatus.StatusTimeStamp)
		if err != nil {
			return nil, err
		}
		refillRequestStatuses = append(refillRequestStatuses, refillRequestStatus)
	}
	return refillRequestStatuses, nil
}

func (d *DataService) AddUnlinkedTreatmentFromPharmacy(unlinkedTreatment *common.Treatment) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	err = d.addUnlinkedTreatmentFromPharmacy(unlinkedTreatment, tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) CreateRefillRequest(refillRequest *common.RefillRequestItem) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err

	}

	err = d.addPharmacyDispensedTreatment(refillRequest.DispensedPrescription, refillRequest.RequestedPrescription, tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	columnsAndData := map[string]interface{}{
		"erx_request_queue_item_id":  refillRequest.RxRequestQueueItemId,
		"requested_drug_description": refillRequest.RequestedDrugDescription,
		"requested_refill_amount":    refillRequest.RequestedRefillAmount,
		"requested_dispense":         refillRequest.RequestedDispense,
		"patient_id":                 refillRequest.Patient.PatientId.Int64(),
		"request_date":               refillRequest.RequestDateStamp,
		"doctor_id":                  refillRequest.Doctor.DoctorId.Int64(),
		"dispensed_treatment_id":     refillRequest.DispensedPrescription.Id.Int64(),
	}

	// only have a link to the unlinked treatment if it so exists
	if refillRequest.RequestedPrescription.IsUnlinked {
		columnsAndData["unlinked_requested_treatment_id"] = refillRequest.RequestedPrescription.Id.Int64()
	} else {
		columnsAndData["requested_treatment_id"] = refillRequest.RequestedPrescription.Id.Int64()

	}

	if refillRequest.ReferenceNumber != "" {
		columnsAndData["reference_number"] = refillRequest.ReferenceNumber
	}

	if refillRequest.PharmacyRxReferenceNumber != "" {
		columnsAndData["pharmacy_rx_reference_number"] = refillRequest.PharmacyRxReferenceNumber
	}

	columns, dataForColumns := getKeysAndValuesFromMap(columnsAndData)

	lastId, err := tx.Exec(fmt.Sprintf(`insert into rx_refill_request (%s) values (%s)`,
		strings.Join(columns, ","), nReplacements(len(columns))), dataForColumns...)
	if err != nil {
		tx.Rollback()
		return err
	}

	refillRequest.Id, err = lastId.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (d *DataService) GetRefillRequestFromId(refillRequestId int64) (*common.RefillRequestItem, error) {
	var refillRequest common.RefillRequestItem
	var patientId, doctorId, pharmacyDispensedTreatmentId int64
	var unlinkedRequestedTreatmentId, requestedTreatmentId, approvedRefillAmount sql.NullInt64
	var refillStatus, notes, denyReason sql.NullString
	// get the refill request
	err := d.DB.QueryRow(`select rx_refill_request.id, rx_refill_request.erx_request_queue_item_id, requested_drug_description, requested_refill_amount,
		approved_refill_amount, requested_dispense, patient_id, request_date, doctor_id, requested_treatment_id, 
		dispensed_treatment_id, unlinked_requested_treatment_id, rx_refill_status_events.rx_refill_status, rx_refill_status_events.notes, deny_refill_reason.reason from rx_refill_request
			left outer join rx_refill_status_events on rx_refill_request.id =  rx_refill_request_id
			left outer join deny_refill_reason on reason_id = rx_refill_status_events.reason_id
				where rx_refill_request.id = ? and rx_refill_status_events.status = ?`, refillRequestId, status_active).Scan(&refillRequest.Id,
		&refillRequest.RxRequestQueueItemId, &refillRequest.RequestedDrugDescription, &refillRequest.RequestedRefillAmount, &approvedRefillAmount,
		&refillRequest.RequestedDispense, &patientId, &refillRequest.RequestDateStamp, &doctorId, &requestedTreatmentId,
		&pharmacyDispensedTreatmentId, &unlinkedRequestedTreatmentId, &refillStatus, &notes, &denyReason)

	refillRequest.Status = refillStatus.String
	refillRequest.ApprovedRefillAmount = approvedRefillAmount.Int64
	refillRequest.Comments = notes.String
	refillRequest.DenialReason = denyReason.String

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// get the patient associated with the refill request
	refillRequest.Patient, err = d.GetPatientFromId(patientId)
	if err != nil {
		return nil, err
	}

	// get the doctor associated with the refill request
	refillRequest.Doctor, err = d.GetDoctorFromId(doctorId)
	if err != nil {
		return nil, err
	}

	// get the pharmacy dispensed treatment
	refillRequest.DispensedPrescription, err = d.getTreatmentForRefillRequest(table_name_pharmacy_dispensed_treatment, pharmacyDispensedTreatmentId)
	if err != nil {
		return nil, err
	}

	if requestedTreatmentId.Valid {
		// get the requested treatment
		rows, err := d.DB.Query(`select treatment.id, treatment.erx_id, treatment.treatment_plan_id, treatment.drug_internal_name, treatment.dosage_strength, treatment.type,
			treatment.dispense_value, treatment.dispense_unit_id, ltext, treatment.refills, treatment.substitutions_allowed, 
			treatment.days_supply, treatment.pharmacy_id, treatment.pharmacy_notes, treatment.patient_instructions, treatment.creation_date, treatment.erx_sent_date,
			treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_visit.patient_id, treatment_plan.patient_visit_id from treatment 
				inner join dispense_unit on treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				inner join treatment_plan on treatment_plan.id = treatment.treatment_plan_id
				inner join patient_visit on treatment_plan.patient_visit_id = patient_visit.id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where treatment.id = ?`, requestedTreatmentId.Int64)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		if rows.Next() {
			refillRequest.RequestedPrescription, err = d.getTreatmentFromCurrentRow(rows)
			if err != nil {
				return nil, err
			}
		}
	}

	// get the unlinked requested treatment
	if unlinkedRequestedTreatmentId.Valid {
		refillRequest.RequestedPrescription, err = d.getTreatmentForRefillRequest(table_name_unlinked_requested_treatment, unlinkedRequestedTreatmentId.Int64)
		if err != nil {
			return nil, err
		}
		refillRequest.RequestedPrescription.IsUnlinked = true
	}

	// get the pharmacy
	var pharmacyLocalId int64
	if refillRequest.DispensedPrescription.PharmacyLocalId != nil {
		pharmacyLocalId = refillRequest.DispensedPrescription.PharmacyLocalId.Int64()
	} else if refillRequest.RequestedPrescription != nil && refillRequest.RequestedPrescription.PharmacyLocalId != nil {
		pharmacyLocalId = refillRequest.RequestedPrescription.PharmacyLocalId.Int64()
	}

	refillRequest.Pharmacy, err = d.GetPharmacyFromId(pharmacyLocalId)
	if err != nil {
		return nil, err
	}

	return &refillRequest, nil
}

func (d *DataService) getTreatmentForRefillRequest(tableName string, treatmentId int64) (*common.Treatment, error) {
	var treatment common.Treatment
	var erxId, pharmacyLocalId int64
	var treatmentType string
	var drugName, drugForm, drugRoute sql.NullString

	err := d.DB.QueryRow(fmt.Sprintf(`select erx_id, drug_internal_name, 
							dosage_strength, type, dispense_value, 
							dispense_unit, refills, substitutions_allowed, 
							pharmacy_id, days_supply, pharmacy_notes, 
							patient_instructions, erx_sent_date,
							erx_last_filled_date,  status, drug_name.name, drug_route.name, drug_form.name from %s 
								left outer join drug_name on drug_name_id = drug_name.id
								left outer join drug_route on drug_route_id = drug_route.id
								left outer join drug_form on drug_form_id = drug_form.id
									where %s.id = ?`, tableName, tableName), treatmentId).Scan(&erxId, &treatment.DrugInternalName,
		&treatment.DosageStrength, &treatmentType, &treatment.DispenseValue,
		&treatment.DispenseUnitDescription, &treatment.NumberRefills,
		&treatment.SubstitutionsAllowed, &pharmacyLocalId,
		&treatment.DaysSupply, &treatment.PharmacyNotes,
		&treatment.PatientInstructions, &treatment.ErxSentDate,
		&treatment.ErxLastDateFilled, &treatment.Status,
		&drugName, &drugForm, &drugRoute)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	treatment.PrescriptionId = common.NewObjectId(erxId)
	treatment.DrugName = drugName.String
	treatment.DrugForm = drugForm.String
	treatment.DrugRoute = drugRoute.String
	treatment.OTC = treatmentType == treatment_otc
	treatment.PharmacyLocalId = common.NewObjectId(pharmacyLocalId)

	return &treatment, nil
}

// this method is used to add treatments that come in from dosespot (either pharmacy dispensed medication or treatments that don't exist but
// are the basis of a refill request)
func (d *DataService) addUnlinkedTreatmentFromPharmacy(treatment *common.Treatment, tx *sql.Tx) error {
	substitutionsAllowedBit := 0
	if treatment.SubstitutionsAllowed {
		substitutionsAllowedBit = 1
	}

	treatmentType := treatment_rx
	if treatment.OTC {
		treatmentType = treatment_otc
	}

	columnsAndData := map[string]interface{}{
		"drug_internal_name":    treatment.DrugInternalName,
		"dosage_strength":       treatment.DosageStrength,
		"type":                  treatmentType,
		"dispense_value":        treatment.DispenseValue,
		"dispense_unit":         treatment.DispenseUnitDescription,
		"refills":               treatment.NumberRefills,
		"substitutions_allowed": substitutionsAllowedBit,
		"days_supply":           treatment.DaysSupply,
		"patient_instructions":  treatment.PatientInstructions,
		"pharmacy_notes":        treatment.PharmacyNotes,
		"status":                treatment.Status,
		"erx_id":                treatment.PrescriptionId.Int64(),
		"erx_sent_date":         treatment.ErxSentDate,
		"erx_last_filled_date":  treatment.ErxLastDateFilled,
		"pharmacy_id":           treatment.PharmacyLocalId,
	}

	if err := d.includeDrugNameComponentIfNonZero(treatment.DrugName, drug_name_table, "drug_name_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := d.includeDrugNameComponentIfNonZero(treatment.DrugForm, drug_form_table, "drug_form_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}
	if err := d.includeDrugNameComponentIfNonZero(treatment.DrugRoute, drug_route_table, "drug_route_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}

	columns, dataForColumns := getKeysAndValuesFromMap(columnsAndData)
	res, err := tx.Exec(fmt.Sprintf(`insert into unlinked_requested_treatment (%s) values (%s)`, strings.Join(columns, ","), nReplacements(len(dataForColumns))), dataForColumns...)
	if err != nil {
		tx.Rollback()
		return err
	}

	treatmentId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	treatment.Id = common.NewObjectId(treatmentId)
	// add drug db ids to the table
	for drugDbTag, drugDbId := range treatment.DrugDBIds {
		_, err := tx.Exec(`insert into unlinked_requested_treatment_drug_db_id (drug_db_id_tag, drug_db_id, unlinked_requested_treatment_id) values (?, ?, ?)`, drugDbTag, drugDbId, treatment.Id.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return nil
}

func (d *DataService) addPharmacyDispensedTreatment(dispensedTreatment, requestedTreatment *common.Treatment, tx *sql.Tx) error {
	substitutionsAllowedBit := 0
	if dispensedTreatment.SubstitutionsAllowed {
		substitutionsAllowedBit = 1
	}

	treatmentType := treatment_rx
	if dispensedTreatment.OTC {
		treatmentType = treatment_otc
	}

	columnsAndData := map[string]interface{}{
		"drug_internal_name":    dispensedTreatment.DrugInternalName,
		"dosage_strength":       dispensedTreatment.DosageStrength,
		"type":                  treatmentType,
		"dispense_value":        dispensedTreatment.DispenseValue,
		"dispense_unit":         dispensedTreatment.DispenseUnitDescription,
		"refills":               dispensedTreatment.NumberRefills,
		"substitutions_allowed": substitutionsAllowedBit,
		"days_supply":           dispensedTreatment.DaysSupply,
		"patient_instructions":  dispensedTreatment.PatientInstructions,
		"pharmacy_notes":        dispensedTreatment.PharmacyNotes,
		"status":                dispensedTreatment.Status,
		"erx_id":                dispensedTreatment.PrescriptionId.Int64(),
		"erx_sent_date":         dispensedTreatment.ErxSentDate,
		"erx_last_filled_date":  dispensedTreatment.ErxLastDateFilled,
		"pharmacy_id":           dispensedTreatment.PharmacyLocalId,
	}

	if requestedTreatment.IsUnlinked {
		columnsAndData["unlinked_requested_treatment_id"] = requestedTreatment.Id.Int64()
	} else {
		columnsAndData["treatment_id"] = requestedTreatment.Id.Int64()
	}

	if err := d.includeDrugNameComponentIfNonZero(dispensedTreatment.DrugName, drug_name_table, "drug_name_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := d.includeDrugNameComponentIfNonZero(dispensedTreatment.DrugForm, drug_form_table, "drug_form_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}
	if err := d.includeDrugNameComponentIfNonZero(dispensedTreatment.DrugRoute, drug_route_table, "drug_route_id", columnsAndData, tx); err != nil {
		tx.Rollback()
		return err
	}

	columns, dataForColumns := getKeysAndValuesFromMap(columnsAndData)
	res, err := tx.Exec(fmt.Sprintf(`insert into pharmacy_dispensed_treatment (%s) values (%s)`, strings.Join(columns, ","), nReplacements(len(dataForColumns))), dataForColumns...)
	if err != nil {
		tx.Rollback()
		return err
	}

	treatmentId, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	dispensedTreatment.Id = common.NewObjectId(treatmentId)
	// add drug db ids to the table
	for drugDbTag, drugDbId := range dispensedTreatment.DrugDBIds {
		_, err := tx.Exec(`insert into pharmacy_dispensed_treatment_drug_db_id (drug_db_id_tag, drug_db_id, pharmacy_dispensed_treatment_id) values (?, ?, ?)`, drugDbTag, drugDbId, dispensedTreatment.Id.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return nil
}

func (d *DataService) GetRefillRequestDenialReasons() ([]*RefillRequestDenialReason, error) {
	rows, err := d.DB.Query(`select id, reason_code, reason from deny_refill_reason`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	denialReasons := make([]*RefillRequestDenialReason, 0)
	for rows.Next() {
		var denialReason RefillRequestDenialReason
		err = rows.Scan(&denialReason.Id, &denialReason.DenialCode, &denialReason.DenialReason)
		if err != nil {
			return nil, err
		}
		denialReasons = append(denialReasons, &denialReason)
	}

	return denialReasons, nil
}

func (d *DataService) MarkRefillRequestAsApproved(approvedRefillCount, rxRefillRequestId, prescriptionId int64, comments string) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update rx_refill_request set erx_id = ?, approved_refill_amount = ? where id = ?`, prescriptionId, approvedRefillCount, rxRefillRequestId)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update rx_refill_status_events set status = ? where rx_refill_request_id = ? and status = ?`, status_inactive, rxRefillRequestId, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into rx_refill_status_events (rx_refill_request_id, rx_refill_status, status, notes, rx_refill_status_date) values (?,?,?,?, now())`, rxRefillRequestId, RX_REFILL_STATUS_APPROVED, status_active, comments)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) MarkRefillRequestAsDenied(denialReasonId, rxRefillRequestId, prescriptionId int64, comments string) error {
	tx, err := d.DB.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update rx_refill_request set erx_id = ? where id = ?`, prescriptionId, rxRefillRequestId)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update rx_refill_status_events set status = ? where rx_refill_request_id = ? and status = ?`, status_inactive, rxRefillRequestId, status_active)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into rx_refill_status_events (rx_refill_request_id, rx_refill_status, reason_id,status,notes, rx_refill_status_date) values (?,?,?,?,?, now())`, rxRefillRequestId, RX_REFILL_STATUS_DENIED, denialReasonId, status_active, comments)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}