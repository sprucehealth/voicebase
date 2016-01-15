package dal

import (
	"database/sql"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	InsertEntity(model *Entity) (EntityID, error)
	Entity(id EntityID) (*Entity, error)
	Entities(ids []EntityID) ([]*Entity, error)
	UpdateEntity(id EntityID, update *EntityUpdate) (int64, error)
	DeleteEntity(id EntityID) (int64, error)
	InsertExternalEntityID(model *ExternalEntityID) error
	ExternalEntityIDs(externalID string) ([]*ExternalEntityID, error)
	ExternalEntityIDsForEntities(entityID []EntityID) ([]*ExternalEntityID, error)
	InsertEntityMembership(model *EntityMembership) error
	EntityMemberships(id EntityID) ([]*EntityMembership, error)
	EntityMembers(id EntityID) ([]*Entity, error)
	InsertEntityContact(model *EntityContact) (EntityContactID, error)
	EntityContact(id EntityContactID) (*EntityContact, error)
	EntityContacts(id EntityID) ([]*EntityContact, error)
	EntityContactsForValue(value string) ([]*EntityContact, error)
	UpdateEntityContact(id EntityContactID, update *EntityContactUpdate) (int64, error)
	DeleteEntityContact(id EntityContactID) (int64, error)
	InsertEvent(model *Event) (EventID, error)
	Event(id EventID) (*Event, error)
	UpdateEvent(id EventID, update *EventUpdate) (int64, error)
	DeleteEvent(id EventID) (int64, error)
	Transact(trans func(dal DAL) error) (err error)
}

type dal struct {
	db tsql.DB
}

// New returns an initialized instance of dal
func New(db *sql.DB) DAL {
	golog.Debugf("Entering dal.New...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.New...") }()
	}
	return &dal{db: tsql.AsDB(db)}
}

// Transact encapsulated the provided function in a transaction and handles rollback and commit actions
func (d *dal) Transact(trans func(dal DAL) error) (err error) {
	golog.Debugf("Entering dal.dal.Transact...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.Transact...") }()
	}
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
			golog.Errorf(string(debug.Stack()))
			err = errors.Trace(fmt.Errorf("Encountered panic during transaction execution: %v", r))
		}
	}()
	if err := trans(tdal); err != nil {
		tx.Rollback()
		return errors.Trace(err)
	}
	return errors.Trace(tx.Commit())
}

// NewEntityContactID returns a new EntityContactID using the provided value. If id is 0
// then the returned EntityContactID is tagged as invalid.
func NewEntityContactID(id uint64) EntityContactID {
	golog.Debugf("Entering dal.NewEntityContactID: %d", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.NewEntityContactID...") }()
	}
	return EntityContactID{
		ObjectID: encoding.ObjectID{
			Uint64Value: id,
			IsValid:     id != 0,
		},
	}
}

// EntityContactID is the ID for a entity_contact object
type EntityContactID struct {
	encoding.ObjectID
}

const (
	contactIDPrefix = "contact"
)

func (e EntityContactID) String() string {
	golog.Debugf("Entering dal.EntityContactID.String...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.EntityContactID.String...") }()
	}
	return fmt.Sprintf("%s:%d", contactIDPrefix, e.Uint64())
}

// ParseEntityContactID transforms the provided id string to a numberic value
func ParseEntityContactID(id string) EntityID {
	golog.Debugf("Entering dal.ParseEntityContactID: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseEntityContactID...") }()
	}
	var entityID EntityID
	seg := strings.Split(id, ":")
	if len(seg) > 1 {
		conc.Go(func() {
			if !strings.EqualFold(seg[len(seg)-2], contactIDPrefix) {
				golog.Errorf("%s was provided as an EntityContactID but does not match prefix %s. Continuing anyway.", id, contactIDPrefix)
			}
		})
		id, err := strconv.ParseInt(seg[len(seg)-1], 10, 64)
		if err == nil {
			entityID = NewEntityID(uint64(id))
		} else {
			golog.Warningf("Error while parsing contact ID: %s", err)
		}
	}
	return entityID
}

