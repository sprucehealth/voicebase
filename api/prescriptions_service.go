package api

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/pharmacy"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/go-sql-driver/mysql"
)

func (d *DataService) AddRefillRequestStatusEvent(refillRequestStatus common.StatusEvent) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update rx_refill_status_events set status = ? where status = ? and rx_refill_request_id = ?`, STATUS_INACTIVE, STATUS_ACTIVE, refillRequestStatus.ItemID)
	if err != nil {
		tx.Rollback()
		return err
	}

	columnsAndData := map[string]interface{}{
		"rx_refill_request_id": refillRequestStatus.ItemID,
		"rx_refill_status":     refillRequestStatus.Status,
		"status":               STATUS_ACTIVE,
		"event_details":        refillRequestStatus.StatusDetails,
	}

	if !refillRequestStatus.ReportedTimestamp.IsZero() {
		columnsAndData["reported_timestamp"] = refillRequestStatus.ReportedTimestamp
	}

	keys, values := getKeysAndValuesFromMap(columnsAndData)
	_, err = tx.Exec(fmt.Sprintf(`insert into rx_refill_status_events (%s) values (%s)`, strings.Join(keys, ","), dbutil.MySQLArgs(len(values))), values...)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetPendingRefillRequestStatusEventsForClinic() ([]common.StatusEvent, error) {
	rows, err := d.db.Query(`
		SELECT rx_refill_request_id, rx_refill_status, rx_refill_status_date, 
		event_details, erx_id  
		FROM rx_refill_status_events 
		INNER JOIN rx_refill_request on rx_refill_request_id = rx_refill_request.id
		WHERE rx_refill_status_events.status = ? AND rx_refill_status = ?`, STATUS_ACTIVE, RX_REFILL_STATUS_REQUESTED)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return getRefillStatusEventsFromRows(rows)
}

func (d *DataService) GetApprovedOrDeniedRefillRequestsForPatient(patientID int64) ([]common.StatusEvent, error) {
	rows, err := d.db.Query(`
		SELECT rx_refill_request_id, rx_refill_status, rx_refill_status_date, event_details, erx_id    
		FROM rx_refill_status_events 
		INNER JOIN rx_refill_request on rx_refill_request_id = rx_refill_request.id
		WHERE rx_refill_status_events.rx_refill_status in (?, ?) and rx_refill_request.patient_id = ?
		AND status = ?
		ORDER BY rx_refill_status_date DESC, rx_refill_status_events.id DESC`,
		RX_REFILL_STATUS_APPROVED, RX_REFILL_STATUS_DENIED, patientID, STATUS_ACTIVE)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return getRefillStatusEventsFromRows(rows)
}

func (d *DataService) GetRefillStatusEventsForRefillRequest(refillRequestID int64) ([]common.StatusEvent, error) {
	rows, err := d.db.Query(`
		SELECT rx_refill_request_id,rx_refill_status, rx_refill_status_date, event_details, erx_id    
		FROM rx_refill_status_events 
		INNER JOIN rx_refill_request on rx_refill_request_id = rx_refill_request.id
		WHERE rx_refill_status_events.rx_refill_request_id = ?
		ORDER BY rx_refill_status_date DESC, rx_refill_status_events.id DESC`, refillRequestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return getRefillStatusEventsFromRows(rows)
}

func getRefillStatusEventsFromRows(rows *sql.Rows) ([]common.StatusEvent, error) {
	refillRequestStatuses := make([]common.StatusEvent, 0)
	for rows.Next() {
		var refillRequestStatus common.StatusEvent
		var prescriptionID sql.NullInt64
		var statusDetails sql.NullString
		err := rows.Scan(&refillRequestStatus.ItemID, &refillRequestStatus.Status,
			&refillRequestStatus.StatusTimestamp, &statusDetails, &prescriptionID)
		if err != nil {
			return nil, err
		}
		refillRequestStatus.StatusDetails = statusDetails.String
		refillRequestStatus.PrescriptionID = prescriptionID.Int64
		refillRequestStatuses = append(refillRequestStatuses, refillRequestStatus)
	}
	return refillRequestStatuses, rows.Err()
}

func (d *DataService) LinkRequestedPrescriptionToOriginalTreatment(requestedTreatment *common.Treatment, patient *common.Patient) error {
	// lookup drug based on the drugIds
	if len(requestedTreatment.DrugDBIDs) == 0 {
		// nothing to compare against to link to originating drug
		return nil
	}

	// lookup drugs prescribed to the patient within a day of the date the requestedPrescription was prescribed
	// we know that it was prescribed based on whether or not it was succesfully sent to the pharmacy
	halfDayBefore := requestedTreatment.ERx.ErxSentDate.Add(-12 * time.Hour)
	halfDayAfter := requestedTreatment.ERx.ErxSentDate.Add(12 * time.Hour)

	treatmentIds := make([]int64, 0)
	rows, err := d.db.Query(`select treatment_id from erx_status_events 
								inner join treatment on treatment_id = treatment.id 
								inner join treatment_plan on treatment_plan_id = treatment.treatment_plan_id
								where erx_status = ? and erx_status_events.creation_date >= ? 
								and erx_status_events.creation_date <= ? and treatment_plan.patient_id = ? `, ERX_STATUS_SENT, halfDayBefore, halfDayAfter, patient.PatientID.Int64())
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var treatmentID int64
		err = rows.Scan(&treatmentID)
		if err != nil {
			return err
		}
		treatmentIds = append(treatmentIds, treatmentID)
	}
	if rows.Err() != nil {
		return rows.Err()
	}

	for _, treatmentID := range treatmentIds {
		// for each of the treatments gathered for the patiend, compare the drug ids against the requested prescription to identify if they
		// match to find the originating prescritpion
		treatment, err := d.GetTreatmentFromID(treatmentID)
		if err != nil {
			return err
		}

		if requestedTreatment.DrugDBIDs[erx.LexiGenProductID] == treatment.DrugDBIDs[erx.LexiGenProductID] &&
			requestedTreatment.DrugDBIDs[erx.LexiDrugSynID] == treatment.DrugDBIDs[erx.LexiDrugSynID] &&
			requestedTreatment.DrugDBIDs[erx.LexiSynonymTypeID] == treatment.DrugDBIDs[erx.LexiSynonymTypeID] {
			// linkage found
			requestedTreatment.OriginatingTreatmentID = treatmentID
			return nil
		}
	}

	return nil
}

func (d *DataService) CreateRefillRequest(refillRequest *common.RefillRequestItem) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err

	}

	if err := d.addTreatment(refillRequestTreatmentType, refillRequest.RequestedPrescription, nil, tx); err != nil {
		tx.Rollback()
		return err
	}

	params := map[string]interface{}{
		"requested_treatment": refillRequest.RequestedPrescription,
	}

	if err := d.addTreatment(pharmacyDispensedTreatmentType, refillRequest.DispensedPrescription, params, tx); err != nil {
		tx.Rollback()
		return err
	}

	columnsAndData := map[string]interface{}{
		"erx_request_queue_item_id": refillRequest.RxRequestQueueItemID,
		"patient_id":                refillRequest.Patient.PatientID.Int64(),
		"request_date":              refillRequest.RequestDateStamp,
		"doctor_id":                 refillRequest.Doctor.DoctorID.Int64(),
		"dispensed_treatment_id":    refillRequest.DispensedPrescription.ID.Int64(),
		"requested_treatment_id":    refillRequest.RequestedPrescription.ID.Int64(),
	}

	if refillRequest.ReferenceNumber != "" {
		columnsAndData["reference_number"] = refillRequest.ReferenceNumber
	}

	if refillRequest.PharmacyRxReferenceNumber != "" {
		columnsAndData["pharmacy_rx_reference_number"] = refillRequest.PharmacyRxReferenceNumber
	}

	columns, dataForColumns := getKeysAndValuesFromMap(columnsAndData)

	lastID, err := tx.Exec(fmt.Sprintf(`insert into rx_refill_request (%s) values (%s)`,
		strings.Join(columns, ","), dbutil.MySQLArgs(len(columns))), dataForColumns...)
	if err != nil {
		tx.Rollback()
		return err
	}

	refillRequest.ID, err = lastID.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetRefillRequestFromID(refillRequestID int64) (*common.RefillRequestItem, error) {
	// get the refill request
	rows, err := d.db.Query(`select rx_refill_request.id, rx_refill_request.erx_request_queue_item_id,rx_refill_request.reference_number, rx_refill_request.erx_id,
		approved_refill_amount, patient_id, request_date, doctor_id, requested_treatment_id, 
		dispensed_treatment_id, comments, deny_refill_reason.reason from rx_refill_request
				left outer join deny_refill_reason on deny_refill_reason.id = denial_reason_id
				where rx_refill_request.id = ?`, refillRequestID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	refillRequests, err := d.getRefillRequestsFromRow(rows)
	if err != nil {
		return nil, err
	}

	switch l := len(refillRequests); {
	case l == 0:
		return nil, NoRowsError
	case l > 1:
		return nil, fmt.Errorf("Expected just one refill request instead got %d", len(refillRequests))
	}

	return refillRequests[0], nil
}

func (d *DataService) FilterOutRefillRequestsThatExist(queueItemIDs []int64) ([]int64, error) {

	if len(queueItemIDs) == 0 {
		return nil, nil
	}

	// get a list of refill requests (identified by their queue item ids)
	// that exist in the database
	rows, err := d.db.Query(`
		SELECT distinct erx_request_queue_item_id
		FROM rx_refill_request
		WHERE erx_request_queue_item_id in (`+dbutil.MySQLArgs(len(queueItemIDs))+`)`,
		dbutil.AppendInt64sToInterfaceSlice(nil, queueItemIDs)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var existingQueueItemIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		existingQueueItemIDs = append(existingQueueItemIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// create a set of the existing queueItemIDs for quick access
	existingQueueItemIDSet := make(map[int64]bool)
	for _, queueItemID := range existingQueueItemIDs {
		existingQueueItemIDSet[queueItemID] = true
	}

	// filter out the existing queueItemIDs
	nonExistingQueueItemIDs := make([]int64, 0, len(queueItemIDs))
	for _, queueItemID := range queueItemIDs {
		if !existingQueueItemIDSet[queueItemID] {
			nonExistingQueueItemIDs = append(nonExistingQueueItemIDs, queueItemID)
		}
	}

	return nonExistingQueueItemIDs, nil
}

func (d *DataService) GetRefillRequestFromPrescriptionID(prescriptionID int64) (*common.RefillRequestItem, error) {

	// get the refill request
	rows, err := d.db.Query(`select rx_refill_request.id, rx_refill_request.erx_request_queue_item_id,rx_refill_request.reference_number, rx_refill_request.erx_id,
		approved_refill_amount, patient_id, request_date, doctor_id, requested_treatment_id, 
		dispensed_treatment_id, comments, deny_refill_reason.reason from rx_refill_request
				left outer join deny_refill_reason on deny_refill_reason.id = denial_reason_id
				where rx_refill_request.erx_id = ?`, prescriptionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	refillRequests, err := d.getRefillRequestsFromRow(rows)
	if err != nil {
		return nil, err
	}

	switch l := len(refillRequests); {
	case l == 0:
		return nil, NoRowsError
	case l > 1:
		return nil, fmt.Errorf("Expected just one refill request instead got %d", len(refillRequests))
	}

	return refillRequests[0], nil
}

func (d *DataService) GetRefillRequestsForPatient(patientID int64) ([]*common.RefillRequestItem, error) {
	// get the refill request
	rows, err := d.db.Query(`select rx_refill_request.id, rx_refill_request.erx_request_queue_item_id,rx_refill_request.reference_number, rx_refill_request.erx_id,
		approved_refill_amount, patient_id, request_date, doctor_id, requested_treatment_id, 
		dispensed_treatment_id, comments, deny_refill_reason.reason from rx_refill_request
				left outer join deny_refill_reason on deny_refill_reason.id = denial_reason_id
				where patient_id = ? order by rx_refill_request.request_date desc`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	refillRequests, err := d.getRefillRequestsFromRow(rows)

	return refillRequests, err
}

func (d *DataService) getRefillRequestsFromRow(rows *sql.Rows) ([]*common.RefillRequestItem, error) {

	refillRequests := make([]*common.RefillRequestItem, 0)

	for rows.Next() {
		var refillRequest common.RefillRequestItem
		var patientID, doctorID, pharmacyDispensedTreatmentId int64
		var requestedTreatmentId, approvedRefillAmount, prescriptionID sql.NullInt64
		var denyReason, comments sql.NullString

		err := rows.Scan(&refillRequest.ID,
			&refillRequest.RxRequestQueueItemID, &refillRequest.ReferenceNumber, &prescriptionID, &approvedRefillAmount,
			&patientID, &refillRequest.RequestDateStamp, &doctorID, &requestedTreatmentId,
			&pharmacyDispensedTreatmentId, &comments, &denyReason)
		if err != nil {
			return nil, err
		}

		refillRequest.PrescriptionID = prescriptionID.Int64
		refillRequest.ApprovedRefillAmount = approvedRefillAmount.Int64
		refillRequest.DenialReason = denyReason.String
		refillRequest.Comments = comments.String

		if err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}

		// get the patient associated with the refill request
		refillRequest.Patient, err = d.GetPatientFromID(patientID)
		if err != nil {
			return nil, err
		}

		// get the doctor associated with the refill request
		refillRequest.Doctor, err = d.GetDoctorFromID(doctorID)
		if err != nil {
			return nil, err
		}

		// get the pharmacy dispensed treatment
		refillRequest.DispensedPrescription, err = d.getTreatmentForRefillRequest(pharmacyDispensedTreatmentTable, pharmacyDispensedTreatmentId)
		if err != nil {
			return nil, err
		}

		// get the unlinked requested treatment
		refillRequest.RequestedPrescription, err = d.getTreatmentForRefillRequest(requestedTreatmentTable, requestedTreatmentId.Int64)
		if err != nil {
			return nil, err
		}

		var originatingTreatmentId sql.NullInt64
		var originatingTreatmentPlanId encoding.ObjectID
		err = d.db.QueryRow(`select originating_treatment_id, treatment_plan_id from requested_treatment 
							inner join treatment on originating_treatment_id = treatment.id
								where requested_treatment.id = ?`, refillRequest.RequestedPrescription.ID.Int64()).Scan(&originatingTreatmentId, &originatingTreatmentPlanId)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}

		if originatingTreatmentId.Valid {
			refillRequest.RequestedPrescription.OriginatingTreatmentID = originatingTreatmentId.Int64
			refillRequest.TreatmentPlanID = originatingTreatmentPlanId
		}

		refillRequest.RxHistory, err = d.GetRefillStatusEventsForRefillRequest(refillRequest.ID)
		if err != nil {
			return nil, err
		}

		refillRequests = append(refillRequests, &refillRequest)
	}

	return refillRequests, rows.Err()
}

func (d *DataService) getTreatmentForRefillRequest(tableName string, treatmentID int64) (*common.Treatment, error) {
	var treatment common.Treatment
	treatment.ERx = &common.ERxData{}
	var erxID, pharmacyLocalId encoding.ObjectID
	var daysSupply, refills encoding.NullInt64
	var doctorID sql.NullInt64
	var treatmentType string
	var drugName, drugForm, drugRoute sql.NullString
	var isControlledSubstance sql.NullBool

	err := d.db.QueryRow(fmt.Sprintf(`select erx_id, drug_internal_name, 
							dosage_strength, type, dispense_value, 
							dispense_unit, refills, substitutions_allowed, 
							pharmacy_id, days_supply, pharmacy_notes, 
							patient_instructions, erx_sent_date,
							erx_last_filled_date,  status, drug_name.name, drug_route.name, drug_form.name, doctor_id, is_controlled_substance from %s
								left outer join drug_name on drug_name_id = drug_name.id
								left outer join drug_route on drug_route_id = drug_route.id
								left outer join drug_form on drug_form_id = drug_form.id
									where %s.id = ?`, tableName, tableName), treatmentID).Scan(&erxID, &treatment.DrugInternalName,
		&treatment.DosageStrength, &treatmentType, &treatment.DispenseValue,
		&treatment.DispenseUnitDescription, &refills,
		&treatment.SubstitutionsAllowed, &pharmacyLocalId,
		&daysSupply, &treatment.PharmacyNotes,
		&treatment.PatientInstructions, &treatment.ERx.ErxSentDate,
		&treatment.ERx.ErxLastDateFilled, &treatment.Status,
		&drugName, &drugForm, &drugRoute, &doctorID, &isControlledSubstance)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	treatment.ID = encoding.NewObjectID(treatmentID)
	treatment.ERx.PrescriptionID = erxID
	treatment.DrugName = drugName.String
	treatment.DrugForm = drugForm.String
	treatment.DrugRoute = drugRoute.String
	treatment.OTC = treatmentType == treatmentOTC
	treatment.DaysSupply = daysSupply
	treatment.NumberRefills = refills
	treatment.ERx.PharmacyLocalID = pharmacyLocalId
	treatment.ERx.Pharmacy, err = d.GetPharmacyFromID(pharmacyLocalId.Int64())
	treatment.IsControlledSubstance = isControlledSubstance.Bool

	if err != nil {
		return nil, err
	}

	if doctorID.Valid {
		treatment.Doctor, err = d.GetDoctorFromID(doctorID.Int64)
		if err != nil {
			return nil, err
		}
	}

	return &treatment, nil
}

func (d *DataService) GetRefillRequestDenialReasons() ([]*RefillRequestDenialReason, error) {
	rows, err := d.db.Query(`select id, reason_code, reason from deny_refill_reason`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	denialReasons := make([]*RefillRequestDenialReason, 0)
	for rows.Next() {
		var denialReason RefillRequestDenialReason
		err = rows.Scan(&denialReason.ID, &denialReason.DenialCode, &denialReason.DenialReason)
		if err != nil {
			return nil, err
		}
		denialReasons = append(denialReasons, &denialReason)
	}

	return denialReasons, rows.Err()
}

func (d *DataService) MarkRefillRequestAsApproved(prescriptionID, approvedRefillCount, rxRefillRequestID int64, comments string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update rx_refill_request set erx_id = ?, approved_refill_amount = ?, comments = ? where id = ?`, prescriptionID, approvedRefillCount, comments, rxRefillRequestID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update rx_refill_status_events set status = ? where rx_refill_request_id = ? and status = ?`, STATUS_INACTIVE, rxRefillRequestID, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into rx_refill_status_events (rx_refill_request_id, rx_refill_status, status) values (?,?,?)`, rxRefillRequestID, RX_REFILL_STATUS_APPROVED, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) MarkRefillRequestAsDenied(prescriptionID, denialReasonID, rxRefillRequestID int64, comments string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update rx_refill_request set erx_id = ?, comments = ?, denial_reason_id = ? where id = ?`, prescriptionID, comments, denialReasonID, rxRefillRequestID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`update rx_refill_status_events set status = ? where rx_refill_request_id = ? and status = ?`, STATUS_INACTIVE, rxRefillRequestID, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into rx_refill_status_events (rx_refill_request_id, rx_refill_status, status) values (?,?,?)`, rxRefillRequestID, RX_REFILL_STATUS_DENIED, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) UpdateUnlinkedDNTFTreatmentWithPharmacyAndErxID(treatment *common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorID int64) error {
	if treatment.ERx.PrescriptionID.Int64() != 0 {
		_, err := d.db.Exec(`update unlinked_dntf_treatment set erx_id = ?, pharmacy_id = ?, erx_sent_date=now() where id = ?`, treatment.ERx.PrescriptionID.Int64(), pharmacySentTo.LocalID, treatment.ID.Int64())
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DataService) AddUnlinkedTreatmentInEventOfDNTF(treatment *common.Treatment, refillRequestID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := d.addTreatment(unlinkedDNTFTreatmentType, treatment, nil, tx); err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into unlinked_dntf_treatment_status_events (unlinked_dntf_treatment_id, erx_status, status) values (?,?,?)`, treatment.ID.Int64(), ERX_STATUS_NEW_RX_FROM_DNTF, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into dntf_mapping (unlinked_dntf_treatment_id, rx_refill_request_id) values (?,?)`, treatment.ID.Int64(), refillRequestID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetUnlinkedDNTFTreatment(treatmentID int64) (*common.Treatment, error) {
	rows, err := d.db.Query(`select unlinked_dntf_treatment.id, unlinked_dntf_treatment.erx_id, unlinked_dntf_treatment.drug_internal_name, unlinked_dntf_treatment.dosage_strength, unlinked_dntf_treatment.type,
			unlinked_dntf_treatment.dispense_value, unlinked_dntf_treatment.dispense_unit_id, ltext, unlinked_dntf_treatment.refills, unlinked_dntf_treatment.substitutions_allowed, 
			unlinked_dntf_treatment.days_supply, unlinked_dntf_treatment.pharmacy_id, unlinked_dntf_treatment.pharmacy_notes, unlinked_dntf_treatment.patient_instructions, unlinked_dntf_treatment.creation_date, unlinked_dntf_treatment.erx_sent_date,
			unlinked_dntf_treatment.erx_last_filled_date, unlinked_dntf_treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_id, unlinked_dntf_treatment.doctor_id, is_controlled_substance from unlinked_dntf_treatment 
				inner join dispense_unit on unlinked_dntf_treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where unlinked_dntf_treatment.id = ? and localized_text.language_id = ?`, treatmentID, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	treatments, err := d.getUnlinkedDNTFTreatmentsFromRow(rows)
	if err != nil {
		return nil, err
	}

	switch l := len(treatments); {
	case l == 1:
		return treatments[0], err
	case l == 0:
		return nil, NoRowsError
	}

	return nil, fmt.Errorf("Expected just one unlinked dntf treatment but got back %d", len(treatments))
}

func (d *DataService) GetUnlinkedDNTFTreatmentFromPrescriptionID(prescriptionID int64) (*common.Treatment, error) {
	rows, err := d.db.Query(`select unlinked_dntf_treatment.id, unlinked_dntf_treatment.erx_id, unlinked_dntf_treatment.drug_internal_name, unlinked_dntf_treatment.dosage_strength, unlinked_dntf_treatment.type,
			unlinked_dntf_treatment.dispense_value, unlinked_dntf_treatment.dispense_unit_id, ltext, unlinked_dntf_treatment.refills, unlinked_dntf_treatment.substitutions_allowed, 
			unlinked_dntf_treatment.days_supply, unlinked_dntf_treatment.pharmacy_id, unlinked_dntf_treatment.pharmacy_notes, unlinked_dntf_treatment.patient_instructions, unlinked_dntf_treatment.creation_date, unlinked_dntf_treatment.erx_sent_date,
			unlinked_dntf_treatment.erx_last_filled_date, unlinked_dntf_treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_id, unlinked_dntf_treatment.doctor_id, is_controlled_substance from unlinked_dntf_treatment 
				inner join dispense_unit on unlinked_dntf_treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where unlinked_dntf_treatment.erx_id = ? and localized_text.language_id = ?`, prescriptionID, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	treatments, err := d.getUnlinkedDNTFTreatmentsFromRow(rows)
	if err != nil {
		return nil, err
	}

	switch l := len(treatments); {
	case l == 0:
		return nil, NoRowsError
	case l > 1:
		return nil, fmt.Errorf("Expected just one unlinked dntf treatment but got back %d", len(treatments))
	}

	return treatments[0], err
}

func (d *DataService) GetUnlinkedDNTFTreatmentsForPatient(patientID int64) ([]*common.Treatment, error) {
	rows, err := d.db.Query(`select unlinked_dntf_treatment.id, unlinked_dntf_treatment.erx_id, unlinked_dntf_treatment.drug_internal_name, unlinked_dntf_treatment.dosage_strength, unlinked_dntf_treatment.type,
			unlinked_dntf_treatment.dispense_value, unlinked_dntf_treatment.dispense_unit_id, ltext, unlinked_dntf_treatment.refills, unlinked_dntf_treatment.substitutions_allowed, 
			unlinked_dntf_treatment.days_supply, unlinked_dntf_treatment.pharmacy_id, unlinked_dntf_treatment.pharmacy_notes, unlinked_dntf_treatment.patient_instructions, unlinked_dntf_treatment.creation_date, unlinked_dntf_treatment.erx_sent_date,
			unlinked_dntf_treatment.erx_last_filled_date, unlinked_dntf_treatment.status, drug_name.name, drug_route.name, drug_form.name,
			patient_id, unlinked_dntf_treatment.doctor_id, is_controlled_substance from unlinked_dntf_treatment 
				inner join dispense_unit on unlinked_dntf_treatment.dispense_unit_id = dispense_unit.id
				inner join localized_text on localized_text.app_text_id = dispense_unit.dispense_unit_text_id
				left outer join drug_name on drug_name_id = drug_name.id
				left outer join drug_route on drug_route_id = drug_route.id
				left outer join drug_form on drug_form_id = drug_form.id
				where patient_id = ? and localized_text.language_id = ? order by unlinked_dntf_treatment.creation_date desc`,
		patientID, EN_LANGUAGE_ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	treatments, err := d.getUnlinkedDNTFTreatmentsFromRow(rows)
	if err != nil {
		return nil, err
	}

	return treatments, err
}

func (d *DataService) getUnlinkedDNTFTreatmentsFromRow(rows *sql.Rows) ([]*common.Treatment, error) {
	treatments := make([]*common.Treatment, 0)
	for rows.Next() {
		var dispenseUnitId, doctorID, patientID, unlinkedDntfTreatmentId, pharmacyID, erxID encoding.ObjectID
		var dispenseValue encoding.HighPrecisionFloat64
		var drugInternalName, dosageStrength, treatmentType, dispenseUnitDescription, pharmacyNotes, patientInstructions string
		var status common.TreatmentStatus
		var creationDate time.Time
		var daysSupply, refills encoding.NullInt64
		var erxSentDate, erxLastFilledDate mysql.NullTime
		var drugName, drugRoute, drugForm sql.NullString
		var substitutionsAllowed bool
		var isControlledSubstance sql.NullBool
		err := rows.Scan(&unlinkedDntfTreatmentId, &erxID, &drugInternalName, &dosageStrength, &treatmentType, &dispenseValue, &dispenseUnitId, &dispenseUnitDescription,
			&refills, &substitutionsAllowed, &daysSupply, &pharmacyID, &pharmacyNotes, &patientInstructions, &creationDate, &erxSentDate, &erxLastFilledDate, &status, &drugName, &drugRoute, &drugForm, &patientID, &doctorID, &isControlledSubstance)
		if err != nil {
			return nil, err
		}

		treatment := &common.Treatment{
			ID:                      unlinkedDntfTreatmentId,
			PatientID:               patientID,
			DoctorID:                doctorID,
			DrugInternalName:        drugInternalName,
			DosageStrength:          dosageStrength,
			DispenseValue:           dispenseValue,
			DispenseUnitID:          dispenseUnitId,
			DispenseUnitDescription: dispenseUnitDescription,
			NumberRefills:           refills,
			SubstitutionsAllowed:    substitutionsAllowed,
			DaysSupply:              daysSupply,
			DrugName:                drugName.String,
			DrugForm:                drugForm.String,
			DrugRoute:               drugRoute.String,
			PatientInstructions:     patientInstructions,
			CreationDate:            &creationDate,
			Status:                  status,
			PharmacyNotes:           pharmacyNotes,
			OTC:                     treatmentType == treatmentOTC,
			IsControlledSubstance: isControlledSubstance.Bool,
			ERx: &common.ERxData{
				ErxLastDateFilled: &erxLastFilledDate.Time,
				ErxSentDate:       &erxSentDate.Time,
			},
		}

		if pharmacyID.IsValid {
			treatment.ERx.PharmacyLocalID = pharmacyID
			treatment.ERx.Pharmacy, err = d.GetPharmacyFromID(pharmacyID.Int64())
			if err != nil {
				return nil, err
			}
		}

		treatment.ERx.PrescriptionID = erxID

		treatment.Doctor, err = d.GetDoctorFromID(treatment.DoctorID.Int64())
		if err != nil {
			return nil, err
		}

		treatment.Patient, err = d.GetPatientFromID(treatment.PatientID.Int64())
		if err != nil {
			return nil, err
		}

		treatment.ERx.RxHistory, err = d.GetErxStatusEventsForDNTFTreatment(unlinkedDntfTreatmentId.Int64())
		if err != nil {
			return nil, err
		}
		treatments = append(treatments, treatment)

	}
	return treatments, rows.Err()
}

func (d *DataService) AddTreatmentToTreatmentPlanInEventOfDNTF(treatment *common.Treatment, refillRequestID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	if err := d.addTreatment(treatmentForPatientType, treatment, nil, tx); err != nil {
		tx.Rollback()
		return err
	}

	if treatment.DoctorTreatmentTemplateID.Int64() != 0 {
		_, err = tx.Exec(`insert into treatment_dr_template_selection (treatment_id, dr_treatment_template_id) values (?,?)`, treatment.ID.Int64(), treatment.DoctorTreatmentTemplateID.Int64())
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.Exec(`insert into erx_status_events (treatment_id, erx_status, status) values (?,?,?)`, treatment.ID.Int64(), ERX_STATUS_NEW_RX_FROM_DNTF, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`insert into dntf_mapping (treatment_id, rx_refill_request_id) values (?,?)`, treatment.ID.Int64(), refillRequestID)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) AddErxStatusEventForDNTFTreatment(statusEvent common.StatusEvent) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`update unlinked_dntf_treatment_status_events set status = ? where unlinked_dntf_treatment_id = ? and status = ?`, STATUS_INACTIVE, statusEvent.ItemID, STATUS_ACTIVE)
	if err != nil {
		tx.Rollback()
		return err
	}

	columnsAndData := map[string]interface{}{
		"unlinked_dntf_treatment_id": statusEvent.ItemID,
		"erx_status":                 statusEvent.Status,
		"status":                     STATUS_ACTIVE,
	}

	if statusEvent.StatusDetails != "" {
		columnsAndData["event_details"] = statusEvent.StatusDetails
	}

	if !statusEvent.ReportedTimestamp.IsZero() {
		columnsAndData["reported_timestamp"] = statusEvent.ReportedTimestamp
	}

	columns, values := getKeysAndValuesFromMap(columnsAndData)

	_, err = tx.Exec(fmt.Sprintf(`insert into unlinked_dntf_treatment_status_events (%s) values (%s)`, strings.Join(columns, ","), dbutil.MySQLArgs(len(values))), values...)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (d *DataService) GetErxStatusEventsForDNTFTreatment(treatmentID int64) ([]common.StatusEvent, error) {
	rows, err := d.db.Query(`
		SELECT un.unlinked_dntf_treatment_id, unlinked_dntf_treatment.erx_id, un.erx_status, un.event_details, un.status, un.creation_date 
		FROM unlinked_dntf_treatment_status_events un
		INNER JOIN unlinked_dntf_treatment ON unlinked_dntf_treatment_id = unlinked_dntf_treatment.id
		WHERE unlinked_dntf_treatment.id = ? 
		ORDER BY un.creation_date DESC, un.id DESC`, treatmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statusEvents := make([]common.StatusEvent, 0)
	for rows.Next() {
		var statusDetails sql.NullString
		var statusEventItem common.StatusEvent
		if err := rows.Scan(&statusEventItem.ItemID, &statusEventItem.PrescriptionID, &statusEventItem.Status, &statusDetails, &statusEventItem.InternalStatus, &statusEventItem.StatusTimestamp); err != nil {
			return nil, err
		}
		statusEventItem.StatusDetails = statusDetails.String
		statusEvents = append(statusEvents, statusEventItem)
	}
	return statusEvents, rows.Err()
}

func (d *DataService) GetErxStatusEventsForDNTFTreatmentBasedOnPatientID(patientID int64) ([]common.StatusEvent, error) {
	rows, err := d.db.Query(`
		SELECT un.unlinked_dntf_treatment_id, unlinked_dntf_treatment.erx_id, un.erx_status, un.status, un.creation_date 
		FROM unlinked_dntf_treatment_status_events un
		INNER JOIN unlinked_dntf_treatment ON unlinked_dntf_treatment_id = unlinked_dntf_treatment.id
		WHERE unlinked_dntf_treatment.patient_id = ? 
		ORDER BY un.creation_date DESC, un.id DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statusEvents := make([]common.StatusEvent, 0)
	for rows.Next() {
		var statusEventItem common.StatusEvent
		if err := rows.Scan(&statusEventItem.ItemID, &statusEventItem.PrescriptionID, &statusEventItem.Status, &statusEventItem.InternalStatus, &statusEventItem.StatusTimestamp); err != nil {
			return nil, err
		}
		statusEvents = append(statusEvents, statusEventItem)
	}

	return statusEvents, rows.Err()
}
