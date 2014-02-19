package api

import (
	"carefront/common"
	"database/sql"
	"strconv"
	"strings"
)

const (
	status_active                          = "ACTIVE"
	status_created                         = "CREATED"
	status_creating                        = "CREATING"
	status_deleted                         = "DELETED"
	status_inactive                        = "INACTIVE"
	status_pending                         = "PENDING"
	status_ongoing                         = "ONGOING"
	ERX_STATUS_SENDING                     = "Sending"
	ERX_STATUS_SENT                        = "eRxSent"
	ERX_STATUS_ERROR                       = "Error"
	ERX_STATUS_SEND_ERROR                  = "Send_Error"
	treatment_otc                          = "OTC"
	treatment_rx                           = "RX"
	dr_drug_supplemental_instruction_table = "dr_drug_supplemental_instruction"
	dr_regimen_step_table                  = "dr_regimen_step"
	dr_advice_point_table                  = "dr_advice_point"
	drug_name_table                        = "drug_name"
	drug_form_table                        = "drug_form"
	drug_route_table                       = "drug_route"
	doctor_phone_type                      = "MAIN"
	SpruceButtonBaseActionUrl              = "spruce:///action/"
	SpruceImageBaseUrl                     = "spruce:///image/"
	event_type_patient_visit               = "PATIENT_VISIT"
	event_type_treatment_plan              = "TREATMENT_PLAN"
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