// NewEventID returns a new EventID using the provided value. If id is 0
// then the returned EventID is tagged as invalid.
func NewEventID(id uint64) EventID {
	golog.Debugf("Entering dal.NewEventID...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.NewEventID...") }()
	}
	return EventID{
		ObjectID: encoding.ObjectID{
			Uint64Value: id,
			IsValid:     id != 0,
		},
	}
}

// EventID is the ID for a event object
type EventID struct {
	encoding.ObjectID
}

// NewEntityID returns a new EntityID using the provided value. If id is 0
// then the returned EntityID is tagged as invalid.
func NewEntityID(id uint64) EntityID {
	golog.Debugf("Entering dal.NewEntityID...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.NewEntityID...") }()
	}
	return EntityID{
		ObjectID: encoding.ObjectID{
			Uint64Value: id,
			IsValid:     id != 0,
		},
	}
}

// EntityID is the ID for a entity object
type EntityID struct {
	encoding.ObjectID
}

const (
	entityIDPrefix = "entity"
)

func (e EntityID) String() string {
	golog.Debugf("Entering dal.EntityID.String...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.EntityID.String...") }()
	}
	return fmt.Sprintf("%s:%d", entityIDPrefix, e.Uint64())
}

// ParseEntityID transforms the provided id string to a numberic value
func ParseEntityID(id string) EntityID {
	golog.Debugf("Entering dal.ParseEntityID...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseAccountID...") }()
	}
	var entityID EntityID
	seg := strings.Split(id, ":")
	if len(seg) > 1 {
		conc.Go(func() {
			if !strings.EqualFold(seg[len(seg)-2], entityIDPrefix) {
				golog.Errorf("%s was provided as an EntityID but does not match prefix %s. Continuing anyway.", id, entityIDPrefix)
			}
		})
		id, err := strconv.ParseInt(seg[len(seg)-1], 10, 64)
		if err == nil {
			entityID = NewEntityID(uint64(id))
		} else {
			golog.Warningf("Error while parsing entity ID: %s", err)
		}
	}
	return entityID
}

// EntityType represents the type associated with the type column of the entity table
type EntityType string

const (
	// EntityTypeOrganization represents the ORGANIZATION state of the type field on a entity record
	EntityTypeOrganization EntityType = "ORGANIZATION"
	// EntityTypeInternal represents the INTERNAL state of the type field on a entity record
	EntityTypeInternal EntityType = "INTERNAL"
	// EntityTypeExternal represents the ECTERNAL state of the type field on a entity record
	EntityTypeExternal EntityType = "EXTERNAL"
)

// ParseEntityType converts a string into the correcponding enum value
func ParseEntityType(s string) (EntityType, error) {
	golog.Debugf("Entering dal.ParseEntityType...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseEntityType...") }()
	}
	switch t := EntityType(strings.ToUpper(s)); t {
	case EntityTypeOrganization, EntityTypeInternal, EntityTypeExternal:
		return t, nil
	}
	return EntityType(""), errors.Trace(fmt.Errorf("Unknown type:%s", s))
}

func (t EntityType) String() string {
	golog.Debugf("Entering dal.EntityType.String...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.EntityType.String...") }()
	}
	return string(t)
}

// Scan allows for scanning of EntityType from a database conforming to the sql.Scanner interface
func (t *EntityType) Scan(src interface{}) error {
	golog.Debugf("Entering dal.EntityType.Scan...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.EntityType.Scan...") }()
	}
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseEntityType(ts)
	case []byte:
		*t, err = ParseEntityType(string(ts))
	}
	return errors.Trace(err)
}

// EntityStatus represents the type associated with the status column of the entity table
type EntityStatus string

const (
	// EntityStatusActive represents the ACTIVE state of the status field on a entity record
	EntityStatusActive EntityStatus = "ACTIVE"
	// EntityStatusDeleted represents the DELETED state of the status field on a entity record
	EntityStatusDeleted EntityStatus = "DELETED"
	// EntityStatusSuspended represents the SUSPENDED state of the status field on a entity record
	EntityStatusSuspended EntityStatus = "SUSPENDED"
)

