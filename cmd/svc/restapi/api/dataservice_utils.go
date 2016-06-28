package api

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

const (
	StatusActive      = "ACTIVE"
	StatusCommitted   = "COMMITTED"
	StatusCreating    = "CREATING"
	StatusDeleted     = "DELETED"
	StatusDeleting    = "DELETING"
	StatusDeprecated  = "DEPRECATED"
	StatusInactive    = "INACTIVE"
	StatusOngoing     = "ONGOING"
	StatusPending     = "PENDING"
	StatusTemp        = "TEMP"
	StatusUncommitted = "UNCOMMITTED"
	StatusUpdating    = "UPDATING"
)

const (
	ERXStatusDeleted       = "Deleted"
	ERXStatusEntered       = "Entered"
	ERXStatusError         = "Error"
	ERXStatusNewRXFromDNTF = "NewRxFromDNTF"
	ERXStatusResolved      = "Resolved"
	ERXStatusSendError     = "Send_Error"
	ERXStatusSending       = "Sending"
	ERXStatusSent          = "eRxSent"
)

const (
	RXRefillDNTFReasonCode      = "DeniedNewRx"
	RXRefillStatusApproved      = "RefillRxApproved"
	RXRefillStatusDeleted       = "RefillRxDeleted"
	RXRefillStatusDenied        = "RefillRxDenied"
	RXRefillStatusError         = "RefillRxError"
	RXRefillStatusErrorResolved = "RefillRxErrorResolved"
	RXRefillStatusRequested     = "RefillRxRequested"
	RXRefillStatusSent          = "RefillRxSent"
)

const (
	treatmentOTC                    = "OTC"
	treatmentRX                     = "RX"
	drugNameTable                   = "drug_name"
	drugFormTable                   = "drug_form"
	drugRouteTable                  = "drug_route"
	pharmacyDispensedTreatmentTable = "pharmacy_dispensed_treatment"
	requestedTreatmentTable         = "requested_treatment"
	addressUSA                      = "USA"
	PendingTaskPatientCard          = "PATIENT_CARD"
)

type dataService struct {
	db                           tsql.DB
	roleTypeMapping              map[string]int64
	roleIDMapping                map[int64]string
	pathwayMapMu                 *sync.RWMutex
	pathwayTagToIDMap            map[string]int64
	pathwayIDToTagMap            map[int64]string
	skuMapMu                     *sync.RWMutex
	skuTypeToIDMap               map[string]int64
	skuIDToTypeMap               map[int64]string
	cfgStore                     cfg.Store
	metricsRegistry              metrics.Registry
	accoundCodeCollisionsCounter *metrics.Counter
}

