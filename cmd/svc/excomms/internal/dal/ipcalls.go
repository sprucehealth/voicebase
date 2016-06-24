package dal

import (
	"database/sql"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"golang.org/x/net/context"
)

func (d *dal) CreateIPCall(ctx context.Context, call *models.IPCall) error {
	if len(call.Participants) < 2 {
		return errors.Trace(errors.New("IPCall requires at least 2 participants"))
	}
	if call.Type == "" {
		return errors.Trace(errors.New("IPCall type required"))
	}

	var err error
	call.ID, err = models.NewIPCallID()
	if err != nil {
		return errors.Trace(err)
	}
	call.InitiatedTime = d.clk.Now()
	call.Pending = true

	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	_, err = tx.Exec(`INSERT INTO ipcall (id, type, pending, initiated, connected) VALUES (?, ?, ?, ?, ?)`,
		call.ID, call.Type, call.Pending, call.InitiatedTime, call.ConnectedTime)
	if err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}

	for _, p := range call.Participants {
		if p.AccountID == "" {
			return errors.Trace(errors.New("IPCallParticipant account ID required"))
		}
		if p.EntityID == "" {
			return errors.Trace(errors.New("IPCallParticipant entity ID required"))
		}
		if p.Identity == "" {
			return errors.Trace(errors.New("IPCallParticipant identity required"))
		}
		if p.Role == "" {
			return errors.Trace(errors.New("IPCallParticipant role required"))
		}
		if p.State == "" {
			return errors.Trace(errors.New("IPCallParticipant state required"))
		}
		if p.NetworkType == "" {
			return errors.Trace(errors.New("IPCallParticipant network type required"))
		}
		_, err := tx.Exec(`
			INSERT INTO ipcall_participant
				(ipcall_id, account_id, entity_id, identity, role, state, network_type)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, call.ID, p.AccountID, p.EntityID, p.Identity, p.Role, p.State, p.NetworkType)
		if err != nil {
			tx.Rollback()
			return errors.Trace(err)
		}
	}

	return errors.Trace(tx.Commit())
}

func (d *dal) IPCall(ctx context.Context, id models.IPCallID, opts ...QueryOption) (*models.IPCall, error) {
	forUpdate := ""
	if queryOptions(opts).Has(ForUpdate) {
		forUpdate = " FOR UPDATE"
	}

	call := &models.IPCall{ID: models.EmptyIPCallID()}
	row := d.db.QueryRow(`SELECT id, type, pending, initiated, connected FROM ipcall WHERE id = ?`+forUpdate, id)
	if err := row.Scan(&call.ID, &call.Type, &call.Pending, &call.InitiatedTime, &call.ConnectedTime); err == sql.ErrNoRows {
		return nil, errors.Trace(ErrIPCallNotFound)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	rows, err := d.db.Query(`
		SELECT account_id, entity_id, identity, role, state, network_type
		FROM ipcall_participant
		WHERE ipcall_id = ?`+forUpdate, call.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	for rows.Next() {
		cp := &models.IPCallParticipant{}
		if err := rows.Scan(&cp.AccountID, &cp.EntityID, &cp.Identity, &cp.Role, &cp.State, &cp.NetworkType); err != nil {
			return nil, errors.Trace(err)
		}
		call.Participants = append(call.Participants, cp)
	}
	return call, errors.Trace(rows.Err())
}

func (d *dal) PendingIPCallsForAccount(ctx context.Context, accountID string) ([]*models.IPCall, error) {
	rows, err := d.db.Query(`
		SELECT c.id, c.type, c.pending, c.initiated, c.connected
		FROM ipcall_participant cp
		INNER JOIN ipcall c ON c.id = cp.ipcall_id
		WHERE cp.account_id = ? AND pending = ?`, accountID, true)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	var calls []*models.IPCall
	for rows.Next() {
		c := &models.IPCall{ID: models.EmptyIPCallID()}
		if err := rows.Scan(&c.ID, &c.Type, &c.Pending, &c.InitiatedTime, &c.ConnectedTime); err != nil {
			return nil, errors.Trace(err)
		}
		calls = append(calls, c)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Trace(err)
	}

	if len(calls) == 0 {
		return calls, nil
	}

	// Query participants

	callIDs := make([]interface{}, len(calls))
	for i, c := range calls {
		callIDs[i] = c.ID
	}
	rows, err = d.db.Query(`
		SELECT ipcall_id, account_id, entity_id, identity, role, state, network_type
		FROM ipcall_participant
		WHERE ipcall_id IN (`+dbutil.MySQLArgs(len(callIDs))+`)`,
		callIDs...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()
	cID := models.EmptyIPCallID()
	for rows.Next() {
		cp := &models.IPCallParticipant{}
		if err := rows.Scan(&cID, &cp.AccountID, &cp.EntityID, &cp.Identity, &cp.Role, &cp.State, &cp.NetworkType); err != nil {
			return nil, errors.Trace(err)
		}
		// The list of calls should generally only have 1 item so this should be plenty efficient
		for _, c := range calls {
			if c.ID.Val == cID.Val {
				c.Participants = append(c.Participants, cp)
				continue
			}
		}
	}
	return calls, errors.Trace(rows.Err())
}

func (d *dal) UpdateIPCall(ctx context.Context, callID models.IPCallID, update *IPCallUpdate) error {
	set := dbutil.MySQLVarArgs()
	if update.Pending != nil {
		set.Append("pending", *update.Pending)
	}
	if update.ConnectedTime != nil {
		set.Append("connected", *update.ConnectedTime)
	}
	if set.IsEmpty() {
		return nil
	}
	_, err := d.db.Exec(`UPDATE ipcall SET `+set.ColumnsForUpdate()+` WHERE id = ?`, append(set.Values(), callID)...)
	return errors.Trace(err)
}

func (d *dal) UpdateIPCallParticipant(ctx context.Context, callID models.IPCallID, accountID string, update *IPCallParticipantUpdate) error {
	set := dbutil.MySQLVarArgs()
	if update.State != nil {
		set.Append("state", *update.State)
	}
	if update.NetworkType != nil {
		set.Append("network_type", *update.NetworkType)
	}
	if set.IsEmpty() {
		return nil
	}
	_, err := d.db.Exec(`
		UPDATE ipcall_participant
		SET `+set.ColumnsForUpdate()+`
		WHERE ipcall_id = ? AND account_id = ?`, append(set.Values(), callID, accountID)...)
	return errors.Trace(err)
}
