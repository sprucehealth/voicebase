package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

const (
	// We will have to deal with collisions in the generation space.
	// We don't want to do this forever so cap our attempts at some reasonable value
	maxAccountCodeGenerationAttempts = 100
	accountCodeUpperBound            = 9999999
	accountCodeLowerBound            = 1000000
)

var (
	// ErrValidAccountCodeNoActiveReferralProgram is a value to be returned when the provided code is a valid promo code but does not map to an active referral program
	ErrValidAccountCodeNoActiveReferralProgram = errors.New("The provided code was a valid account code but did not map to an active referral program.")
	errPromoCodeDoesNotExist                   = errors.New("Promotion code does not exist")
)

// AccountPromotionOption represents an option for promotion manipulation
type AccountPromotionOption int

const (
	// APOPendingOnly implies that the operation should be applied to Pending promotions only
	APOPendingOnly AccountPromotionOption = 1 << iota

	// APONone implies no option
	APONone AccountPromotionOption = 0
)

// Has returns a boolean value representing if the desired option is in the bit set
func (apo AccountPromotionOption) Has(o AccountPromotionOption) bool {
	return (apo & o) != 0
}

// LookupPromoCode returns the promotion_code record of the indicated promo code.
// If the code provided maps to an account_code then the promo code for that account's active referral_program will be returned
func (d *dataService) LookupPromoCode(code string) (*common.PromoCode, error) {
	// Determine if the code provided is an account_code and if so we should return the promo_code of the active referral_program for that account
	// We know account codes are purely numeric. So if that doesn't pass we know to treat it as a standard promo_code
	if accountCode, err := strconv.ParseInt(code, 10, 64); err == nil {
		account, err := d.AccountForAccountCode(uint64(accountCode))
		if err != nil && !IsErrNotFound(err) {
			return nil, errors.Trace(err)
		}
		if !IsErrNotFound(err) {
			// If the account has generated an account code then it should have an associated active referral_program
			referralProgram, err := d.ActiveReferralProgramForAccount(account.ID, common.PromotionTypes)
			if IsErrNotFound(err) {
				return nil, ErrValidAccountCodeNoActiveReferralProgram
			} else if err != nil {
				return nil, errors.Trace(err)
			}

			code = referralProgram.Code
		}
	}

	var promoCode common.PromoCode
	err := d.db.QueryRow(`SELECT id, code, is_referral FROM promotion_code WHERE code = ?`, code).Scan(&promoCode.ID, &promoCode.Code, &promoCode.IsReferral)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("promotion_code"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &promoCode, nil
}

// PromoCodeForAccountExists determines if an account has claimed a given promo code
func (d *dataService) PromoCodeForAccountExists(accountID, codeID int64) (bool, error) {
	var id int64
	if err := d.db.QueryRow(`SELECT promotion_code_id FROM account_promotion
		WHERE account_id = ? AND promotion_code_id = ? AND status != ? LIMIT 1`, accountID, codeID, common.PSDeleted.String()).
		Scan(&id); err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, errors.Trace(err)
	}
	return true, nil
}

// PromotionCountInGroupForAccount returns the number of promotions claimed in a given group by a user
func (d *dataService) PromotionCountInGroupForAccount(accountID int64, group string) (int, error) {
	var count int
	if err := d.db.QueryRow(`
		SELECT count(*)
		FROM account_promotion
		INNER JOIN promotion_group ON promotion_group.id = promotion_group_id
		WHERE promotion_group.name = ?
		AND account_id = ?
		AND account_promotion.status != ?`, group, accountID, common.PSDeleted.String()).Scan(&count); err == sql.ErrNoRows {
		return 0, errors.Trace(ErrNotFound("account_promotion"))
	} else if err != nil {
		return 0, errors.Trace(err)
	}

	return count, nil
}

// PromoCodePrefixes returns the set of available promo code prefixes
func (d *dataService) PromoCodePrefixes() ([]string, error) {
	rows, err := d.db.Query(`SELECT prefix FROM promo_code_prefix where status = 'ACTIVE'`)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var prefixes []string
	for rows.Next() {
		var prefix string
		if err := rows.Scan(&prefix); err != nil {
			return nil, errors.Trace(err)
		}

		prefixes = append(prefixes, prefix)
	}

	return prefixes, errors.Trace(rows.Err())
}

// CreatePromoCodePrefix inserts a new promo_code_prefix record with the provided prefix
func (d *dataService) CreatePromoCodePrefix(prefix string) error {
	_, err := d.db.Exec(`INSERT INTO promo_code_prefix (prefix, status) VALUES (?,?)`, prefix, StatusActive)
	return errors.Trace(err)
}

// CreatePromotionGroup inserts a new promotion_group matching the dynamic aspects of the provided promotionGroup
func (d *dataService) CreatePromotionGroup(promotionGroup *common.PromotionGroup) (int64, error) {
	res, err := d.db.Exec(`INSERT INTO promotion_group (name, max_allowed_promos) VALUES (?, ?)`, promotionGroup.Name, promotionGroup.MaxAllowedPromos)
	if err != nil {
		return 0, errors.Trace(err)
	}
	promotionGroup.ID, err = res.LastInsertId()
	if err != nil {
		return 0, errors.Trace(err)
	}
	return promotionGroup.ID, nil
}

