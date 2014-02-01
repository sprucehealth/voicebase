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
	ERX_STATUS_SENDING                     = "SENDING"
	ERX_STATUS_SENT                        = "SENT"
	ERX_STATUS_ERROR                       = "ERROR"
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
)

type DataService struct {
	DB *sql.DB
}

func infoIdsFromMap(m map[int64]*common.AnswerIntake) []int64 {
	infoIds := make([]int64, 0)
	for key, _ := range m {
		infoIds = append(infoIds, key)
	}
	return infoIds
}

func createKeysArrayFromMap(m map[int64]bool) []int64 {
	keys := make([]int64, 0)
	for key, _ := range m {
		keys = append(keys, key)
	}
	return keys
}

func createValuesArrayFromMap(m map[int64]int64) []int64 {
	values := make([]int64, 0)
	for _, value := range m {
		values = append(values, value)
	}
	return values
}

func enumerateItemsIntoString(ids []int64) string {
	if ids == nil || len(ids) == 0 {
		return ""
	}
	idsStr := make([]string, 0)
	for _, id := range ids {
		idsStr = append(idsStr, strconv.FormatInt(id, 10))
	}
	return strings.Join(idsStr, ",")
}