func NewDataService(db *sql.DB, cfgStore cfg.Store, metricsRegistry metrics.Registry) (DataAPI, error) {
	dataService := &dataService{
		db:                tsql.AsDB(db),
		roleTypeMapping:   make(map[string]int64),
		roleIDMapping:     make(map[int64]string),
		pathwayMapMu:      &sync.RWMutex{},
		pathwayTagToIDMap: make(map[string]int64),
		pathwayIDToTagMap: make(map[int64]string),
		skuMapMu:          &sync.RWMutex{},
		skuTypeToIDMap:    make(map[string]int64),
		skuIDToTypeMap:    make(map[int64]string),
		cfgStore:          cfgStore,
		metricsRegistry:   metricsRegistry,
	}

	// get the role type mapping into memory for quick access
	rows, err := dataService.db.Query(`
		SELECT id, role_type_tag
		FROM role_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var roleTypeTag string
		if err := rows.Scan(&id, &roleTypeTag); err != nil {
			return nil, err
		}
		dataService.roleTypeMapping[roleTypeTag] = id
		dataService.roleIDMapping[id] = roleTypeTag
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Track metrics related to collisions for scaling purposes
	dataService.accoundCodeCollisionsCounter = metrics.NewCounter()
	dataService.metricsRegistry.Add("AssociateRandomAccountCode/collisions", dataService.accoundCodeCollisionsCounter)

	registerCfgValues(cfgStore)
	return dataService, rows.Err()
}

// Transact encapsulated the provided function in a transaction and handles rollback and commit actions
func (d *dataService) Transact(trans func(dal DataAPI) error) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	// Recover from any inner panics that happened and close the transaction
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = errors.Trace(fmt.Errorf("Encountered panic during transaction execution: %v", r))
		}
	}()
	tdal := &dataService{
		db:                tsql.AsSafeTx(tx),
		roleTypeMapping:   d.roleTypeMapping,
		roleIDMapping:     d.roleIDMapping,
		pathwayMapMu:      d.pathwayMapMu,
		pathwayTagToIDMap: d.pathwayTagToIDMap,
		pathwayIDToTagMap: d.pathwayIDToTagMap,
		skuMapMu:          d.skuMapMu,
		skuTypeToIDMap:    d.skuTypeToIDMap,
		skuIDToTypeMap:    d.skuIDToTypeMap,
		cfgStore:          d.cfgStore,
		metricsRegistry:   d.metricsRegistry,
	}
	if err := trans(tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	err = errors.Trace(tx.Commit())
	return err
}

// registerCfgValues is a bootstrap helper function to group and initialize the Cfg values used by the data service
func registerCfgValues(cfgStore cfg.Store) {
	cfgStore.Register(doctorFTPQueryMaxThreads)
}

func (d *dataService) skuTypeFromID(id int64) (string, error) {
	d.skuMapMu.RLock()
	skuType, ok := d.skuIDToTypeMap[id]
	d.skuMapMu.RUnlock()
	if ok {
		return skuType, nil
	}

	row := d.db.QueryRow(`SELECT type FROM sku WHERE id = ?`, id)
	if err := row.Scan(&skuType); err == sql.ErrNoRows {
		return "", ErrNotFound("sku")
	} else if err != nil {
		return "", err
	}

	d.skuMapMu.Lock()
	d.skuIDToTypeMap[id] = skuType
	d.skuTypeToIDMap[skuType] = id
	d.skuMapMu.Unlock()

	return skuType, nil
}

func (d *dataService) skuIDFromType(skuType string) (int64, error) {
	d.skuMapMu.RLock()
	skuID, ok := d.skuTypeToIDMap[skuType]
	d.skuMapMu.RUnlock()
	if ok {
		return skuID, nil
	}

	row := d.db.QueryRow(`SELECT id FROM sku WHERE type = ?`, skuType)
	if err := row.Scan(&skuID); err == sql.ErrNoRows {
		return 0, ErrNotFound("sku")
	} else if err != nil {
		return 0, err
	}

	d.skuMapMu.Lock()
	d.skuIDToTypeMap[skuID] = skuType
	d.skuTypeToIDMap[skuType] = skuID
	d.skuMapMu.Unlock()

	return skuID, nil
}

func (d *dataService) pathwayTagFromID(id int64) (string, error) {
	d.pathwayMapMu.RLock()
	tag, ok := d.pathwayIDToTagMap[id]
	d.pathwayMapMu.RUnlock()
	if ok {
		return tag, nil
	}

	row := d.db.QueryRow(`SELECT tag FROM clinical_pathway WHERE id = ?`, id)
	if err := row.Scan(&tag); err == sql.ErrNoRows {
		return "", ErrNotFound("clinical_pathway")
	} else if err != nil {
		return "", err
	}

	d.pathwayMapMu.Lock()
	d.pathwayIDToTagMap[id] = tag
	d.pathwayTagToIDMap[tag] = id
	d.pathwayMapMu.Unlock()
	return tag, nil
}

func (d *dataService) pathwayIDFromTag(tag string) (int64, error) {
	tag = strings.ToLower(tag)

	d.pathwayMapMu.RLock()
	id, ok := d.pathwayTagToIDMap[tag]
	d.pathwayMapMu.RUnlock()
	if ok {
		return id, nil
	}

	row := d.db.QueryRow(`SELECT id FROM clinical_pathway WHERE tag = ?`, tag)
	if err := row.Scan(&id); err == sql.ErrNoRows {
		return 0, ErrNotFound("clinical_pathway")
	} else if err != nil {
		return 0, err
	}

	d.pathwayMapMu.Lock()
	d.pathwayTagToIDMap[tag] = id
	d.pathwayIDToTagMap[id] = tag
	d.pathwayMapMu.Unlock()
	return id, nil
}

func enumerateItemsIntoString(ids []int64) string {
	if ids == nil || len(ids) == 0 {
		return ""
	}
	idsStr := make([]string, len(ids))
	for i, id := range ids {
		idsStr[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(idsStr, ",")
}

func getKeysAndValuesFromMap(m map[string]interface{}) ([]string, []interface{}) {
	var values []interface{}
	var keys []string
	for key, value := range m {
		keys = append(keys, key)
		values = append(values, value)
	}
	return keys, values
}

// ByCareProviderRole is a type that enables sorting of care provider
// assignments to surface doctors to the top of the list of care providers.
type ByCareProviderRole []*common.CareProviderAssignment

func (c ByCareProviderRole) Len() int           { return len(c) }
func (c ByCareProviderRole) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c ByCareProviderRole) Less(i, j int) bool { return c[i].ProviderRole == RoleDoctor }

type treatmentType int64

const (
	treatmentForPatientType treatmentType = iota
	pharmacyDispensedTreatmentType
	refillRequestTreatmentType
	unlinkedDNTFTreatmentType
	doctorFavoriteTreatmentType
)

var possibleTreatmentTables = map[treatmentType]string{
	treatmentForPatientType:        "treatment",
	pharmacyDispensedTreatmentType: "pharmacy_dispensed_treatment",
	refillRequestTreatmentType:     "requested_treatment",
	unlinkedDNTFTreatmentType:      "unlinked_dntf_treatment",
	doctorFavoriteTreatmentType:    "dr_favorite_treatment",
}

func (d *dataService) addTreatment(tType treatmentType, treatment *common.Treatment, params map[string]interface{}, db db) error {
	medicationType := treatmentRX
	if treatment.OTC {
		medicationType = treatmentOTC
	}

	// collecting data for fields that are common to all types of treatment
	columnsAndData := map[string]interface{}{
		"drug_internal_name":      treatment.DrugInternalName,
		"dosage_strength":         treatment.DosageStrength,
		"type":                    medicationType,
		"dispense_value":          treatment.DispenseValue.Float64(),
		"refills":                 treatment.NumberRefills.Int64Value,
		"substitutions_allowed":   treatment.SubstitutionsAllowed,
		"patient_instructions":    treatment.PatientInstructions,
		"pharmacy_notes":          treatment.PharmacyNotes,
		"status":                  common.TStatusCreated.String(),
		"is_controlled_substance": treatment.IsControlledSubstance,
	}

	if treatment.DaysSupply.IsValid {
		columnsAndData["days_supply"] = treatment.DaysSupply.Int64Value
	}

	// Do the drug_ table lookups outside of the transaction to allow new values to be seen by subsequent calls.
	// Also, it's totally fine for these to succeed even if the tx is rolled back.
	if err := d.includeDrugNameComponentIfNonZero(treatment.GenericDrugName, drugNameTable, "generic_drug_name_id", columnsAndData, d.db); err != nil {
		return err
	}

	if err := d.includeDrugNameComponentIfNonZero(treatment.DrugName, drugNameTable, "drug_name_id", columnsAndData, d.db); err != nil {
		return err
	}

	if err := d.includeDrugNameComponentIfNonZero(treatment.DrugForm, drugFormTable, "drug_form_id", columnsAndData, d.db); err != nil {
		return err
	}

	if err := d.includeDrugNameComponentIfNonZero(treatment.DrugRoute, drugRouteTable, "drug_route_id", columnsAndData, d.db); err != nil {
		return err
	}

	// add any treatment type specific information to the table
	switch tType {
	case treatmentForPatientType:
		columnsAndData["dispense_unit_id"] = treatment.DispenseUnitID.Int64()
		if treatment.TreatmentPlanID.Int64() != 0 {
			columnsAndData["treatment_plan_id"] = treatment.TreatmentPlanID.Int64()
		}
	case doctorFavoriteTreatmentType:
		columnsAndData["dispense_unit_id"] = treatment.DispenseUnitID.Int64()
		drFavoriteTreatmentID, ok := params["dr_favorite_treatment_plan_id"]
		if !ok {
			return errors.New("Expected dr_favorite_treatment_planid to be present in the params but it wasnt")
		}
		columnsAndData["dr_favorite_treatment_plan_id"] = drFavoriteTreatmentID
	case pharmacyDispensedTreatmentType:
		columnsAndData["doctor_id"] = treatment.Doctor.ID.Int64()
		columnsAndData["erx_id"] = treatment.ERx.PrescriptionID.Int64()

		if treatment.ERx.ErxLastDateFilled != nil && !treatment.ERx.ErxLastDateFilled.IsZero() {
			columnsAndData["erx_last_filled_date"] = treatment.ERx.ErxLastDateFilled
		}

		if treatment.ERx.ErxSentDate != nil && !treatment.ERx.ErxSentDate.IsZero() {
			columnsAndData["erx_sent_date"] = treatment.ERx.ErxSentDate
		}

		columnsAndData["pharmacy_id"] = treatment.ERx.PharmacyLocalID.Int64()
		columnsAndData["dispense_unit"] = treatment.DispenseUnitDescription
		requestedTreatment, ok := params["requested_treatment"].(*common.Treatment)
		if !ok {
			return errors.New("Expected requested_treatment to be present in the params for adding a pharmacy_dispensed_treatment")
		}
		columnsAndData["requested_treatment_id"] = requestedTreatment.ID.Int64()

	case refillRequestTreatmentType:
		columnsAndData["doctor_id"] = treatment.Doctor.ID.Int64()
		columnsAndData["erx_id"] = treatment.ERx.PrescriptionID.Int64()

		if treatment.ERx.ErxLastDateFilled != nil && !treatment.ERx.ErxLastDateFilled.IsZero() {
			columnsAndData["erx_last_filled_date"] = treatment.ERx.ErxLastDateFilled
		}

		if treatment.ERx.ErxSentDate != nil && !treatment.ERx.ErxSentDate.IsZero() {
			columnsAndData["erx_sent_date"] = treatment.ERx.ErxSentDate
		}

		columnsAndData["pharmacy_id"] = treatment.ERx.PharmacyLocalID.Int64()
		columnsAndData["dispense_unit"] = treatment.DispenseUnitDescription
		if treatment.OriginatingTreatmentID != 0 {
			columnsAndData["originating_treatment_id"] = treatment.OriginatingTreatmentID
		}

	case unlinkedDNTFTreatmentType:
		columnsAndData["doctor_id"] = treatment.DoctorID.Int64()
		columnsAndData["patient_id"] = treatment.PatientID.Int64()
		columnsAndData["dispense_unit_id"] = treatment.DispenseUnitID.Int64()

	default:
		return errors.New("Unexpected type of treatment trying to be added to a table")
	}

	columns, values := getKeysAndValuesFromMap(columnsAndData)
	res, err := db.Exec(fmt.Sprintf(`insert into %s (%s) values (%s)`, possibleTreatmentTables[tType], strings.Join(columns, ","), dbutil.MySQLArgs(len(values))), values...)
	if err != nil {
		return err
	}

	treatmentID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// update the treatment object with the information
	treatment.ID = encoding.DeprecatedNewObjectID(treatmentID)

	st, err := db.Prepare(fmt.Sprintf(`INSERT INTO %s_drug_db_id (drug_db_id_tag, drug_db_id, %s_id) VALUES (?, ?, ?)`,
		possibleTreatmentTables[tType], possibleTreatmentTables[tType]))
	if err != nil {
		return err
	}
	defer st.Close()

	// add drug db ids to the table
	for drugDBTag, drugDBID := range treatment.DrugDBIDs {
		if _, err := st.Exec(drugDBTag, drugDBID, treatmentID); err != nil {
			return err
		}
	}

	return nil
}

func (d *dataService) includeDrugNameComponentIfNonZero(drugNameComponent, tableName, columnName string, columnsAndData map[string]interface{}, db db) error {
	if drugNameComponent != "" {
		componentID, err := d.getOrInsertNameInTable(db, tableName, drugNameComponent)
		if err != nil {
			return err
		}
		columnsAndData[columnName] = componentID
	}
	return nil
}