// PromotionGroup returns the promotion_group data record matching the provided name
func (d *dataService) PromotionGroup(name string) (*common.PromotionGroup, error) {
	var promotionGroup common.PromotionGroup
	if err := d.db.QueryRow(`SELECT id, name, max_allowed_promos FROM promotion_group WHERE name = ?`, name).
		Scan(&promotionGroup.ID, &promotionGroup.Name, &promotionGroup.MaxAllowedPromos); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("promotion_group"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &promotionGroup, nil
}

// PromotionGroups returns all promotion_group records
func (d *dataService) PromotionGroups() ([]*common.PromotionGroup, error) {
	rows, err := d.db.Query(`SELECT name, max_allowed_promos FROM promotion_group ORDER BY name ASC`)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var promotionGroups []*common.PromotionGroup
	for rows.Next() {
		promotionGroup := &common.PromotionGroup{}
		if err := rows.Scan(&promotionGroup.Name, &promotionGroup.MaxAllowedPromos); err != nil {
			return nil, errors.Trace(err)
		}
		promotionGroups = append(promotionGroups, promotionGroup)
	}

	return promotionGroups, errors.Trace(rows.Err())
}

// CreatePromotion inserts a new promotino record matching the dynamic aspects of the provided promotion and returns the ID of the record
func (d *dataService) CreatePromotion(promotion *common.Promotion) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, errors.Trace(err)
	}

	id, err := createPromotion(tx, promotion)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	return id, errors.Trace(tx.Commit())
}

// Promotion returns the promotion record matching the provided promotion_code ID and maps to the provided type map
func (d *dataService) Promotion(codeID int64, types map[string]reflect.Type) (*common.Promotion, error) {
	var promotion common.Promotion
	var promotionType string
	var data []byte
	err := d.db.QueryRow(`
		SELECT promotion_code.code, promo_type, promo_data, promotion_group.name, expires, created
		FROM promotion
		INNER JOIN promotion_code on promotion_code.id = promotion_code_id
		INNER JOIN promotion_group on promotion_group.id = promotion_group_id
		WHERE promotion_code_id = ?`, codeID).Scan(
		&promotion.Code,
		&promotionType,
		&data,
		&promotion.Group,
		&promotion.Expires,
		&promotion.Created)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("promotion"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	promotionDataType, ok := types[promotionType]
	if !ok {
		return nil, errors.Trace(fmt.Errorf("Unable to find promotion type: %s", promotionType))
	}

	promotion.Data = reflect.New(promotionDataType).Interface().(common.Typed)
	if err := json.Unmarshal(data, &promotion.Data); err != nil {
		return nil, errors.Trace(err)
	}

	return &promotion, nil
}

// UpdatePromotion applied the provided update to the promotin record matching the provided promotion_code_id and returns the count of rows affected
func (d *dataService) UpdatePromotion(pu *common.PromotionUpdate) (int64, error) {
	varArgs := dbutil.MySQLVarArgs()
	varArgs.Append(`expires`, pu.Expires)
	res, err := d.db.Exec(`UPDATE promotion SET `+varArgs.Columns()+` WHERE promotion_code_id = ?`, append(varArgs.Values(), pu.CodeID)...)
	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// Promotions returns the promotion records matching the provided promotion_code IDs and maps to the provided type map
func (d *dataService) Promotions(codeIDs []int64, promoTypes []string, types map[string]reflect.Type) ([]*common.Promotion, error) {
	q := `
		SELECT promotion_code.code, promotion_code.id, promo_type, promo_data, promotion_group.name, expires, created
		FROM promotion
		INNER JOIN promotion_code on promotion_code.id = promotion_code_id
		INNER JOIN promotion_group on promotion_group.id = promotion_group_id`
	var vs []interface{}
	if len(codeIDs) > 0 || len(promoTypes) > 0 {
		q += ` WHERE`
	}
	if len(codeIDs) > 0 {
		q += ` promotion_code_id IN (` + dbutil.MySQLArgs(len(codeIDs)) + `)`
		vs = dbutil.AppendInt64sToInterfaceSlice(vs, codeIDs)
	}
	if len(promoTypes) > 0 {
		if len(codeIDs) > 0 {
			q += ` AND `
		}
		q += ` promo_type IN (` + dbutil.MySQLArgs(len(promoTypes)) + `)`
		vs = dbutil.AppendStringsToInterfaceSlice(vs, promoTypes)
	}

	q += ` ORDER BY created DESC LIMIT 100`
	rows, err := d.db.Query(q, vs...)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("promotion"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var promotions []*common.Promotion
	for rows.Next() {
		var data []byte
		var promotionType string
		promotion := &common.Promotion{}
		if err := rows.Scan(
			&promotion.Code,
			&promotion.CodeID,
			&promotionType,
			&data,
			&promotion.Group,
			&promotion.Expires,
			&promotion.Created); err != nil {
			return nil, errors.Trace(err)
		}

		promotionDataType, ok := types[promotionType]
		if !ok {
			return nil, errors.Trace(fmt.Errorf("Unable to find promotion type: %s", promotionType))
		}

		promotion.Data = reflect.New(promotionDataType).Interface().(common.Typed)
		if err := json.Unmarshal(data, &promotion.Data); err != nil {
			return nil, errors.Trace(err)
		}

		promotions = append(promotions, promotion)
	}

	return promotions, errors.Trace(rows.Err())
}

// CreateReferralProgramTemplate inserts a new referral_program_template record that matched the dynamic aspects of the provided template and returns the ID of the record
func (d *dataService) CreateReferralProgramTemplate(template *common.ReferralProgramTemplate) (int64, error) {
	jsonData, err := json.Marshal(template.Data)
	if err != nil {
		return 0, errors.Trace(err)
	}

	tx, err := d.db.Begin()
	if err != nil {
		return 0, errors.Trace(err)
	}

	res, err := tx.Exec(`
		INSERT INTO referral_program_template (role_type_id, referral_type, referral_data, status, promotion_code_id)
			VALUES (?,?,?,?,?)
		`, d.roleTypeMapping[template.Role], template.Data.TypeName(), jsonData, template.Status.String(), template.PromotionCodeID)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	template.ID, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	return template.ID, errors.Trace(tx.Commit())
}

// ReferralProgramTemplate returns the referral_program_template records matching the provided ID
func (d *dataService) ReferralProgramTemplate(id int64, types map[string]reflect.Type) (*common.ReferralProgramTemplate, error) {
	var data []byte
	var referralType string
	template := &common.ReferralProgramTemplate{}
	if err := d.db.QueryRow(`
		SELECT id, role_type_id, referral_type, referral_data, status, promotion_code_id, created
			FROM referral_program_template
			WHERE id = ?`, id).Scan(
		&template.ID,
		&template.RoleTypeID,
		&referralType,
		&data,
		&template.Status,
		&template.PromotionCodeID,
		&template.Created); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound(`referral_program_template`))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	referralDataType, ok := types[referralType]
	if !ok {
		return nil, errors.Trace(fmt.Errorf("Unable to find referral type: %s", referralType))
	}

	template.Data = reflect.New(referralDataType).Interface().(common.Typed)
	err := json.Unmarshal(data, &template.Data)

	return template, errors.Trace(err)
}

// ReferralProgramTemplates returns the referral_program_template records matching the provided ReferralProgramStatuses
func (d *dataService) ReferralProgramTemplates(statuses common.ReferralProgramStatusList, types map[string]reflect.Type) ([]*common.ReferralProgramTemplate, error) {
	rows, err := d.db.Query(`
		SELECT id, role_type_id, referral_type, referral_data, status, promotion_code_id, created
			FROM referral_program_template
			WHERE status IN (`+dbutil.MySQLArgs(len(statuses))+`)
			ORDER BY id DESC`, dbutil.AppendStringsToInterfaceSlice(nil, []string(statuses))...)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound(`referral_program_template`))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var templates []*common.ReferralProgramTemplate
	for rows.Next() {
		template := &common.ReferralProgramTemplate{}
		var referralType string
		var data []byte
		if err := rows.Scan(
			&template.ID,
			&template.RoleTypeID,
			&referralType,
			&data,
			&template.Status,
			&template.PromotionCodeID,
			&template.Created); err != nil {
			return nil, errors.Trace(err)
		}

		referralDataType, ok := types[referralType]
		if !ok {
			return nil, errors.Trace(fmt.Errorf("Unable to find referral type: %s", referralType))
		}

		template.Data = reflect.New(referralDataType).Interface().(common.Typed)
		if err := json.Unmarshal(data, &template.Data); err != nil {
			return nil, errors.Trace(err)
		}

		templates = append(templates, template)
	}

	return templates, errors.Trace(rows.Err())
}

// DefaultReferralProgramTemplate returns the referral_program_template record that maps to the common.RSDefault status
func (d *dataService) DefaultReferralProgramTemplate(types map[string]reflect.Type) (*common.ReferralProgramTemplate, error) {
	var template common.ReferralProgramTemplate
	var referralType string
	var data []byte
	err := d.db.QueryRow(`
		SELECT id, role_type_id, referral_type, referral_data, status, promotion_code_id, created
			FROM referral_program_template
			WHERE status = ?`, common.RSDefault.String()).Scan(
		&template.ID,
		&template.RoleTypeID,
		&referralType,
		&data,
		&template.Status,
		&template.PromotionCodeID,
		&template.Created)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("referral_program_template"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	referralDataType, ok := types[referralType]
	if !ok {
		return nil, errors.Trace(fmt.Errorf("Unable to find referral type: %s", referralType))
	}

	template.Data = reflect.New(referralDataType).Interface().(common.Typed)
	if err := json.Unmarshal(data, &template.Data); err != nil {
		return nil, errors.Trace(err)
	}

	return &template, nil
}

// SetDefaultReferralProgramTemplate declares a the referral_program_template matching the provided ID as DEFAULT
// This will move the existing DEFAULT template to the ACTIVE
// This is all performed within the context of a transaction
func (d *dataService) SetDefaultReferralProgramTemplate(id int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	// Get the old default so we can inactivate it
	oldDefaultTemplate, err := d.DefaultReferralProgramTemplate(common.PromotionTypes)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	// Move the indicated template to the DEFAULT state
	if aff, err := d.updateReferralProgramTemplate(tx, &common.ReferralProgramTemplateUpdate{
		ID:     id,
		Status: common.RSDefault,
	}); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	} else if aff == 0 {
		tx.Rollback()
		return errors.Trace(ErrNotFound(`referral_program_template`))
	}

	// Move the old template to the Active state
	if aff, err := d.updateReferralProgramTemplate(tx, &common.ReferralProgramTemplateUpdate{
		ID:     oldDefaultTemplate.ID,
		Status: common.RSActive,
	}); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	} else if aff == 0 {
		tx.Rollback()
		return errors.Trace(errors.New(`Old default referral_program_template was not updated`))
	}

	return errors.Trace(tx.Commit())
}

