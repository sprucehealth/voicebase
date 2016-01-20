package dal

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	modellib "github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
	"github.com/sprucehealth/backend/svc/notification"
)

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	Transact(trans func(dal DAL) error) (err error)
	InsertPushConfig(model *PushConfig) (PushConfigID, error)
	PushConfig(id PushConfigID) (*PushConfig, error)
	PushConfigForDeviceID(deviceID string) (*PushConfig, error)
	PushConfigsForExternalGroupID(externalGroupID string) ([]*PushConfig, error)
	UpdatePushConfig(id PushConfigID, update *PushConfigUpdate) (int64, error)
	DeletePushConfig(id PushConfigID) (int64, error)
}

type dal struct {
	db tsql.DB
}

// New returns an initialized instance of dal
func New(db *sql.DB) DAL {
	return &dal{db: tsql.AsDB(db)}
}

// Transact encapsulated the provided function in a transaction and handles rollback and commit actions
func (d *dal) Transact(trans func(dal DAL) error) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	tdal := &dal{
		db: tsql.AsSafeTx(tx),
	}
	// Recover from any inner panics that happened and close the transaction
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = errors.Trace(fmt.Errorf("Encountered panic during transaction execution: %v", r))
		}
	}()
	if err := trans(tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return errors.Trace(tx.Commit())
}

// NewPushConfigID returns a new PushConfigID.
func NewPushConfigID() (PushConfigID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return PushConfigID{}, errors.Trace(err)
	}
	return PushConfigID{
		modellib.ObjectID{
			Prefix:  notification.PushConfigIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyPushConfigID returns an empty initialized instance of PushConfigID
func EmptyPushConfigID() PushConfigID {
	return PushConfigID{
		modellib.ObjectID{
			Prefix:  notification.PushConfigIDPrefix,
			IsValid: false,
		},
	}
}

// PushConfigID is the ID for a PushConfigID object
type PushConfigID struct {
	modellib.ObjectID
}

// PushConfig represents a push_config record
type PushConfig struct {
	ID              PushConfigID
	ExternalGroupID string
	Platform        string
	PlatformVersion string
	AppVersion      string
	DeviceID        string
	DeviceToken     []byte
	PushEndpoint    string
	Device          string
	DeviceModel     string
	Modified        time.Time
	Created         time.Time
}

// PushConfigUpdate represents the mutable aspects of a push_config record
type PushConfigUpdate struct {
	DeviceToken     []byte
	PushEndpoint    *string
	ExternalGroupID *string
	Platform        *string
	PlatformVersion *string
	AppVersion      *string
}

// InsertPushConfig inserts a push_config record
func (d *dal) InsertPushConfig(model *PushConfig) (PushConfigID, error) {
	if !model.ID.IsValid {
		id, err := NewPushConfigID()
		if err != nil {
			return PushConfigID{}, errors.Trace(err)
		}
		model.ID = id
	}

	_, err := d.db.Exec(
		`INSERT INTO push_config
          (platform, platform_version, app_version, device_id, id, external_group_id, device, device_model, device_token, push_endpoint)
          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, model.Platform, model.PlatformVersion, model.AppVersion, model.DeviceID, model.ID, model.ExternalGroupID, model.Device, model.DeviceModel, model.DeviceToken, model.PushEndpoint)
	if err != nil {
		return PushConfigID{}, errors.Trace(err)
	}

	return model.ID, nil
}

// PushConfig retrieves a push_config record
func (d *dal) PushConfig(id PushConfigID) (*PushConfig, error) {
	model := &PushConfig{
		ID: EmptyPushConfigID(),
	}
	if err := d.db.QueryRow(
		`SELECT device_model, created, device_token, push_endpoint, device, platform_version, app_version, device_id, modified, id, external_group_id, platform
          FROM push_config
          WHERE id = ?`, id.Val).Scan(&model.DeviceModel, &model.Created, &model.DeviceToken, &model.PushEndpoint, &model.Device, &model.PlatformVersion, &model.AppVersion, &model.DeviceID, &model.Modified, &model.ID, &model.ExternalGroupID, &model.Platform); err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("push_config not found"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return model, nil
}

// PushConfigForDeviceID retrieves a push_config record
func (d *dal) PushConfigForDeviceID(deviceID string) (*PushConfig, error) {
	model := &PushConfig{
		ID: EmptyPushConfigID(),
	}
	if err := d.db.QueryRow(
		`SELECT device_model, created, device_token, push_endpoint, device, platform_version, app_version, device_id, modified, id, external_group_id, platform
          FROM push_config
          WHERE device_id = ?`, deviceID).Scan(&model.DeviceModel, &model.Created, &model.DeviceToken, &model.PushEndpoint, &model.Device, &model.PlatformVersion, &model.AppVersion, &model.DeviceID, &model.Modified, &model.ID, &model.ExternalGroupID, &model.Platform); err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("push_config not found"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return model, nil
}

// PushConfigsForExternalGroupID retrieves the set of push configs that map to the provided external group id
func (d *dal) PushConfigsForExternalGroupID(externalGroupID string) ([]*PushConfig, error) {
	rows, err := d.db.Query(
		`SELECT device_model, created, device_token, push_endpoint, device, platform_version, app_version, device_id, modified, id, external_group_id, platform
          FROM push_config
          WHERE external_group_id = ?`, externalGroupID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*PushConfig
	for rows.Next() {
		model := &PushConfig{
			ID: EmptyPushConfigID(),
		}
		if err := rows.Scan(&model.DeviceModel, &model.Created, &model.DeviceToken, &model.PushEndpoint, &model.Device, &model.PlatformVersion, &model.AppVersion, &model.DeviceID, &model.Modified, &model.ID, &model.ExternalGroupID, &model.Platform); err == sql.ErrNoRows {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}

	return models, errors.Trace(rows.Err())
}

// UpdatePushConfig updates the mutable aspects of a push_config record
func (d *dal) UpdatePushConfig(id PushConfigID, update *PushConfigUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if len(update.DeviceToken) != 0 {
		args.Append("device_token", update.DeviceToken)
	}
	if update.PushEndpoint != nil {
		args.Append("push_endpoint", *update.PushEndpoint)
	}
	if update.ExternalGroupID != nil {
		args.Append("external_group_id", *update.ExternalGroupID)
	}
	if update.Platform != nil {
		args.Append("platform", *update.Platform)
	}
	if update.PlatformVersion != nil {
		args.Append("platform_version", *update.PlatformVersion)
	}
	if update.AppVersion != nil {
		args.Append("app_version", *update.AppVersion)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE push_config
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeletePushConfig deletes a push_config record
func (d *dal) DeletePushConfig(id PushConfigID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM push_config
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}
