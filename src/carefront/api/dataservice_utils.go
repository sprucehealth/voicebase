package api

import (
	"carefront/common"
	"database/sql"
	"strconv"
	"strings"
)

const (
	STATUS_ACTIVE                      = "ACTIVE"
	STATUS_CREATED                     = "CREATED"
	STATUS_CREATING                    = "CREATING"
	STATUS_DELETING                    = "DELETING"
	STATUS_UPDATING                    = "UPDATING"
	STATUS_DELETED                     = "DELETED"
	STATUS_INACTIVE                    = "INACTIVE"
	STATUS_PENDING                     = "PENDING"
	STATUS_ONGOING                     = "ONGOING"
	ERX_STATUS_SENDING                 = "Sending"
	ERX_STATUS_SENT                    = "eRxSent"
	ERX_STATUS_ERROR                   = "Error"
	ERX_STATUS_SEND_ERROR              = "Send_Error"
	ERX_STATUS_DELETED                 = "Deleted"
	ERX_STATUS_RESOLVED                = "Resolved"
	ERX_STATUS_NEW_RX_FROM_DNTF        = "NewRxFromDNTF"
	treatmentOTC                       = "OTC"
	treatmentRX                        = "RX"
	RX_REFILL_STATUS_SENT              = "RefillRxSent"
	RX_REFILL_STATUS_DELETED           = "RefillRxDeleted"
	RX_REFILL_STATUS_ERROR             = "RefillRxError"
	RX_REFILL_STATUS_ERROR_RESOLVED    = "RefillRxErrorResolved"
	RX_REFILL_STATUS_REQUESTED         = "RefillRxRequested"
	RX_REFILL_STATUS_APPROVED          = "RefillRxApproved"
	RX_REFILL_STATUS_DENIED            = "RefillRxDenied"
	RX_REFILL_DNTF_REASON_CODE         = "DeniedNewRx"
	drDrugSupplementalInstructionTable = "dr_drug_supplemental_instruction"
	drRegimenStepTable                 = "dr_regimen_step"
	drAdvicePointTable                 = "dr_advice_point"
	drugNameTable                      = "drug_name"
	drugFormTable                      = "drug_form"
	drugRouteTable                     = "drug_route"
	doctorPhoneType                    = "MAIN"
	SpruceButtonBaseActionUrl          = "spruce:///action/"
	SpruceImageBaseUrl                 = "spruce:///image/"
	treatmentTable                     = "treatment"
	pharmacyDispensedTreatmentTable    = "pharmacy_dispensed_treatment"
	requestedTreatmentTable            = "requested_treatment"
	unlinkedDntfTreatmentTable         = "unlinked_dntf_treatment"
	asDoctorTemplate                   = true
	asPatientTreatment                 = false
	addressUsa                         = "USA"
	PENDING_TASK_PATIENT_CARD          = "PATIENT_CARD"
)

type DataService struct {
	DB *sql.DB
}

func infoIdsFromMap(m map[int64]*common.AnswerIntake) []int64 {
	infoIds := make([]int64, 0, len(m))
	for key := range m {
		infoIds = append(infoIds, key)
	}
	return infoIds
}

func createKeysArrayFromMap(m map[int64]bool) []int64 {
	keys := make([]int64, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func createValuesArrayFromMap(m map[int64]int64) []int64 {
	values := make([]int64, 0, len(m))
	for _, value := range m {
		values = append(values, value)
	}
	return values
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
	values := make([]interface{}, 0)
	keys := make([]string, 0)
	for key, value := range m {
		keys = append(keys, key)
		values = append(values, value)
	}
	return keys, values
}

func nReplacements(n int) string {
	if n == 0 {
		return ""
	}

	result := make([]byte, 2*n-1)
	for i := 0; i < len(result)-1; i += 2 {
		result[i] = '?'
		result[i+1] = ','
	}
	result[len(result)-1] = '?'
	return string(result)
}

func appendStringsToInterfaceSlice(interfaceSlice []interface{}, strSlice []string) []interface{} {
	for _, strItem := range strSlice {
		interfaceSlice = append(interfaceSlice, strItem)
	}
	return interfaceSlice
}

func appendInt64sToInterfaceSlice(interfaceSlice []interface{}, int64Slice []int64) []interface{} {
	for _, int64Item := range int64Slice {
		interfaceSlice = append(interfaceSlice, int64Item)
	}
	return interfaceSlice
}