// InactivateReferralProgramTemplate moves the referral_program_template to the INACTIVE state and move any associated referral_program records to the INACTIVE state
func (d *dataService) InactivateReferralProgramTemplate(id int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	// Move the old template to the Active state
	if aff, err := d.updateReferralProgramTemplate(tx, &common.ReferralProgramTemplateUpdate{
		ID:     id,
		Status: common.RSInactive,
	}); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	} else if aff == 0 {
		tx.Rollback()
		return errors.Trace(ErrNotFound(`referral_program_template`))
	}

	varArgs := dbutil.MySQLVarArgs()
	varArgs.Append(`status`, common.RSInactive.String())
	if _, err := tx.Exec(`UPDATE referral_program SET `+varArgs.Columns()+` WHERE referral_program_template_id = ?`, append(varArgs.Values(), id)...); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	return errors.Trace(tx.Commit())
}

// UpdateReferralProgramTemplate updates the referral_program_template to the state indicated in the provided update structure and matching the structure's ID and returns the number of rows affected
func (d *dataService) UpdateReferralProgramTemplate(rpt *common.ReferralProgramTemplateUpdate) (int64, error) {
	return d.updateReferralProgramTemplate(d.db, rpt)
}

func (d *dataService) updateReferralProgramTemplate(db db, rpt *common.ReferralProgramTemplateUpdate) (int64, error) {
	varArgs := dbutil.MySQLVarArgs()
	varArgs.Append(`status`, rpt.Status.String())
	res, err := db.Exec(`UPDATE referral_program_template SET `+varArgs.Columns()+` WHERE id = ?`, append(varArgs.Values(), rpt.ID)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	n, err := res.RowsAffected()
	return n, errors.Trace(err)
}

// ReferralProgram returns the referral_program record matching the provided promotion_code ID and maps to the provided type map
func (d *dataService) ReferralProgram(codeID int64, types map[string]reflect.Type) (*common.ReferralProgram, error) {
	var referralProgram common.ReferralProgram
	var referralType string
	var referralData []byte
	if err := d.db.QueryRow(`
		SELECT referral_program_template_id, account_id, promotion_code_id, code, referral_type, referral_data, created, status, promotion_referral_route_id
		FROM referral_program
		INNER JOIN promotion_code on promotion_code.id = promotion_code_id
		WHERE promotion_code_id = ?`, codeID).Scan(
		&referralProgram.TemplateID,
		&referralProgram.AccountID,
		&referralProgram.CodeID,
		&referralProgram.Code,
		&referralType,
		&referralData,
		&referralProgram.Created,
		&referralProgram.Status,
		&referralProgram.PromotionReferralRouteID); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("referral_program"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	referralDataType, ok := types[referralType]
	if !ok {
		return nil, errors.Trace(fmt.Errorf("Unable to find referral type: %s", referralType))
	}

	referralProgram.Data = reflect.New(referralDataType).Interface().(common.Typed)
	if err := json.Unmarshal(referralData, &referralProgram.Data); err != nil {
		return nil, errors.Trace(err)
	}

	return &referralProgram, nil
}

// ActiveReferralProgramForAccount returns the referral_program record matching the provided account ID and maps to the provided type map and common.RSActive status
func (d *dataService) ActiveReferralProgramForAccount(accountID int64, types map[string]reflect.Type) (*common.ReferralProgram, error) {
	var referralProgram common.ReferralProgram
	var referralType string
	var referralData []byte
	if err := d.db.QueryRow(
		`SELECT referral_program_template_id, account_id, promotion_code_id, code, referral_type, referral_data, created, status, promotion_referral_route_id
		FROM referral_program
		INNER JOIN promotion_code on promotion_code.id = promotion_code_id
		WHERE account_id = ? AND status = ?`, accountID, common.RSActive.String()).Scan(
		&referralProgram.TemplateID,
		&referralProgram.AccountID,
		&referralProgram.CodeID,
		&referralProgram.Code,
		&referralType,
		&referralData,
		&referralProgram.Created,
		&referralProgram.Status,
		&referralProgram.PromotionReferralRouteID); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("referral_program"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	referralDataType, ok := types[referralType]
	if !ok {
		return nil, errors.Trace(fmt.Errorf("Unable to find referral type: %s", referralType))
	}

	referralProgram.Data = reflect.New(referralDataType).Interface().(common.Typed)
	if err := json.Unmarshal(referralData, &referralProgram.Data); err != nil {
		return nil, errors.Trace(err)
	}

	return &referralProgram, nil
}

// PendingPromotionsForAccount returns the promotion record matching the provided account ID and maps to the provided type map and common.PSPending status
func (d *dataService) PendingPromotionsForAccount(accountID int64, types map[string]reflect.Type) ([]*common.AccountPromotion, error) {
	rows, err := d.db.Query(`
			SELECT promotion_code.code, account_id, promotion_code_id, promotion_group_id, promo_type, promo_data, expires, created, status
			FROM account_promotion
			JOIN promotion_code ON promotion_code_id = promotion_code.id
			WHERE account_id = ?
			AND status = ?
			ORDER BY created ASC`, accountID, common.PSPending.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var pendingPromotions []*common.AccountPromotion
	for rows.Next() {
		var promotion common.AccountPromotion
		var promotionType string
		var data sql.RawBytes

		if err := rows.Scan(
			&promotion.Code,
			&promotion.AccountID,
			&promotion.CodeID,
			&promotion.GroupID,
			&promotionType,
			&data,
			&promotion.Expires,
			&promotion.Created,
			&promotion.Status); err != nil {
			return nil, errors.Trace(err)
		}

		promotionDataType, ok := types[promotionType]
		if !ok {
			return nil, errors.Trace(fmt.Errorf("Unable to find promotion type: %s", promotionType))
		}

		promotion.Data = reflect.New(promotionDataType).Interface().(common.Typed)
		if err := json.Unmarshal(data, &promotion.Data); err != nil {
			return nil, errors.Trace(err)
		}

		pendingPromotions = append(pendingPromotions, &promotion)
	}

	return pendingPromotions, errors.Trace(rows.Err())
}

// DeleteAccountPromotion updates the account_promotion record matching the provided account ID and promotion_code ID to the status of common.PSDeleted and returns the number of rows affected
func (d *dataService) DeleteAccountPromotion(accountID, promotionCodeID int64) (int64, error) {
	res, err := d.db.Exec(`UPDATE account_promotion SET status = ? WHERE account_id = ? AND promotion_code_id = ?`, common.PSDeleted.String(), accountID, promotionCodeID)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return res.RowsAffected()
}

// CreateReferralProgram inserts a referral_program record matching the dynamic aspects of the provided referral_program
// Notes:
// 1. Any referral_program record with the same account_id as the provided record will be updated to the common.RSInactive status
// 2. A new promotion_code record will be inserted matching the promotion_code.Code in the provided referral_program
// Notes:
// All of the described functionality is performed within a single transaction and any failure will result in a rollback
func (d *dataService) CreateReferralProgram(referralProgram *common.ReferralProgram) error {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	// make any other referral programs for this particular accountID inactive
	_, err = tx.Exec(`UPDATE referral_program SET status = ? WHERE account_id = ? and status = ? `, common.RSInactive.String(), referralProgram.AccountID, common.RSActive.String())
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	// create the promotion code
	res, err := tx.Exec(`INSERT INTO promotion_code (code, is_referral) values (?,?)`, referralProgram.Code, true)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	referralProgram.CodeID, err = res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	jsonData, err := json.Marshal(referralProgram.Data)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	_, err = tx.Exec(`INSERT INTO referral_program (referral_program_template_id, account_id, promotion_code_id, referral_type, referral_data, status, promotion_referral_route_id)
		VALUES (?,?,?,?,?,?,?)`, referralProgram.TemplateID, referralProgram.AccountID, referralProgram.CodeID, referralProgram.Data.TypeName(), jsonData, referralProgram.Status.String(), referralProgram.PromotionReferralRouteID)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	return errors.Trace(tx.Commit())
}

// UpdateReferralProgramStatusesForRoute updates all referral_program records for the provided promotion_referral_route ID to the provided status and returns the number of rows affected
func (d *dataService) UpdateReferralProgramStatusesForRoute(routeID int64, newStatus common.ReferralProgramStatus) (int64, error) {
	res, err := d.db.Exec(`
		UPDATE referral_program
		SET status = ?
		WHERE promotion_referral_route_id = ?`, newStatus.String(), routeID)
	if err != nil {
		return 0, errors.Trace(err)
	}

	return res.RowsAffected()
}

// UpdateReferralProgram updates the referral_program record matching the provided account ID and promotion_code ID with the provided common.Typed data. (This populates the referral_data field of the record)
func (d *dataService) UpdateReferralProgram(accountID int64, codeID int64, data common.Typed) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = d.db.Exec(`
		UPDATE referral_program
		SET referral_data = ?
		WHERE account_id = ? and promotion_code_id = ?`, jsonData, accountID, codeID)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func createPromotion(tx *sql.Tx, promotion *common.Promotion) (int64, error) {
	// create promotion code entry
	res, err := tx.Exec(`INSERT INTO promotion_code (code, is_referral) values (?,?)`, promotion.Code, false)
	if err != nil {
		return 0, errors.Trace(err)
	}

	promotion.CodeID, err = res.LastInsertId()
	if err != nil {
		return 0, errors.Trace(err)
	}

	// get the promotionGroupID
	var promotionGroupID int64
	err = tx.QueryRow(`SELECT id from promotion_group where name = ?`, promotion.Group).Scan(&promotionGroupID)
	if err == sql.ErrNoRows {
		return 0, errors.Trace(ErrNotFound("promotion_group"))
	} else if err != nil {
		return 0, errors.Trace(err)
	}

	// encode the data
	jsonData, err := json.Marshal(promotion.Data)
	if err != nil {
		return 0, errors.Trace(err)
	}

	// create the promotion
	_, err = tx.Exec(`
		INSERT INTO promotion (promotion_code_id, promo_type, promo_data, promotion_group_id, expires)
		VALUES (?,?,?,?,?)`, promotion.CodeID, promotion.Data.TypeName(), jsonData, promotionGroupID, promotion.Expires)
	if err != nil {
		return 0, errors.Trace(err)
	}

	return promotion.CodeID, nil
}

// CreateAccountPromotion inserts an account_promotion record matching the dynamic aspects of the account_promotion provided
// Notes:
// If a promotion_code ID is not provided then one is look up by attempting to match the provided account_promotion.Code field with the promotion_code table
// If an account_promotion_group ID is not provided then one is look up by attempting to match the provided account_promotion.Group field with the account_promotion_group table
// The provided account_promotion.Data field is marshalled to JSON before insertion
func (d *dataService) CreateAccountPromotion(accountPromotion *common.AccountPromotion) error {
	// lookup code based on id
	if accountPromotion.CodeID == 0 {
		if err := d.db.QueryRow(`SELECT id from promotion_code where code = ?`, accountPromotion.Code).
			Scan(&accountPromotion.CodeID); err == sql.ErrNoRows {
			return errors.Trace(errPromoCodeDoesNotExist)
		} else if err != nil {
			return errors.Trace(err)
		}
	}

	if accountPromotion.GroupID == 0 {
		if err := d.db.QueryRow(`SELECT id from promotion_group where name = ?`, accountPromotion.Group).
			Scan(&accountPromotion.GroupID); err == sql.ErrNoRows {
			return errors.Trace(ErrNotFound("promotion_group"))
		} else if err != nil {
			return errors.Trace(err)
		}
	}

	jsonData, err := json.Marshal(accountPromotion.Data)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = d.db.Exec(`
		INSERT INTO account_promotion (account_id, promotion_code_id, promotion_group_id, promo_type, promo_data, expires, status)
		VALUES (?,?,?,?,?,?,?)`, accountPromotion.AccountID,
		accountPromotion.CodeID, accountPromotion.GroupID, accountPromotion.Data.TypeName(),
		jsonData, accountPromotion.Expires, accountPromotion.Status.String())

	return errors.Trace(err)
}

// UpdateAccountPromotion updates the account_promotion record matching the provided with the account ID and promotion code ID to the state represented in the provided AccountPromotionUpdate
func (d *dataService) UpdateAccountPromotion(accountID, promoCodeID int64, update *AccountPromotionUpdate, apo AccountPromotionOption) error {
	if update == nil {
		return nil
	}

	args := dbutil.MySQLVarArgs()
	if update.PromotionData != nil {
		jsonData, err := json.Marshal(update.PromotionData)
		if err != nil {
			return errors.Trace(err)
		}
		args.Append("promo_data", jsonData)
	}
	if update.Status != nil {
		args.Append("status", update.Status.String())
	}
	if args.IsEmpty() {
		return nil
	}
	vals := []interface{}{accountID, promoCodeID}
	q := `UPDATE account_promotion SET ` + args.Columns() + ` WHERE account_id = ? AND promotion_code_id = ?`
	if apo.Has(APOPendingOnly) {
		q += ` AND status = ?`
		vals = append(vals, common.PSPending.String())
	}

	_, err := d.db.Exec(q, append(args.Values(), vals...)...)
	return errors.Trace(err)
}

// UpdateCredit updates the account_credit record matching the provided account ID to include the new credit delta
// Notes:
// The credit changes are also audited into the account_credit_history table
// The described functionality is performed withing the context of a transaction and rolledback on failure
func (d *dataService) UpdateCredit(accountID int64, credit int, description string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	var accountCredit int
	if err := tx.QueryRow(`SELECT credit FROM account_credit WHERE account_id = ? FOR UPDATE`, accountID).
		Scan(&accountCredit); err != sql.ErrNoRows && err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	accountCredit += credit

	// add to credit history
	res, err := tx.Exec(`
		INSERT INTO account_credit_history (account_id, credit, description)
		VALUES (?,?,?)`, accountID, credit, description)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	creditHistoryID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	_, err = tx.Exec(`
		INSERT INTO account_credit (account_id, credit, last_checked_account_credit_history_id)
		VALUES (?,?,?)
		ON DUPLICATE KEY UPDATE credit = ?,last_checked_account_credit_history_id=? `, accountID, accountCredit, creditHistoryID, accountCredit, creditHistoryID)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	return tx.Commit()
}

// AccountCredit returns the account_credit record matching the provided account ID
func (d *dataService) AccountCredit(accountID int64) (*common.AccountCredit, error) {
	var accountCredit common.AccountCredit
	err := d.db.QueryRow(`SELECT account_id, credit FROM account_credit WHERE account_id = ?`, accountID).
		Scan(&accountCredit.AccountID, &accountCredit.Credit)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("account_credit"))
	}

	return &accountCredit, nil
}

// PendingReferralTrackingForAccount returns the account_referral_tracking matching the provided account_id and the common.RTSPending state
func (d *dataService) PendingReferralTrackingForAccount(accountID int64) (*common.ReferralTrackingEntry, error) {
	var entry common.ReferralTrackingEntry

	if err := d.db.QueryRow(`
		SELECT promotion_code_id, claiming_account_id, referring_account_id, created, status
		FROM account_referral_tracking
		WHERE status = ? and claiming_account_id = ?`, common.RTSPending.String(), accountID).Scan(
		&entry.CodeID,
		&entry.ClaimingAccountID,
		&entry.ReferringAccountID,
		&entry.Created,
		&entry.Status); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("account_referral_tracking"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &entry, nil
}

// TrackAccountReferral replaces into the account_referral_tracking table matching the dynamic aspects of the provided account_referral_tracking
func (d *dataService) TrackAccountReferral(referralTracking *common.ReferralTrackingEntry) error {
	_, err := d.db.Exec(`
		REPLACE INTO account_referral_tracking
		(promotion_code_id, claiming_account_id, referring_account_id, status)
		VALUES (?,?,?,?)`, referralTracking.CodeID, referralTracking.ClaimingAccountID, referralTracking.ReferringAccountID, referralTracking.Status.String())
	return errors.Trace(err)
}

// UpdateAccountReferral updated the account_referral_tracking record matching the provided account ID to the provided status
func (d *dataService) UpdateAccountReferral(accountID int64, status common.ReferralTrackingStatus) error {
	_, err := d.db.Exec(`UPDATE account_referral_tracking SET status = ? WHERE claiming_account_id = ?`, status.String(), accountID)
	return errors.Trace(err)
}

// CreateParkedAccount insert ignores a parked_account record matching the dynamic aspects of the provided parked_account and returns the ID of the record
func (d *dataService) CreateParkedAccount(parkedAccount *common.ParkedAccount) (int64, error) {
	parkedAccount.Email = normalizeEmail(parkedAccount.Email)
	res, err := d.db.Exec(`INSERT IGNORE INTO parked_account (email, state, promotion_code_id, account_created) VALUES (?,?,?,?)`,
		parkedAccount.Email, parkedAccount.State, parkedAccount.CodeID, parkedAccount.AccountCreated)
	if err != nil {
		return 0, errors.Trace(err)
	}
	parkedAccount.ID, err = res.LastInsertId()
	return parkedAccount.ID, errors.Trace(err)
}

// ParkedAccount returns the parked_account record matching the provided email
func (d *dataService) ParkedAccount(email string) (*common.ParkedAccount, error) {
	var parkedAccount common.ParkedAccount
	if err := d.db.QueryRow(`
		SELECT parked_account.id, email, state, promotion_code_id, code, is_referral, account_created
		FROM parked_account
		INNER JOIN promotion_code on promotion_code.id = promotion_code_id
		WHERE email = ?`, email).Scan(
		&parkedAccount.ID,
		&parkedAccount.Email,
		&parkedAccount.State,
		&parkedAccount.CodeID,
		&parkedAccount.Code,
		&parkedAccount.IsReferral,
		&parkedAccount.AccountCreated,
	); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("parked_account"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &parkedAccount, nil
}

// MarkParkedAccountAsAccountCreated updates the account_created field of the parked_account record matching the provided ID to true
func (d *dataService) MarkParkedAccountAsAccountCreated(id int64) error {
	_, err := d.db.Exec(`UPDATE parked_account set account_created = 1 WHERE id = ?`, id)
	return errors.Trace(err)
}

// AccountCode returns the account_code associated with the indicated account. This may be nil as it is a nullable field. This indicates that one has never been associated
func (d *dataService) AccountCode(accountID int64) (*uint64, error) {
	var code *uint64
	if err := d.db.QueryRow("SELECT account_code FROM account WHERE id = ?", accountID).Scan(&code); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Trace(ErrNotFound("account"))
		}
		return nil, errors.Trace(err)
	}
	return code, nil
}

// AccountForAccountCode returns the account associated with the indicated promo code
func (d *dataService) AccountForAccountCode(accountCode uint64) (*common.Account, error) {
	account := &common.Account{}
	if err := d.db.QueryRow(`
		SELECT account.id, role_type_tag, email, registration_date, two_factor_enabled, account_code
			FROM account
			INNER JOIN role_type ON role_type_id = role_type.id
			WHERE account.account_code = ?`, accountCode,
	).Scan(&account.ID, &account.Role, &account.Email, &account.Registered, &account.TwoFactorEnabled, &account.AccountCode); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("account"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return account, nil
}

// AssociateRandomAccountCode generates a random account code and updates the indicated account record with the new code and returns the new code
// This will return an error if an account code already exists
func (d *dataService) AssociateRandomAccountCode(accountID int64) (uint64, error) {
	// guard against changing an already associated account code
	if code, err := d.AccountCode(accountID); err != nil {
		return 0, errors.Trace(err)
	} else if code != nil {
		return 0, errors.Trace(fmt.Errorf("Cannot generate and associate a promo code for an account that already has one associated. Account ID: %d - Existing Account Code: %d", accountID, *code))
	}

	// Our account code will be a 7 digit number to give us a large collision free space but also hopefully not annoyingly long
	randRange := int64(accountCodeUpperBound - accountCodeLowerBound)
	var code uint64
	var id int64
	var err error
	for attempts := 0; ; attempts++ {

		// Determine if there is a collision by looking up the account associated with the account code. If there is no collision then associate the code
		code = uint64(accountCodeLowerBound + rand.Int63n(randRange))
		if err = d.db.QueryRow(`SELECT id FROM account WHERE account_code = ?`, code).Scan(&id); err == sql.ErrNoRows {
			if _, err := d.db.Exec(`UPDATE account SET account_code = ? WHERE id = ?`, code, accountID); err != nil {
				return 0, errors.Trace(err)
			}

			break
		} else if err != nil {
			return 0, errors.Trace(err)
		}

		d.accoundCodeCollisionsCounter.Inc(1)
		if attempts >= maxAccountCodeGenerationAttempts {
			errMsg := fmt.Sprintf("Unable to generate unique account code after %d attempts", attempts)
			golog.Errorf(errMsg)
			return 0, errors.Trace(errors.New(errMsg))
		}
	}
	return code, nil
}

// InsertPromotionReferralRoute inserts a promotion_referral_route record matching the dynamic aspects of the provided route and returns the ID of the record
func (d *dataService) InsertPromotionReferralRoute(route *common.PromotionReferralRoute) (int64, error) {
	res, err := d.db.Exec(`
		INSERT INTO promotion_referral_route
		(promotion_code_id, priority, lifecycle, gender, age_lower, age_upper, state, pharmacy)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		route.PromotionCodeID,
		route.Priority,
		route.Lifecycle.String(),
		route.Gender.String(),
		route.AgeLower,
		route.AgeUpper,
		route.State,
		route.Pharmacy)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return res.LastInsertId()
}

// UpdatePromotionReferralRoute updates the corresponding route record and returns the count of affected rows
func (d *dataService) UpdatePromotionReferralRoute(routeUpdate *common.PromotionReferralRouteUpdate) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, errors.Trace(err)
	}

	res, err := d.db.Exec(`UPDATE promotion_referral_route SET lifecycle = ? WHERE id = ?`, routeUpdate.Lifecycle.String(), routeUpdate.ID)
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	// If we are moving this route to a deprecated lifecycle then we need to make all associated referral programs inactive
	if routeUpdate.Lifecycle == common.PRRLifecycleDeprecated {
		if _, err := d.UpdateReferralProgramStatusesForRoute(routeUpdate.ID, common.RSInactive); err != nil {
			tx.Rollback()
			return 0, errors.Trace(err)
		}
	}

	aff, err := res.RowsAffected()
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Trace(err)
	}

	return aff, nil
}

// PromotionReferralRoutes returns all promotion_referral_route records within the provided lifecycle slice
func (d *dataService) PromotionReferralRoutes(lifecycles []string) ([]*common.PromotionReferralRoute, error) {
	rows, err := d.db.Query(`
		SELECT id, promotion_code_id, created, modified, priority, lifecycle, gender, age_lower, age_upper, state, pharmacy
		FROM promotion_referral_route
		WHERE lifecycle IN (`+dbutil.MySQLArgs(len(lifecycles))+`)
		ORDER BY priority DESC`, dbutil.AppendStringsToInterfaceSlice(nil, lifecycles)...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var routes []*common.PromotionReferralRoute
	for rows.Next() {
		route := &common.PromotionReferralRoute{}
		if err := rows.Scan(
			&route.ID,
			&route.PromotionCodeID,
			&route.Created,
			&route.Modified,
			&route.Priority,
			&route.Lifecycle,
			&route.Gender,
			&route.AgeLower,
			&route.AgeUpper,
			&route.State,
			&route.Pharmacy); err != nil {
			return nil, errors.Trace(err)
		}
		routes = append(routes, route)
	}

	return routes, errors.Trace(rows.Err())
}

// RouteQueryParams represents all the data needed to query the promotion_referral_route table
type RouteQueryParams struct {
	Age      *int64
	Gender   *common.PRRGender
	Pharmacy *string
	State    *string
}

// ReferralProgramTemplateRouteQuery utilizes the provided params to find the most relevant referral_program_template and returns that record
func (d *dataService) ReferralProgramTemplateRouteQuery(params *RouteQueryParams) (*int64, *common.ReferralProgramTemplate, error) {
	q := `
		SELECT id, promotion_code_id, priority,
			IF(gender IS NULL, 0, 1) +
			IF(age_lower IS NULL, 0, 1) +
			IF(age_upper IS NULL, 0, 1) +
			IF(state IS NULL, 0, 1) +
			IF(pharmacy IS NULL, 0, 1) AS dim_match_count FROM promotion_referral_route WHERE `
	cs := []string{`lifecycle = ?`}
	vs := []interface{}{string(common.PRRLifecycleActive)}
	if params.Gender != nil {
		cs = append(cs, `(gender IS NULL OR gender = ?)`)
		vs = dbutil.AppendStringsToInterfaceSlice(vs, []string{(*params.Gender).String()})
	} else {
		cs = append(cs, (`gender IS NULL`))
	}
	if params.Age != nil {
		cs = append(cs, `(age_lower IS NULL OR age_lower < ?)`, `(age_upper IS NULL OR age_upper > ?)`)
		vs = dbutil.AppendInt64sToInterfaceSlice(vs, []int64{*params.Age, *params.Age})
	} else {
		cs = append(cs, `age_lower IS NULL AND age_upper IS NULL`)
	}
	if params.Pharmacy != nil {
		cs = append(cs, `(pharmacy IS NULL OR pharmacy = ?)`)
		vs = dbutil.AppendStringsToInterfaceSlice(vs, []string{*params.Pharmacy})
	} else {
		cs = append(cs, `pharmacy IS NULL`)
	}
	if params.State != nil {
		cs = append(cs, `(state IS NULL OR state = ?)`)
		vs = dbutil.AppendStringsToInterfaceSlice(vs, []string{*params.State})
	} else {
		cs = append(cs, `state IS NULL`)
	}
	suffix := ` ORDER BY dim_match_count DESC, priority DESC`
	q = q + strings.Join(cs, " AND ") + suffix

	rows, err := d.db.Query(q, vs...)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	defer rows.Close()

	var prrs []*common.PromotionReferralRoute
	lPriority := math.MinInt64
	lDimMatch := math.MaxInt32
	firstElement := true
	for rows.Next() {
		var dimMatch int
		prr := &common.PromotionReferralRoute{}
		if err := rows.Scan(&prr.ID, &prr.PromotionCodeID, &prr.Priority, &dimMatch); err != nil {
			return nil, nil, errors.Trace(err)

			// Stop building out possible matches if the priority has changed or the dimension match count has changed and it is not the first element
		} else if (prr.Priority != lPriority || lDimMatch != dimMatch) && !firstElement {
			break
		}

		lDimMatch = dimMatch
		lPriority = prr.Priority
		prrs = append(prrs, prr)
		firstElement = false
	}

	if rows.Err() != nil {
		return nil, nil, errors.Trace(err)
	} else if len(prrs) == 0 {
		template, err := d.DefaultReferralProgramTemplate(common.PromotionTypes)
		return nil, template, errors.Trace(err)
	}

	prr := prrs[rand.Int63n(int64(len(prrs)))]
	var data []byte
	var referralType string
	template := &common.ReferralProgramTemplate{}
	err = d.db.QueryRow(`
		SELECT id, role_type_id, referral_type, referral_data, status, promotion_code_id, created
		FROM referral_program_template
		WHERE promotion_code_id = ?
		AND status = ?`, prr.PromotionCodeID, common.RSActive.String()).Scan(
		&template.ID,
		&template.RoleTypeID,
		&referralType,
		&data,
		&template.Status,
		&template.PromotionCodeID,
		&template.Created)
	if err == sql.ErrNoRows {
		defaultTemplate, err := d.DefaultReferralProgramTemplate(common.PromotionTypes)
		if err != nil {
			return nil, nil, errors.Trace(err)
		}
		return nil, defaultTemplate, nil
	} else if err != nil {
		return nil, nil, errors.Trace(err)
	}

	referralDataType, ok := common.PromotionTypes[referralType]
	if !ok {
		return nil, nil, errors.Trace(fmt.Errorf("Unable to find referral type: %s", referralType))
	}

	template.Data = reflect.New(referralDataType).Interface().(common.Typed)
	if err := json.Unmarshal(data, &template.Data); err != nil {
		return nil, nil, errors.Trace(err)
	}

	return &prr.ID, template, nil
}

// RouteQueryParamsForAccount given the provided account creates and returns the param structure needed to route the account to the correct referral program
func (d *dataService) RouteQueryParamsForAccount(accountID int64) (*RouteQueryParams, error) {
	patient, err := d.GetPatientFromAccountID(accountID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	params := &RouteQueryParams{}
	if err := d.db.QueryRow(`
		SELECT FLOOR(DATEDIFF(CURRENT_DATE(), STR_TO_DATE(CONCAT_WS('-',dob_day,dob_month,dob_year), '%d-%m-%Y'))/365) AS age, gender, patient_location.state, pharmacy_selection.name AS pharmacy
			FROM patient
			LEFT JOIN patient_location ON patient.id = patient_location.patient_id
			LEFT JOIN patient_pharmacy_selection ON patient.id = patient_pharmacy_selection.patient_id
			LEFT JOIN pharmacy_selection ON patient_pharmacy_selection.pharmacy_selection_id = pharmacy_selection.id
			WHERE patient.id = ?`, patient.ID.Int64()).Scan(&params.Age, &params.Gender, &params.State, &params.Pharmacy); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound(`patient`))
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	return params, nil
}