// ParseEntityStatus converts a string into the correcponding enum value
func ParseEntityStatus(s string) (EntityStatus, error) {
	golog.Debugf("Entering dal.ParseEntityStatus...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseEntityStatus...") }()
	}
	switch t := EntityStatus(strings.ToUpper(s)); t {
	case EntityStatusActive, EntityStatusDeleted, EntityStatusSuspended:
		return t, nil
	}
	return EntityStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t EntityStatus) String() string {
	golog.Debugf("Entering dal.ParseEntityStatus.String...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseEntityStatus.String...") }()
	}
	return string(t)
}

// Scan allows for scanning of EntityStatus from a database conforming to the sql.Scanner interface
func (t *EntityStatus) Scan(src interface{}) error {
	golog.Debugf("Entering dal.ParseEntityStatus.Scan...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseEntityStatus.Scan...") }()
	}
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseEntityStatus(ts)
	case []byte:
		*t, err = ParseEntityStatus(string(ts))
	}
	return errors.Trace(err)
}

// EntityMembershipStatus represents the type associated with the status column of the entity_membership table
type EntityMembershipStatus string

const (
	// EntityMembershipStatusActive represents the ACTIVE state of the status field on a entity_membership record
	EntityMembershipStatusActive EntityMembershipStatus = "ACTIVE"
	// EntityMembershipStatusDeleted represents the DELETED state of the status field on a entity_membership record
	EntityMembershipStatusDeleted EntityMembershipStatus = "DELETED"
	// EntityMembershipStatusSuspended represents the SUSPENDED state of the status field on a entity_membership record
	EntityMembershipStatusSuspended EntityMembershipStatus = "SUSPENDED"
)

// ParseEntityMembershipStatus converts a string into the correcponding enum value
func ParseEntityMembershipStatus(s string) (EntityMembershipStatus, error) {
	golog.Debugf("Entering dal.ParseEntityMembershipStatus...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseEntityMembershipStatus...") }()
	}
	switch t := EntityMembershipStatus(strings.ToUpper(s)); t {
	case EntityMembershipStatusActive, EntityMembershipStatusDeleted, EntityMembershipStatusSuspended:
		return t, nil
	}
	return EntityMembershipStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t EntityMembershipStatus) String() string {
	golog.Debugf("Entering dal.ParseEntityMembershipStatus.String...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseEntityMembershipStatus.String...") }()
	}
	return string(t)
}

// Scan allows for scanning of EntityMembershipStatus from a database conforming to the sql.Scanner interface
func (t *EntityMembershipStatus) Scan(src interface{}) error {
	golog.Debugf("Entering dal.ParseEntityMembershipStatus.Scan...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseEntityMembershipStatus.Scan...") }()
	}
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseEntityMembershipStatus(ts)
	case []byte:
		*t, err = ParseEntityMembershipStatus(string(ts))
	}
	return errors.Trace(err)
}

// EntityContactType represents the type associated with the type column of the entity_contact table
type EntityContactType string

const (
	// EntityContactTypePhone represents the PHONE state of the type field on a entity_contact record
	EntityContactTypePhone EntityContactType = "PHONE"
	// EntityContactTypeEmail represents the EMAIL state of the type field on a entity_contact record
	EntityContactTypeEmail EntityContactType = "EMAIL"
)

// ParseEntityContactType converts a string into the correcponding enum value
func ParseEntityContactType(s string) (EntityContactType, error) {
	golog.Debugf("Entering dal.ParseEntityContactType...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseEntityContactType...") }()
	}
	switch t := EntityContactType(strings.ToUpper(s)); t {
	case EntityContactTypePhone, EntityContactTypeEmail:
		return t, nil
	}
	return EntityContactType(""), errors.Trace(fmt.Errorf("Unknown type:%s", s))
}

func (t EntityContactType) String() string {
	golog.Debugf("Entering dal.ParseEntityContactType.String...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseEntityContactType.String...") }()
	}
	return string(t)
}

