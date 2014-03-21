package api

import (
	"carefront/common"
	"database/sql"
	"strconv"
	"strings"
)

const (
	STATUS_ACTIVE                           = "ACTIVE"
	STATUS_CREATED                          = "CREATED"
	STATUS_CREATING                         = "CREATING"
	STATUS_DELETING                         = "DELETING"
	STATUS_UPDATING                         = "UPDATING"
	STATUS_DELETED                          = "DELETED"
	STATUS_INACTIVE                         = "INACTIVE"
	STATUS_PENDING                          = "PENDING"
	STATUS_ONGOING                          = "ONGOING"
	ERX_STATUS_SENDING                      = "Sending"
	ERX_STATUS_SENT                         = "eRxSent"
	ERX_STATUS_ERROR                        = "Error"
	ERX_STATUS_DELETED                      = "Deleted"
	ERX_STATUS_SEND_ERROR                   = "Send_Error"
	ERX_STATUS_RESOLVED                     = "Resolved"
	treatment_otc                           = "OTC"
	treatment_rx                            = "RX"
	RX_REFILL_STATUS_SENT                   = "RefillRxSent"
	RX_REFILL_STATUS_DELETED                = "RefillRxDeleted"
	RX_REFILL_STATUS_ERROR                  = "RefillRxError"
	RX_REFILL_STATUS_ERROR_RESOLVED         = "RefillRxErrorResolved"
	RX_REFILL_STATUS_REQUESTED              = "RefillRxRequested"
	RX_REFILL_STATUS_APPROVED               = "RefillRxApproved"
	RX_REFILL_STATUS_DENIED                 = "RefillRxDenied"
	dr_drug_supplemental_instruction_table  = "dr_drug_supplemental_instruction"
	dr_regimen_step_table                   = "dr_regimen_step"
	dr_advice_point_table                   = "dr_advice_point"
	drug_name_table                         = "drug_name"
	drug_form_table                         = "drug_form"
	drug_route_table                        = "drug_route"
	doctor_phone_type                       = "MAIN"
	SpruceButtonBaseActionUrl               = "spruce:///action/"
	SpruceImageBaseUrl                      = "spruce:///image/"
	table_name_treatment                    = "treatment"
	table_name_pharmacy_dispensed_treatment = "pharmacy_dispensed_treatment"
	table_name_requested_treatment          = "requested_treatment"
	without_link_to_treatment_plan          = true
	with_link_to_treatment_plan             = false
	address_usa                             = "USA"
	PENDING_TASK_PATIENT_CARD               = "PATIENT_CARD"
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
