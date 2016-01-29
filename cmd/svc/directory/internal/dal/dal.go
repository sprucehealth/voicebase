package dal

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	modellib "github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
	"github.com/sprucehealth/backend/svc/directory"
)

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	Transact(trans func(dal DAL) error) (err error)
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
	InsertEntityContacts(models []*EntityContact) error
	EntityContact(id EntityContactID) (*EntityContact, error)
	EntityContacts(id EntityID) ([]*EntityContact, error)
	EntityContactsForValue(value string) ([]*EntityContact, error)
	UpdateEntityContact(id EntityContactID, update *EntityContactUpdate) (int64, error)
	DeleteEntityContact(id EntityContactID) (int64, error)
	DeleteEntityContactsForEntityID(id EntityID) (int64, error)
	InsertEvent(model *Event) (EventID, error)
	Event(id EventID) (*Event, error)
	UpdateEvent(id EventID, update *EventUpdate) (int64, error)
	DeleteEvent(id EventID) (int64, error)
	EntityDomain(id *EntityID, domain *string) (EntityID, string, error)
	InsertEntityDomain(id EntityID, domain string) error
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

// EventIDPrefix represents the string that is attached to the beginning of these identifiers
const EventIDPrefix = "event_"

// NewEventID returns a new EventID.
func NewEventID() (EventID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return EventID{}, errors.Trace(err)
	}
	return EventID{
		modellib.ObjectID{
			Prefix:  EventIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyEventID returns an empty initialized ID
func EmptyEventID() EventID {
	return EventID{
		modellib.ObjectID{
			Prefix:  EventIDPrefix,
			IsValid: false,
		},
	}
}

// ParseEventID transforms an EventID from it's string representation into the actual ID value
func ParseEventID(s string) (EventID, error) {
	id := EmptyEventID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// EventID is the ID for a EventID object
type EventID struct {
	modellib.ObjectID
}

// NewEntityID returns a new EntityID.
func NewEntityID() (EntityID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return EntityID{}, errors.Trace(err)
	}
	return EntityID{
		modellib.ObjectID{
			Prefix:  directory.EntityIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyEntityID returns an empty initialized ID
func EmptyEntityID() EntityID {
	return EntityID{
		modellib.ObjectID{
			Prefix:  directory.EntityIDPrefix,
			IsValid: false,
		},
	}
}

// ParseEntityID transforms an EntityID from it's string representation into the actual ID value
func ParseEntityID(s string) (EntityID, error) {
	id := EmptyEntityID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// EntityID is the ID for a EntityID object
type EntityID struct {
	modellib.ObjectID
}

// NewEntityContactID returns a new EntityContactID.
func NewEntityContactID() (EntityContactID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return EntityContactID{}, errors.Trace(err)
	}
	return EntityContactID{
		modellib.ObjectID{
			Prefix:  directory.EntityContactIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyEntityContactID returns an empty initialized ID
func EmptyEntityContactID() EntityContactID {
	return EntityContactID{
		modellib.ObjectID{
			Prefix:  directory.EntityContactIDPrefix,
			IsValid: false,
		},
	}
}

// ParseEntityContactID transforms an EntityContactID from it's string representation into the actual ID value
func ParseEntityContactID(s string) (EntityContactID, error) {
	id := EmptyEntityContactID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// EntityContactID is the ID for a EntityContactID object
type EntityContactID struct {
	modellib.ObjectID
}

// EntityType represents the type associated with the type column of the entity table
type EntityType string

const (
	// EntityTypeOrganization represents the ORGANIZATION state of the type field on a entity record
	EntityTypeOrganization EntityType = "ORGANIZATION"
	// EntityTypeInternal represents the INTERNAL state of the type field on a entity record
	EntityTypeInternal EntityType = "INTERNAL"
	// EntityTypeExternal represents the EXTERNAL state of the type field on a entity record
	EntityTypeExternal EntityType = "EXTERNAL"
)

// ParseEntityType converts a string into the correcponding enum value
func ParseEntityType(s string) (EntityType, error) {
	switch t := EntityType(strings.ToUpper(s)); t {
	case EntityTypeOrganization, EntityTypeInternal, EntityTypeExternal:
		return t, nil
	}
	return EntityType(""), errors.Trace(fmt.Errorf("Unknown type:%s", s))
}

func (t EntityType) String() string {
	return string(t)
}

// Scan allows for scanning of EntityType from a database conforming to the sql.Scanner interface
func (t *EntityType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseEntityType(ts)
	case []byte:
		*t, err = ParseEntityType(string(ts))
	default:
		return errors.Trace(fmt.Errorf("Unsupported type %T with value %+v in enumeration scan", src, src))
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
	switch t := EntityStatus(strings.ToUpper(s)); t {
	case EntityStatusActive, EntityStatusDeleted, EntityStatusSuspended:
		return t, nil
	}
	return EntityStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t EntityStatus) String() string {
	return string(t)
}

// Scan allows for scanning of EntityStatus from a database conforming to the sql.Scanner interface
func (t *EntityStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseEntityStatus(ts)
	case []byte:
		*t, err = ParseEntityStatus(string(ts))
	default:
		return errors.Trace(fmt.Errorf("Unsupported type %T with value %+v in enumeration scan", src, src))
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
	switch t := EntityMembershipStatus(strings.ToUpper(s)); t {
	case EntityMembershipStatusActive, EntityMembershipStatusDeleted, EntityMembershipStatusSuspended:
		return t, nil
	}
	return EntityMembershipStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t EntityMembershipStatus) String() string {
	return string(t)
}

// Scan allows for scanning of EntityMembershipStatus from a database conforming to the sql.Scanner interface
func (t *EntityMembershipStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseEntityMembershipStatus(ts)
	case []byte:
		*t, err = ParseEntityMembershipStatus(string(ts))
	default:
		return errors.Trace(fmt.Errorf("Unsupported type %T with value %+v in enumeration scan", src, src))
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
	switch t := EntityContactType(strings.ToUpper(s)); t {
	case EntityContactTypePhone, EntityContactTypeEmail:
		return t, nil
	}
	return EntityContactType(""), errors.Trace(fmt.Errorf("Unknown type:%s", s))
}

func (t EntityContactType) String() string {
	return string(t)
}

// Scan allows for scanning of EntityContactType from a database conforming to the sql.Scanner interface
func (t *EntityContactType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseEntityContactType(ts)
	case []byte:
		*t, err = ParseEntityContactType(string(ts))
	default:
		return errors.Trace(fmt.Errorf("Unsupported type %T with value %+v in enumeration scan", src, src))
	}
	return errors.Trace(err)
}

// Event represents a event record
type Event struct {
	ID       EventID
	EntityID EntityID
	Event    string
	Created  time.Time
}

// EventUpdate represents the mutable aspects of a event record
type EventUpdate struct {
	Event *string
}

// EntityContact represents a entity_contact record
type EntityContact struct {
	EntityID    EntityID
	Type        EntityContactType
	Value       string
	Created     time.Time
	Modified    time.Time
	ID          EntityContactID
	Label       string
	Provisioned bool
}

// EntityContactUpdate represents the mutable aspects of a entity_contact record
type EntityContactUpdate struct {
	Type  *EntityContactType
	Value *string
	Label *string
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
	Modified   time.Time
	EntityID   EntityID
	ExternalID string
	Created    time.Time
}

// ExternalEntityIDUpdate represents the mutable aspects of a external_entity_id record
type ExternalEntityIDUpdate struct {
	EntityID   EntityID
	ExternalID *string
}

// Entity represents a entity record
type Entity struct {
	ID            EntityID
	Type          EntityType
	Status        EntityStatus
	DisplayName   string
	FirstName     string
	GroupName     string
	Note          string
	MiddleInitial string
	LastName      string
	ShortTitle    string
	LongTitle     string
	Created       time.Time
	Modified      time.Time
}

// EntityUpdate represents the mutable aspects of a entity record
type EntityUpdate struct {
	DisplayName   *string
	FirstName     *string
	GroupName     *string
	Type          *EntityType
	Status        *EntityStatus
	MiddleInitial *string
	LastName      *string
	ShortTitle    *string
	LongTitle     *string
	Note          *string
}

// InsertEntity inserts a entity record
func (d *dal) InsertEntity(model *Entity) (EntityID, error) {
	if !model.ID.IsValid {
		id, err := NewEntityID()
		if err != nil {
			return EmptyEntityID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO entity
          (display_name, first_name, group_name, type, status, id, middle_initial, last_name, note, short_title, long_title)
          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, model.DisplayName, model.FirstName, model.GroupName, model.Type.String(), model.Status.String(), model.ID, model.MiddleInitial, model.LastName, model.Note, model.ShortTitle, model.LongTitle)
	if err != nil {
		return EmptyEntityID(), errors.Trace(err)
	}

	return model.ID, nil
}

// Entity retrieves a entity record
func (d *dal) Entity(id EntityID) (*Entity, error) {
	row := d.db.QueryRow(
		selectEntity+` WHERE id = ?`, id.Val)
	model, err := scanEntity(row)
	return model, errors.Trace(err)
}

// Entities returns the entity record associated with the provided IDs
func (d *dal) Entities(ids []EntityID) ([]*Entity, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	vals := make([]interface{}, len(ids))
	for i, v := range ids {
		vals[i] = v
	}
	rows, err := d.db.Query(
		selectEntity+` WHERE id IN (`+dbutil.MySQLArgs(len(ids))+`)`, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var entities []*Entity
	for rows.Next() {
		entity, err := scanEntity(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		entities = append(entities, entity)
	}
	return entities, errors.Trace(rows.Err())
}

// UpdateEntity updates the mutable aspects of a entity record
func (d *dal) UpdateEntity(id EntityID, update *EntityUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.DisplayName != nil {
		args.Append("display_name", *update.DisplayName)
	}
	if update.FirstName != nil {
		args.Append("first_name", *update.FirstName)
	}
	if update.GroupName != nil {
		args.Append("group_name", *update.GroupName)
	}
	if update.ShortTitle != nil {
		args.Append("short_title", *update.ShortTitle)
	}
	if update.LongTitle != nil {
		args.Append("long_title", *update.LongTitle)
	}
	if update.Type != nil {
		args.Append("type", *update.Type)
	}
	if update.Status != nil {
		args.Append("status", *update.Status)
	}
	if update.MiddleInitial != nil {
		args.Append("middle_initial", *update.MiddleInitial)
	}
	if update.LastName != nil {
		args.Append("last_name", *update.LastName)
	}
	if update.Note != nil {
		args.Append("note", *update.Note)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE entity
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteEntity deletes a entity record
func (d *dal) DeleteEntity(id EntityID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM entity
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertExternalEntityID inserts a external_entity_id record
func (d *dal) InsertExternalEntityID(model *ExternalEntityID) error {
	_, err := d.db.Exec(
		`INSERT INTO external_entity_id
          (entity_id, external_id)
          VALUES (?, ?)`, model.EntityID, model.ExternalID)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// ExternalEntityIDs returns the external_entity_id records associated with the externalID
func (d *dal) ExternalEntityIDs(externalID string) ([]*ExternalEntityID, error) {
	rows, err := d.db.Query(
		selectExternalEntityID+` WHERE external_id = ?`, externalID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var externalEntityIDs []*ExternalEntityID
	for rows.Next() {
		externalEntityID, err := scanExternalEntityID(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		externalEntityIDs = append(externalEntityIDs, externalEntityID)
	}
	return externalEntityIDs, errors.Trace(rows.Err())
}

// ExternalEntityIDsForEntities returns all the external ids that map to the provided list of entity ids
func (d *dal) ExternalEntityIDsForEntities(entityIDs []EntityID) ([]*ExternalEntityID, error) {
	if len(entityIDs) == 0 {
		return nil, nil
	}

	vals := make([]interface{}, len(entityIDs))
	for i, v := range entityIDs {
		vals[i] = v
	}
	rows, err := d.db.Query(
		selectExternalEntityID+` WHERE entity_id IN (`+dbutil.MySQLArgs(len(entityIDs))+`)`, vals...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var externalEntityIDs []*ExternalEntityID
	for rows.Next() {
		externalEntityID, err := scanExternalEntityID(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		externalEntityIDs = append(externalEntityIDs, externalEntityID)
	}
	return externalEntityIDs, errors.Trace(rows.Err())
}

// InsertEntityMembership inserts a entity_membership record
func (d *dal) InsertEntityMembership(model *EntityMembership) error {
	_, err := d.db.Exec(
		`INSERT INTO entity_membership
          (entity_id, target_entity_id, status)
          VALUES (?, ?, ?)`, model.EntityID, model.TargetEntityID, model.Status.String())
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// EntityMemberships returns the memberships for the provided entity ID
func (d *dal) EntityMemberships(id EntityID) ([]*EntityMembership, error) {
	rows, err := d.db.Query(
		selectEntityMembership+` WHERE entity_id = ?`, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var memberships []*EntityMembership
	for rows.Next() {
		membership, err := scanEntityMembership(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		memberships = append(memberships, membership)
	}
	return memberships, errors.Trace(rows.Err())
}

// EntityMembers returns all the members of the provided entity id
func (d *dal) EntityMembers(id EntityID) ([]*Entity, error) {
	rows, err := d.db.Query(
		selectEntity+` JOIN entity_membership ON entity_membership.entity_id = entity.id
		  WHERE entity_membership.target_entity_id = ?`, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var entities []*Entity
	for rows.Next() {
		entity, err := scanEntity(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		entities = append(entities, entity)
	}
	return entities, errors.Trace(rows.Err())
}

// InsertEntityContact inserts a entity_contact record
func (d *dal) InsertEntityContact(model *EntityContact) (EntityContactID, error) {
	if !model.ID.IsValid {
		id, err := NewEntityContactID()
		if err != nil {
			return EmptyEntityContactID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO entity_contact
          (id, entity_id, type, value, provisioned, label)
          VALUES (?, ?, ?, ?, ?, ?)`, model.ID, model.EntityID, model.Type.String(), model.Value, model.Provisioned, model.Label)
	if err != nil {
		return EmptyEntityContactID(), errors.Trace(err)
	}

	return model.ID, nil
}

// InsertEntityContacts inserts a set of entity_contact record
func (d *dal) InsertEntityContacts(models []*EntityContact) error {
	if len(models) == 0 {
		return nil
	}

	ins := dbutil.MySQLMultiInsert(len(models))
	for i, m := range models {
		if !m.ID.IsValid {
			id, err := NewEntityContactID()
			if err != nil {
				return errors.Trace(err)
			}
			models[i].ID = id
		}
		ins.Append(m.ID, m.EntityID, m.Type.String(), m.Value, m.Provisioned, m.Label)
	}
	_, err := d.db.Exec(
		`INSERT IGNORE INTO entity_contact
          (id, entity_id, type, value, provisioned, label)
          VALUES `+ins.Query(), ins.Values()...)
	return errors.Trace(err)
}

// EntityContacts returns the entity_contact rescords for the provided entity id
func (d *dal) EntityContacts(id EntityID) ([]*EntityContact, error) {
	rows, err := d.db.Query(
		selectEntityContact+` WHERE entity_id = ?`, id)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var entityContacts []*EntityContact
	for rows.Next() {
		entityContact, err := scanEntityContact(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		entityContacts = append(entityContacts, entityContact)
	}
	return entityContacts, errors.Trace(rows.Err())
}

// EntityContactsForValue returns the entity_contact records with the provided value
func (d *dal) EntityContactsForValue(value string) ([]*EntityContact, error) {
	rows, err := d.db.Query(
		selectEntityContact+` WHERE value = ?`, value)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var entityContacts []*EntityContact
	for rows.Next() {
		entityContact, err := scanEntityContact(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		entityContacts = append(entityContacts, entityContact)
	}
	return entityContacts, errors.Trace(rows.Err())
}

// EntityContact retrieves a entity_contact record
func (d *dal) EntityContact(id EntityContactID) (*EntityContact, error) {
	row := d.db.QueryRow(
		selectEntityContact+` WHERE id = ?`, id.Val)
	model, err := scanEntityContact(row)
	return model, errors.Trace(err)
}

// UpdateEntityContact updates the mutable aspects of a entity_contact record
func (d *dal) UpdateEntityContact(id EntityContactID, update *EntityContactUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Type != nil {
		args.Append("type", update.Type.String())
	}
	if update.Value != nil {
		args.Append("value", *update.Value)
	}
	if update.Label != nil {
		args.Append("label", *update.Label)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE entity_contact
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteEntityContact deletes a entity_contact record
func (d *dal) DeleteEntityContact(id EntityContactID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM entity_contact
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteEntityContactForEntityID deletes all entity_contact records for the provided entity_id
func (d *dal) DeleteEntityContactsForEntityID(id EntityID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM entity_contact
          WHERE entity_id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertEvent inserts a event record
func (d *dal) InsertEvent(model *Event) (EventID, error) {
	if !model.ID.IsValid {
		id, err := NewEventID()
		if err != nil {
			return EmptyEventID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO event
          (id, entity_id, event)
          VALUES (?, ?, ?)`, model.ID, model.EntityID, model.Event)
	if err != nil {
		return EmptyEventID(), errors.Trace(err)
	}

	return model.ID, nil
}

// Event retrieves a event record
func (d *dal) Event(id EventID) (*Event, error) {
	row := d.db.QueryRow(
		selectEvent+` WHERE id = ?`, id.Val)
	model, err := scanEvent(row)
	return model, errors.Trace(err)
}

// UpdateEvent updates the mutable aspects of a event record
func (d *dal) UpdateEvent(id EventID, update *EventUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Event != nil {
		args.Append("event", *update.Event)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE event
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteEvent deletes a event record
func (d *dal) DeleteEvent(id EventID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM event
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

func (d *dal) EntityDomain(id *EntityID, domain *string) (EntityID, string, error) {
	if id == nil && domain == nil {
		return EmptyEntityID(), "", errors.Trace(errors.New("either entity_id or domain must be specified to lookup entity_domain"))
	}

	where := make([]string, 0, 2)
	vals := make([]interface{}, 0, 2)

	if id != nil {
		where = append(where, "entity_id = ?")
		vals = append(vals, id)
	}
	if domain != nil {
		where = append(where, "domain = ?")
		vals = append(vals, domain)
	}

	var queriedDomain string
	var queriedEntityID EntityID
	if err := d.db.QueryRow(`
		SELECT entity_id, domain
		FROM entity_domain
		WHERE `+strings.Join(where, " AND "), vals...).Scan(&queriedEntityID, &queriedDomain); err == sql.ErrNoRows {
		return EmptyEntityID(), "", errors.Trace(api.ErrNotFound("entity_domain not found"))
	} else if err != nil {
		return EmptyEntityID(), "", errors.Trace(err)
	}

	return queriedEntityID, queriedDomain, nil
}

func (d *dal) InsertEntityDomain(id EntityID, domain string) error {
	_, err := d.db.Exec(`REPLACE INTO entity_domain (entity_id, domain) VALUES (?,?)`, id, domain)
	return errors.Trace(err)
}

const selectExternalEntityID = `
    SELECT external_entity_id.created, external_entity_id.modified, external_entity_id.entity_id, external_entity_id.external_id
      FROM external_entity_id`

func scanExternalEntityID(row dbutil.Scanner) (*ExternalEntityID, error) {
	var m ExternalEntityID
	m.EntityID = EmptyEntityID()

	err := row.Scan(&m.Created, &m.Modified, &m.EntityID, &m.ExternalID)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("directory - ExternalEntityID not found"))
	}
	return &m, errors.Trace(err)
}

const selectEntityMembership = `
    SELECT entity_membership.entity_id, entity_membership.target_entity_id, entity_membership.status, entity_membership.created, entity_membership.modified
      FROM entity_membership`

func scanEntityMembership(row dbutil.Scanner) (*EntityMembership, error) {
	var m EntityMembership
	m.EntityID = EmptyEntityID()
	m.TargetEntityID = EmptyEntityID()

	err := row.Scan(&m.EntityID, &m.TargetEntityID, &m.Status, &m.Created, &m.Modified)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("directory - EntityMembership not found"))
	}
	return &m, errors.Trace(err)
}

const selectEntityContact = `
    SELECT entity_contact.modified, entity_contact.id, entity_contact.entity_id, entity_contact.type, entity_contact.value, entity_contact.created, entity_contact.provisioned, entity_contact.label
      FROM entity_contact`

func scanEntityContact(row dbutil.Scanner) (*EntityContact, error) {
	var m EntityContact
	m.ID = EmptyEntityContactID()
	m.EntityID = EmptyEntityID()

	err := row.Scan(&m.Modified, &m.ID, &m.EntityID, &m.Type, &m.Value, &m.Created, &m.Provisioned, &m.Label)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("directory - EntityContact not found"))
	}
	return &m, errors.Trace(err)
}

const selectEvent = `
    SELECT event.id, event.entity_id, event.event, event.created
      FROM event`

func scanEvent(row dbutil.Scanner) (*Event, error) {
	var m Event
	m.ID = EmptyEventID()
	m.EntityID = EmptyEntityID()

	err := row.Scan(&m.ID, &m.EntityID, &m.Event, &m.Created)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("directory - Event not found"))
	}
	return &m, errors.Trace(err)
}

const selectEntity = `
    SELECT entity.id, entity.middle_initial, entity.last_name, entity.note, entity.created, entity.modified, entity.display_name, entity.first_name, entity.group_name, entity.type, entity.status, entity.short_title, entity.long_title
      FROM entity`

func scanEntity(row dbutil.Scanner) (*Entity, error) {
	var m Entity
	m.ID = EmptyEntityID()

	err := row.Scan(&m.ID, &m.MiddleInitial, &m.LastName, &m.Note, &m.Created, &m.Modified, &m.DisplayName, &m.FirstName, &m.GroupName, &m.Type, &m.Status, &m.ShortTitle, &m.LongTitle)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(api.ErrNotFound("directory - Entity not found"))
	}
	return &m, errors.Trace(err)
}