// Scan allows for scanning of EntityContactType from a database conforming to the sql.Scanner interface
func (t *EntityContactType) Scan(src interface{}) error {
	golog.Debugf("Entering dal.ParseEntityContactType.Scan...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.ParseEntityContactType.Scan...") }()
	}
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseEntityContactType(ts)
	case []byte:
		*t, err = ParseEntityContactType(string(ts))
	}
	return errors.Trace(err)
}

// Event represents a event record
type Event struct {
	Event    string
	Created  time.Time
	ID       EventID
	EntityID *EntityID
}

// EventUpdate represents the mutable aspects of a event record
type EventUpdate struct {
	Event *string
}

// EntityContact represents a entity_contact record
type EntityContact struct {
	ID          EntityContactID
	EntityID    EntityID
	Type        EntityContactType
	Value       string
	Provisioned bool
	Created     time.Time
	Modified    time.Time
}

// EntityContactUpdate represents the mutable aspects of a entity_contact record
type EntityContactUpdate struct {
	Type  *EntityContactType
	Value *string
}

// EntityMembership represents a entity_membership record
type EntityMembership struct {
	EntityID       EntityID
	TargetEntityID EntityID
	Status         EntityMembershipStatus
	Created        time.Time
	Modified       time.Time
}

// EntityMembershipUpdate represents the mutable aspects of a entity_membership record
type EntityMembershipUpdate struct {
	Status *EntityMembershipStatus
}

// ExternalEntityID represents a external_entity_id record
type ExternalEntityID struct {
	Created    time.Time
	Modified   time.Time
	EntityID   EntityID
	ExternalID string
}

// ExternalEntityIDUpdate represents the mutable aspects of a external_entity_id record
type ExternalEntityIDUpdate struct {
	EntityID   *EntityID
	ExternalID *string
}

// Entity represents a entity record
type Entity struct {
	ID       EntityID
	Name     string
	Type     EntityType
	Status   EntityStatus
	Created  time.Time
	Modified time.Time
}

// EntityUpdate represents the mutable aspects of a entity record
type EntityUpdate struct {
	Name   *string
	Type   *EntityType
	Status *EntityStatus
}

func (d *dal) InsertEntityMembership(model *EntityMembership) error {
	golog.Debugf("Entering dal.dal.InsertEntityMembership...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.InsertEntityMembership...") }()
	}
	_, err := d.db.Exec(
		`INSERT INTO entity_membership
          (entity_id, target_entity_id, status)
          VALUES (?, ?, ?)`, model.EntityID.Uint64(), model.TargetEntityID.Uint64(), model.Status.String())
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (d *dal) EntityMemberships(id EntityID) ([]*EntityMembership, error) {
	golog.Debugf("Entering dal.dal.EntityMemberships...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.EntityMemberships...") }()
	}
	rows, err := d.db.Query(
		`SELECT entity_id, target_entity_id, status, created, modified
		  FROM entity_membership
		  WHERE entity_id = ?`, id.Uint64())
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var memberships []*EntityMembership
	for rows.Next() {
		membership := &EntityMembership{}
		var entityID uint64
		var targetEntityID uint64
		if err := rows.Scan(&entityID, &targetEntityID, &membership.Status, &membership.Created, &membership.Modified); err != nil {
			return nil, errors.Trace(err)
		}
		membership.EntityID = NewEntityID(entityID)
		membership.TargetEntityID = NewEntityID(targetEntityID)
		memberships = append(memberships, membership)
	}
	return memberships, errors.Trace(rows.Err())
}

func (d *dal) EntityMembers(id EntityID) ([]*Entity, error) {
	golog.Debugf("Entering dal.dal.EntityMembers: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.EntityMemberships...") }()
	}
	rows, err := d.db.Query(
		`SELECT entity.created, entity.modified, entity.id, entity.name, entity.type, entity.status
		  FROM entity
		  JOIN entity_membership ON entity_membership.entity_id = entity.id
		  WHERE entity_membership.target_entity_id = ?`, id.Uint64())
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var entities []*Entity
	for rows.Next() {
		entity := &Entity{}
		var entityID uint64
		if err := rows.Scan(&entity.Created, &entity.Modified, &entityID, &entity.Name, &entity.Type, &entity.Status); err != nil {
			return nil, errors.Trace(err)
		}
		entity.ID = NewEntityID(entityID)
		entities = append(entities, entity)
	}
	return entities, errors.Trace(rows.Err())
}

