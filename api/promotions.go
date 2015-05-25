package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/libs/golog"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
)

const (
	// We will have to deal with collisions in the generation space.
	// We don't want to do this forever so cap our attempts at some reasonable value
	maxAccountCodeGenerationAttempts = 100
	accountCodeUpperBound            = 9999999
	accountCodeLowerBound            = 1000000
)

var (
	errPromoCodeDoesNotExist = errors.New("Promotion code does not exist")
)

// LookupPromoCode returns the promotion_code record of the indicated promo code.
// If the code provided maps to an account_code then the promo code for that account's active referral_program will be returned
func (d *DataService) LookupPromoCode(code string) (*common.PromoCode, error) {
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
			if err != nil {
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

func (d *DataService) PromoCodeForAccountExists(accountID, codeID int64) (bool, error) {
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

func (d *DataService) PromotionCountInGroupForAccount(accountID int64, group string) (int, error) {
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

func (d *DataService) PromoCodePrefixes() ([]string, error) {
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

func (d *DataService) CreatePromoCodePrefix(prefix string) error {
	_, err := d.db.Exec(`INSERT INTO promo_code_prefix (prefix, status) VALUES (?,?)`, prefix, StatusActive)
	return errors.Trace(err)
}

func (d *DataService) CreatePromotionGroup(promotionGroup *common.PromotionGroup) (int64, error) {
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

func (d *DataService) PromotionGroup(name string) (*common.PromotionGroup, error) {
	var promotionGroup common.PromotionGroup
	if err := d.db.QueryRow(`SELECT name, max_allowed_promos FROM promotion_group WHERE name = ?`, name).
		Scan(&promotionGroup.Name, &promotionGroup.MaxAllowedPromos); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("promotion_group"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return &promotionGroup, nil
}

func (d *DataService) CreatePromotion(promotion *common.Promotion) error {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	if err := createPromotion(tx, promotion); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	return tx.Commit()
}

func (d *DataService) Promotion(codeID int64, types map[string]reflect.Type) (*common.Promotion, error) {
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

func (d *DataService) CreateReferralProgramTemplate(template *common.ReferralProgramTemplate) (int64, error) {
	jsonData, err := json.Marshal(template.Data)
	if err != nil {
		return 0, errors.Trace(err)
	}

	tx, err := d.db.Begin()
	if err != nil {
		return 0, errors.Trace(err)
	}

	_, err = tx.Exec(`UPDATE referral_program_template set status = ? where role_type_id = ?`,
		common.RSInactive.String(), d.roleTypeMapping[template.Role])
	if err != nil {
		tx.Rollback()
		return 0, errors.Trace(err)
	}

	res, err := tx.Exec(`
		INSERT INTO referral_program_template (role_type_id, referral_type, referral_data, status)
		VALUES (?,?,?,?)
		`, d.roleTypeMapping[template.Role], template.Data.TypeName(), jsonData, template.Status.String())
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

func (d *DataService) ActiveReferralProgramTemplate(role string, types map[string]reflect.Type) (*common.ReferralProgramTemplate, error) {
	var template common.ReferralProgramTemplate
	var referralType string
	var data []byte
	err := d.db.QueryRow(`
		SELECT id, role_type_id, referral_type, referral_data, status
		FROM referral_program_template
		WHERE role_type_id = ? and status = ?`, d.roleTypeMapping[role], common.RSActive.String()).Scan(
		&template.ID,
		&template.RoleTypeID,
		&referralType,
		&data,
		&template.Status)
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

func (d *DataService) ReferralProgram(codeID int64, types map[string]reflect.Type) (*common.ReferralProgram, error) {
	var referralProgram common.ReferralProgram
	var referralType string
	var referralData []byte
	if err := d.db.QueryRow(`
		SELECT referral_program_template_id, account_id, promotion_code_id, code, referral_type, referral_data, created, status
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
		&referralProgram.Status); err == sql.ErrNoRows {
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

func (d *DataService) ActiveReferralProgramForAccount(accountID int64, types map[string]reflect.Type) (*common.ReferralProgram, error) {
	var referralProgram common.ReferralProgram
	var referralType string
	var referralData []byte
	if err := d.db.QueryRow(
		`SELECT referral_program_template_id, account_id, promotion_code_id, code, referral_type, referral_data, created, status
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
		&referralProgram.Status); err == sql.ErrNoRows {
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

func (d *DataService) PendingPromotionsForAccount(accountID int64, types map[string]reflect.Type) ([]*common.AccountPromotion, error) {
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

func (d *DataService) DeleteAccountPromotion(accountID, promotionCodeID int64) (int64, error) {
	res, err := d.db.Exec(`UPDATE account_promotion SET status = ? WHERE account_id = ? AND promotion_code_id = ?`, common.PSDeleted.String(), accountID, promotionCodeID)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return res.RowsAffected()
}

func (d *DataService) CreateReferralProgram(referralProgram *common.ReferralProgram) error {
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

	_, err = tx.Exec(`INSERT INTO referral_program (referral_program_template_id, account_id, promotion_code_id, referral_type, referral_data, status) 
		VALUES (?,?,?,?,?,?)`, referralProgram.TemplateID, referralProgram.AccountID, referralProgram.CodeID, referralProgram.Data.TypeName(), jsonData, referralProgram.Status.String())
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	return errors.Trace(tx.Commit())
}

func (d *DataService) UpdateReferralProgram(accountID int64, codeID int64, data common.Typed) error {
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

func createPromotion(tx *sql.Tx, promotion *common.Promotion) error {
	// create promotion code entry
	res, err := tx.Exec(`INSERT INTO promotion_code (code, is_referral) values (?,?)`, promotion.Code, false)
	if err != nil {
		return errors.Trace(err)
	}

	promotion.CodeID, err = res.LastInsertId()
	if err != nil {
		return errors.Trace(err)
	}

	// get the promotionGroupID
	var promotionGroupID int64
	err = tx.QueryRow(`SELECT id from promotion_group where name = ?`, promotion.Group).Scan(&promotionGroupID)
	if err == sql.ErrNoRows {
		return errors.Trace(ErrNotFound("promotion_group"))
	} else if err != nil {
		return errors.Trace(err)
	}

	// encode the data
	jsonData, err := json.Marshal(promotion.Data)
	if err != nil {
		return errors.Trace(err)
	}

	// create the promotion
	_, err = tx.Exec(`
		INSERT INTO promotion (promotion_code_id, promo_type, promo_data, promotion_group_id, expires)
		VALUES (?,?,?,?,?)`, promotion.CodeID, promotion.Data.TypeName(), jsonData, promotionGroupID, promotion.Expires)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (d *DataService) CreateAccountPromotion(accountPromotion *common.AccountPromotion) error {
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

func (d *DataService) UpdateAccountPromotion(accountID, promoCodeID int64, update *AccountPromotionUpdate) error {
	if update == nil {
		return nil
	}

	var cols []string
	var vals []interface{}

	if update.PromotionData != nil {
		jsonData, err := json.Marshal(update.PromotionData)
		if err != nil {
			return errors.Trace(err)
		}

		cols = append(cols, "promo_data = ?")
		vals = append(vals, jsonData)
	}

	if update.Status != nil {
		cols = append(cols, "status = ?")
		vals = append(vals, update.Status.String())
	}

	if len(cols) == 0 {
		return nil
	}

	vals = append(vals, accountID, promoCodeID)

	_, err := d.db.Exec(fmt.Sprintf(
		`UPDATE account_promotion SET %s WHERE account_id = ? AND promotion_code_id = ?`,
		strings.Join(cols, ",")), vals...)
	return errors.Trace(err)
}

func (d *DataService) UpdateCredit(accountID int64, credit int, description string) error {
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

func (d *DataService) AccountCredit(accountID int64) (*common.AccountCredit, error) {
	var accountCredit common.AccountCredit
	err := d.db.QueryRow(`SELECT account_id, credit FROM account_credit WHERE account_id = ?`, accountID).
		Scan(&accountCredit.AccountID, &accountCredit.Credit)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound("account_credit"))
	}

	return &accountCredit, nil
}

func (d *DataService) PendingReferralTrackingForAccount(accountID int64) (*common.ReferralTrackingEntry, error) {
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

func (d *DataService) TrackAccountReferral(referralTracking *common.ReferralTrackingEntry) error {
	_, err := d.db.Exec(`
		REPLACE INTO account_referral_tracking
		(promotion_code_id, claiming_account_id, referring_account_id, status)
		VALUES (?,?,?,?)`, referralTracking.CodeID, referralTracking.ClaimingAccountID, referralTracking.ReferringAccountID, referralTracking.Status.String())
	return errors.Trace(err)
}

func (d *DataService) UpdateAccountReferral(accountID int64, status common.ReferralTrackingStatus) error {
	_, err := d.db.Exec(`UPDATE account_referral_tracking SET status = ? WHERE claiming_account_id = ?`, status.String(), accountID)
	return errors.Trace(err)
}

func (d *DataService) CreateParkedAccount(parkedAccount *common.ParkedAccount) (int64, error) {
	parkedAccount.Email = normalizeEmail(parkedAccount.Email)
	res, err := d.db.Exec(`INSERT IGNORE INTO parked_account (email, state, promotion_code_id, account_created) VALUES (?,?,?,?)`,
		parkedAccount.Email, parkedAccount.State, parkedAccount.CodeID, parkedAccount.AccountCreated)
	if err != nil {
		return 0, errors.Trace(err)
	}
	parkedAccount.ID, err = res.LastInsertId()
	return parkedAccount.ID, errors.Trace(err)
}

func (d *DataService) ParkedAccount(email string) (*common.ParkedAccount, error) {
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

func (d *DataService) MarkParkedAccountAsAccountCreated(id int64) error {
	_, err := d.db.Exec(`UPDATE parked_account set account_created = 1 WHERE id = ?`, id)
	return errors.Trace(err)
}

// AccountCode returns the account_code associated with the indicated account. This may be nil as it is a nullable field. This indicates that one has never been associated
func (d *DataService) AccountCode(accountID int64) (*uint64, error) {
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
func (d *DataService) AccountForAccountCode(accountCode uint64) (*common.Account, error) {
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

// AssociateRandomAccountCode generates a random account code  and updates the indicated account record with the new code.
// This will return an error if an account code already exists and persisting that code is important.
func (d *DataService) AssociateRandomAccountCode(accountID int64) (uint64, error) {
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