func (d *dal) InsertEntityContact(model *EntityContact) (EntityContactID, error) {
	golog.Debugf("Entering dal.dal.InsertEntityContact...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.InsertEntityContact...") }()
	}
	if !model.ID.IsValid {
		id, err := idgen.NewID()
		if err != nil {
			return NewEntityContactID(0), errors.Trace(err)
		}
		model.ID = NewEntityContactID(id)
	}

	if _, err := d.db.Exec(
		`INSERT INTO entity_contact
          (value, id, entity_id, type, provisioned)
          VALUES (?, ?, ?, ?, ?)`, model.Value, model.ID.Uint64(), model.EntityID.Uint64(), model.Type.String(), model.Provisioned); err != nil {
		return NewEntityContactID(0), errors.Trace(err)
	}

	return NewEntityContactID(model.ID.Uint64()), nil
}

func (d *dal) EntityContacts(id EntityID) ([]*EntityContact, error) {
	golog.Debugf("Entering dal.dal.EntityContacts: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.EntityContacts...") }()
	}
	rows, err := d.db.Query(
		`SELECT entity_id, type, value, created, modified, id, provisioned
		  FROM entity_contact
		  WHERE entity_id = ?`, id.Uint64())
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var entityContacts []*EntityContact
	for rows.Next() {
		entityContact := &EntityContact{}
		var entityID uint64
		var entityContactID uint64
		if err := rows.Scan(&entityID, &entityContact.Type, &entityContact.Value, &entityContact.Created, &entityContact.Modified, &entityContactID, &entityContact.Provisioned); err != nil {
			return nil, errors.Trace(err)
		}
		entityContact.ID = NewEntityContactID(entityContactID)
		entityContact.EntityID = NewEntityID(entityID)
		entityContacts = append(entityContacts, entityContact)
	}
	return entityContacts, errors.Trace(rows.Err())
}

func (d *dal) EntityContact(id EntityContactID) (*EntityContact, error) {
	golog.Debugf("Entering dal.dal.EntityContact: %s", id)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.EntityContact...") }()
	}
	var entityIDv uint64
	var idv uint64
	model := &EntityContact{}
	if err := d.db.QueryRow(
		`SELECT entity_id, type, value, created, modified, id, provisioned
          FROM entity_contact
          WHERE id = ?`, id.Uint64()).Scan(&entityIDv, &model.Type, &model.Value, &model.Created, &model.Modified, &idv, &model.Provisioned); err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("entity_contact not found"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	model.EntityID = NewEntityID(entityIDv)
	model.ID = NewEntityContactID(idv)
	return model, nil
}

func (d *dal) EntityContactsForValue(value string) ([]*EntityContact, error) {
	golog.Debugf("Entering dal.dal.EntityContactsForValue: %s", value)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.EntityContactsForValue...") }()
	}
	rows, err := d.db.Query(
		`SELECT entity_id, type, value, created, modified, id, provisioned
		  FROM entity_contact
		  WHERE value = ?`, value)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var entityContacts []*EntityContact
	for rows.Next() {
		entityContact := &EntityContact{}
		var entityID uint64
		var entityContactID uint64
		if err := rows.Scan(&entityID, &entityContact.Type, &entityContact.Value, &entityContact.Created, &entityContact.Modified, &entityContactID, &entityContact.Provisioned); err != nil {
			return nil, errors.Trace(err)
		}
		entityContact.ID = NewEntityContactID(entityContactID)
		entityContact.EntityID = NewEntityID(entityID)
		entityContacts = append(entityContacts, entityContact)
	}
	return entityContacts, errors.Trace(rows.Err())
}

func (d *dal) UpdateEntityContact(id EntityContactID, update *EntityContactUpdate) (int64, error) {
	golog.Debugf("Entering dal.dal.UpdateEntityContact - ID: %s, Update: %v", id, update)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.UpdateEntityContact...") }()
	}
	args := dbutil.MySQLVarArgs()
	if update.Type != nil {
		args.Append("type", *update.Type)
	}
	if update.Value != nil {
		args.Append("value", *update.Value)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE entity_contact
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Uint64())...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) DeleteEntityContact(id EntityContactID) (int64, error) {
	golog.Debugf("Entering dal.dal.DeleteEntityContact...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.DeleteEntityContact...") }()
	}
	res, err := d.db.Exec(
		`DELETE FROM entity_contact
          WHERE id = ?`, id.Uint64())
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) InsertEvent(model *Event) (EventID, error) {
	golog.Debugf("Entering dal.dal.InsertEvent...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.InsertEvent...") }()
	}
	if !model.ID.IsValid {
		id, err := idgen.NewID()
		if err != nil {
			return NewEventID(0), errors.Trace(err)
		}
		model.ID = NewEventID(id)
	}

	if _, err := d.db.Exec(
		`INSERT INTO event
          (id, entity_id, event)
          VALUES (?, ?, ?, ?)`, model.ID.Uint64(), model.EntityID.Uint64(), model.Event); err != nil {
		return NewEventID(0), errors.Trace(err)
	}

	return NewEventID(model.ID.Uint64()), nil
}

func (d *dal) Event(id EventID) (*Event, error) {
	golog.Debugf("Entering dal.dal.Event...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.Event...") }()
	}
	var idv uint64
	var entityIDv *uint64
	model := &Event{}
	if err := d.db.QueryRow(
		`SELECT id, entity_id, event, created
          FROM event
          WHERE id = ?`, id.Uint64()).Scan(&idv, &entityIDv, &model.Event, &model.Created); err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("event not found"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	model.ID = NewEventID(idv)
	if entityIDv != nil {
		nID := NewEntityID(*entityIDv)
		model.EntityID = &nID
	}

	return model, nil
}

func (d *dal) UpdateEvent(id EventID, update *EventUpdate) (int64, error) {
	golog.Debugf("Entering dal.dal.UpdateEvent...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.UpdateEvent...") }()
	}
	args := dbutil.MySQLVarArgs()
	if update.Event != nil {
		args.Append("event", *update.Event)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE event
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Uint64())...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) DeleteEvent(id EventID) (int64, error) {
	golog.Debugf("Entering dal.dal.DeleteEvent...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.DeleteEvent...") }()
	}
	res, err := d.db.Exec(
		`DELETE FROM event
          WHERE id = ?`, id.Uint64())
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) InsertEntity(model *Entity) (EntityID, error) {
	golog.Debugf("Entering dal.dal.InsertEntity...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.InsertEntity...") }()
	}
	if !model.ID.IsValid {
		id, err := idgen.NewID()
		if err != nil {
			return NewEntityID(0), errors.Trace(err)
		}
		model.ID = NewEntityID(id)
	}

	if _, err := d.db.Exec(
		`INSERT INTO entity
          (type, status, id, name)
          VALUES (?, ?, ?, ?)`, model.Type.String(), model.Status.String(), model.ID.Uint64(), model.Name); err != nil {
		return NewEntityID(0), errors.Trace(err)
	}

	return NewEntityID(model.ID.Uint64()), nil
}

func (d *dal) Entity(id EntityID) (*Entity, error) {
	golog.Debugf("Entering dal.dal.Entity...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.Entity...") }()
	}
	var idv uint64
	model := &Entity{}
	if err := d.db.QueryRow(
		`SELECT created, modified, id, name, type, status
          FROM entity
          WHERE id = ?`, id.Uint64()).Scan(&model.Created, &model.Modified, &idv, &model.Name, &model.Type, &model.Status); err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("entity not found"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	model.ID = NewEntityID(idv)
	return model, nil
}

func (d *dal) Entities(ids []EntityID) ([]*Entity, error) {
	golog.Debugf("Entering dal.dal.Entities...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.Entities...") }()
	}
	if len(ids) == 0 {
		return nil, nil
	}

	argIDs := make([]interface{}, len(ids))
	for i, id := range ids {
		argIDs[i] = id.Uint64()
	}
	rows, err := d.db.Query(
		`SELECT created, modified, id, name, type, status
		  FROM entity
		  WHERE id IN (`+dbutil.MySQLArgs(len(ids))+`)`, argIDs...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var entities []*Entity
	for rows.Next() {
		entity := &Entity{}
		var id uint64
		if err := rows.Scan(&entity.Created, &entity.Modified, &id, &entity.Name, &entity.Type, &entity.Status); err != nil {
			return nil, errors.Trace(err)
		}
		entity.ID = NewEntityID(id)
		entities = append(entities, entity)
	}
	return entities, errors.Trace(rows.Err())
}

func (d *dal) UpdateEntity(id EntityID, update *EntityUpdate) (int64, error) {
	golog.Debugf("Entering dal.dal.UpdateEntity...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.UpdateEntity...") }()
	}
	args := dbutil.MySQLVarArgs()
	if update.Name != nil {
		args.Append("name", *update.Name)
	}
	if update.Type != nil {
		args.Append("type", *update.Type)
	}
	if update.Status != nil {
		args.Append("status", *update.Status)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE entity
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Uint64())...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) DeleteEntity(id EntityID) (int64, error) {
	golog.Debugf("Entering dal.dal.DeleteEntity...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.DeleteEntity...") }()
	}
	res, err := d.db.Exec(
		`DELETE FROM entity
          WHERE id = ?`, id.Uint64())
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) InsertExternalEntityID(model *ExternalEntityID) error {
	golog.Debugf("Entering dal.dal.InsertExternalEntityID...")
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.InsertExternalEntityID...") }()
	}
	if _, err := d.db.Exec(
		`INSERT INTO external_entity_id
          (entity_id, external_id)
          VALUES (?, ?)`, model.EntityID.Uint64(), model.ExternalID); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (d *dal) ExternalEntityIDs(externalID string) ([]*ExternalEntityID, error) {
	golog.Debugf("Entering dal.dal.ExternalEntities: %s", externalID)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.ExternalEntities...") }()
	}
	rows, err := d.db.Query(
		`SELECT entity_id, external_id, created, modified
		  FROM external_entity_id
		  WHERE external_id = ?`, externalID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var externalEntityIDs []*ExternalEntityID
	for rows.Next() {
		externalEntityID := &ExternalEntityID{}
		var entityID uint64
		if err := rows.Scan(&entityID, &externalEntityID.ExternalID, &externalEntityID.Created, &externalEntityID.Modified); err != nil {
			return nil, errors.Trace(err)
		}
		externalEntityID.EntityID = NewEntityID(entityID)
		externalEntityIDs = append(externalEntityIDs, externalEntityID)
	}
	return externalEntityIDs, errors.Trace(rows.Err())
}

// ExternalEntityIDsForEntities returns all the external ids that map to the provided list of entity ids
func (d *dal) ExternalEntityIDsForEntities(entityIDs []EntityID) ([]*ExternalEntityID, error) {
	golog.Debugf("Entering dal.dal.ExternalEntityIDsForEntities: %s", entityIDs)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving dal.dal.ExternalEntityIDsForEntities...") }()
	}
	if len(entityIDs) == 0 {
		return nil, nil
	}

	values := make([]interface{}, len(entityIDs))
	for i, v := range entityIDs {
		values[i] = v.Uint64()
	}
	rows, err := d.db.Query(
		`SELECT entity_id, external_id, created, modified
		  FROM external_entity_id
		  WHERE entity_id IN (`+dbutil.MySQLArgs(len(entityIDs))+`)`, values...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var externalEntityIDs []*ExternalEntityID
	for rows.Next() {
		externalEntityID := &ExternalEntityID{}
		var entityID uint64
		if err := rows.Scan(&entityID, &externalEntityID.ExternalID, &externalEntityID.Created, &externalEntityID.Modified); err != nil {
			return nil, errors.Trace(err)
		}
		externalEntityID.EntityID = NewEntityID(entityID)
		externalEntityIDs = append(externalEntityIDs, externalEntityID)
	}
	return externalEntityIDs, errors.Trace(rows.Err())
}
