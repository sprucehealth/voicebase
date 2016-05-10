package dal

import (
	"database/sql"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	modellib "github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/transactional/tsql"
)

var ErrNotFound = errors.New("deploy/dal: object not found")

// DAL represents the methods required to provide data access layer functionality
type DAL interface {
	Transact(trans func(dal DAL) error) (err error)
	InsertDeployableVector(model *DeployableVector) (DeployableVectorID, error)
	DeployableVector(id DeployableVectorID) (*DeployableVector, error)
	DeployableVectorsForDeployable(depID DeployableID) ([]*DeployableVector, error)
	DeployableVectorsForDeployableAndSource(id DeployableID, source DeployableVectorSourceType) ([]*DeployableVector, error)
	DeployableVectorsForDeployableAndSourceEnvironment(depID DeployableID, sourceEnv EnvironmentID) ([]*DeployableVector, error)
	DeployableVectorForDeployableSourceTarget(depID DeployableID, sourceType DeployableVectorSourceType, sourceEnvID, targetEnvID EnvironmentID) (*DeployableVector, error)
	DeleteDeployableVector(id DeployableVectorID) (int64, error)
	InsertDeployableGroup(model *DeployableGroup) (DeployableGroupID, error)
	DeployableGroup(id DeployableGroupID) (*DeployableGroup, error)
	DeployableGroupForName(name string) (*DeployableGroup, error)
	DeployablesForGroup(groupID DeployableGroupID) ([]*Deployable, error)
	DeployableGroups() ([]*DeployableGroup, error)
	UpdateDeployableGroup(id DeployableGroupID, update *DeployableGroupUpdate) (int64, error)
	DeleteDeployableGroup(id DeployableGroupID) (int64, error)
	InsertDeployable(model *Deployable) (DeployableID, error)
	Deployable(id DeployableID) (*Deployable, error)
	UpdateDeployable(id DeployableID, update *DeployableUpdate) (int64, error)
	DeleteDeployable(id DeployableID) (int64, error)
	DeployableForNameAndGroup(name string, groupID DeployableGroupID) (*Deployable, error)
	InsertEnvironment(model *Environment) (EnvironmentID, error)
	Environment(id EnvironmentID) (*Environment, error)
	EnvironmentForNameAndGroup(name string, groupID DeployableGroupID) (*Environment, error)
	EnvironmentsForGroup(id DeployableGroupID) ([]*Environment, error)
	UpdateEnvironment(id EnvironmentID, update *EnvironmentUpdate) (int64, error)
	DeleteEnvironment(id EnvironmentID) (int64, error)
	InsertEnvironmentConfig(model *EnvironmentConfig) (EnvironmentConfigID, error)
	DeprecateActiveEnvironmentConfig(envID EnvironmentID) (int64, error)
	EnvironmentConfig(id EnvironmentConfigID) (*EnvironmentConfig, error)
	EnvironmentConfigsForStatus(environmentID EnvironmentID, status EnvironmentConfigStatus) ([]*EnvironmentConfig, error)
	UpdateEnvironmentConfig(id EnvironmentConfigID, update *EnvironmentConfigUpdate) (int64, error)
	DeleteEnvironmentConfig(id EnvironmentConfigID) (int64, error)
	InsertEnvironmentConfigValue(model *EnvironmentConfigValue) error
	InsertEnvironmentConfigValues(model []*EnvironmentConfigValue) error
	EnvironmentConfigValues(id EnvironmentConfigID) ([]*EnvironmentConfigValue, error)
	InsertDeployableConfig(model *DeployableConfig) (DeployableConfigID, error)
	DeprecateActiveDeployableConfig(deployableID DeployableID, envID EnvironmentID) (int64, error)
	DeployableConfig(id DeployableConfigID) (*DeployableConfig, error)
	DeployableConfigsForStatus(depID DeployableID, environmentID EnvironmentID, status DeployableConfigStatus) ([]*DeployableConfig, error)
	UpdateDeployableConfig(id DeployableConfigID, update *DeployableConfigUpdate) (int64, error)
	DeleteDeployableConfig(id DeployableConfigID) (int64, error)
	InsertDeployableConfigValue(model *DeployableConfigValue) error
	InsertDeployableConfigValues(model []*DeployableConfigValue) error
	DeployableConfigValues(id DeployableConfigID) ([]*DeployableConfigValue, error)
	InsertDeployment(model *Deployment) (DeploymentID, error)
	Deployment(id DeploymentID) (*Deployment, error)
	Deployments(depID DeployableID) ([]*Deployment, error)
	DeploymentsForStatus(depID DeployableID, status DeploymentStatus) ([]*Deployment, error)
	DeleteDeployment(id DeploymentID) (int64, error)
	DeploymentsForDeploymentGroup(depID DeployableGroupID, envID EnvironmentID, buildNumber string) ([]*Deployment, error)
	ActiveDeployment(depID DeployableID, envID EnvironmentID) (*Deployment, error)
	NextPendingDeployment() (*Deployment, error)
	SetDeploymentStatus(id DeploymentID, s DeploymentStatus) error
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

// DeploymentIDPrefix represents the string that is attached to the beginning of these identifiers
const DeploymentIDPrefix = "deployment_"

// NewDeploymentID returns a new DeploymentID.
func NewDeploymentID() (DeploymentID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return DeploymentID{}, errors.Trace(err)
	}
	return DeploymentID{
		modellib.ObjectID{
			Prefix:  DeploymentIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyDeploymentID returns an empty initialized ID
func EmptyDeploymentID() DeploymentID {
	return DeploymentID{
		modellib.ObjectID{
			Prefix:  DeploymentIDPrefix,
			IsValid: false,
		},
	}
}

// ParseDeploymentID transforms an DeploymentID from it's string representation into the actual ID value
func ParseDeploymentID(s string) (DeploymentID, error) {
	id := EmptyDeploymentID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// DeploymentID is the ID for a DeploymentID object
type DeploymentID struct {
	modellib.ObjectID
}

// DeployableConfigIDPrefix represents the string that is attached to the beginning of these identifiers
const DeployableConfigIDPrefix = "deployableConfig_"

// NewDeployableConfigID returns a new DeployableConfigID.
func NewDeployableConfigID() (DeployableConfigID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return DeployableConfigID{}, errors.Trace(err)
	}
	return DeployableConfigID{
		modellib.ObjectID{
			Prefix:  DeployableConfigIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyDeployableConfigID returns an empty initialized ID
func EmptyDeployableConfigID() DeployableConfigID {
	return DeployableConfigID{
		modellib.ObjectID{
			Prefix:  DeployableConfigIDPrefix,
			IsValid: false,
		},
	}
}

// ParseDeployableConfigID transforms an DeployableConfigID from it's string representation into the actual ID value
func ParseDeployableConfigID(s string) (DeployableConfigID, error) {
	id := EmptyDeployableConfigID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// DeployableConfigID is the ID for a DeployableConfigID object
type DeployableConfigID struct {
	modellib.ObjectID
}

// EnvironmentIDPrefix represents the string that is attached to the beginning of these identifiers
const EnvironmentIDPrefix = "environment_"

// NewEnvironmentID returns a new EnvironmentID.
func NewEnvironmentID() (EnvironmentID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return EnvironmentID{}, errors.Trace(err)
	}
	return EnvironmentID{
		modellib.ObjectID{
			Prefix:  EnvironmentIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyEnvironmentID returns an empty initialized ID
func EmptyEnvironmentID() EnvironmentID {
	return EnvironmentID{
		modellib.ObjectID{
			Prefix:  EnvironmentIDPrefix,
			IsValid: false,
		},
	}
}

// ParseEnvironmentID transforms an EnvironmentID from it's string representation into the actual ID value
func ParseEnvironmentID(s string) (EnvironmentID, error) {
	id := EmptyEnvironmentID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// EnvironmentID is the ID for a EnvironmentID object
type EnvironmentID struct {
	modellib.ObjectID
}

// EnvironmentConfigIDPrefix represents the string that is attached to the beginning of these identifiers
const EnvironmentConfigIDPrefix = "environmentConfig_"

// NewEnvironmentConfigID returns a new EnvironmentConfigID.
func NewEnvironmentConfigID() (EnvironmentConfigID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return EnvironmentConfigID{}, errors.Trace(err)
	}
	return EnvironmentConfigID{
		modellib.ObjectID{
			Prefix:  EnvironmentConfigIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyEnvironmentConfigID returns an empty initialized ID
func EmptyEnvironmentConfigID() EnvironmentConfigID {
	return EnvironmentConfigID{
		modellib.ObjectID{
			Prefix:  EnvironmentConfigIDPrefix,
			IsValid: false,
		},
	}
}

// ParseEnvironmentConfigID transforms an EnvironmentConfigID from it's string representation into the actual ID value
func ParseEnvironmentConfigID(s string) (EnvironmentConfigID, error) {
	id := EmptyEnvironmentConfigID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// EnvironmentConfigID is the ID for a EnvironmentConfigID object
type EnvironmentConfigID struct {
	modellib.ObjectID
}

// DeployableVectorIDPrefix represents the string that is attached to the beginning of these identifiers
const DeployableVectorIDPrefix = "deployableVector_"

// NewDeployableVectorID returns a new DeployableVectorID.
func NewDeployableVectorID() (DeployableVectorID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return DeployableVectorID{}, errors.Trace(err)
	}
	return DeployableVectorID{
		modellib.ObjectID{
			Prefix:  DeployableVectorIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyDeployableVectorID returns an empty initialized ID
func EmptyDeployableVectorID() DeployableVectorID {
	return DeployableVectorID{
		modellib.ObjectID{
			Prefix:  DeployableVectorIDPrefix,
			IsValid: false,
		},
	}
}

// ParseDeployableVectorID transforms an DeployableVectorID from it's string representation into the actual ID value
func ParseDeployableVectorID(s string) (DeployableVectorID, error) {
	id := EmptyDeployableVectorID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// DeployableVectorID is the ID for a DeployableVectorID object
type DeployableVectorID struct {
	modellib.ObjectID
}

// DeployableGroupIDPrefix represents the string that is attached to the beginning of these identifiers
const DeployableGroupIDPrefix = "deployableGroup_"

// NewDeployableGroupID returns a new DeployableGroupID.
func NewDeployableGroupID() (DeployableGroupID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return DeployableGroupID{}, errors.Trace(err)
	}
	return DeployableGroupID{
		modellib.ObjectID{
			Prefix:  DeployableGroupIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyDeployableGroupID returns an empty initialized ID
func EmptyDeployableGroupID() DeployableGroupID {
	return DeployableGroupID{
		modellib.ObjectID{
			Prefix:  DeployableGroupIDPrefix,
			IsValid: false,
		},
	}
}

// ParseDeployableGroupID transforms an DeployableGroupID from it's string representation into the actual ID value
func ParseDeployableGroupID(s string) (DeployableGroupID, error) {
	id := EmptyDeployableGroupID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// DeployableGroupID is the ID for a DeployableGroupID object
type DeployableGroupID struct {
	modellib.ObjectID
}

// DeployableIDPrefix represents the string that is attached to the beginning of these identifiers
const DeployableIDPrefix = "deployable_"

// NewDeployableID returns a new DeployableID.
func NewDeployableID() (DeployableID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return DeployableID{}, errors.Trace(err)
	}
	return DeployableID{
		modellib.ObjectID{
			Prefix:  DeployableIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// EmptyDeployableID returns an empty initialized ID
func EmptyDeployableID() DeployableID {
	return DeployableID{
		modellib.ObjectID{
			Prefix:  DeployableIDPrefix,
			IsValid: false,
		},
	}
}

// ParseDeployableID transforms an DeployableID from it's string representation into the actual ID value
func ParseDeployableID(s string) (DeployableID, error) {
	id := EmptyDeployableID()
	err := id.UnmarshalText([]byte(s))
	return id, errors.Trace(err)
}

// DeployableID is the ID for a DeployableID object
type DeployableID struct {
	modellib.ObjectID
}

// DeploymentStatus represents the type associated with the status column of the deployment table
type DeploymentStatus string

const (
	// DeploymentStatusPending represents the PENDING state of the status field on a deployment record
	DeploymentStatusPending DeploymentStatus = "PENDING"
	// DeploymentStatusInProgress represents the IN_PROGRESS state of the status field on a deployment record
	DeploymentStatusInProgress DeploymentStatus = "IN_PROGRESS"
	// DeploymentStatusComplete represents the COMPLETE state of the status field on a deployment record
	DeploymentStatusComplete DeploymentStatus = "COMPLETE"
	// DeploymentStatusFailed represents the FAILED state of the status field on a deployment record
	DeploymentStatusFailed DeploymentStatus = "FAILED"
)

// ParseDeploymentStatus converts a string into the correcponding enum value
func ParseDeploymentStatus(s string) (DeploymentStatus, error) {
	switch t := DeploymentStatus(strings.ToUpper(s)); t {
	case DeploymentStatusPending, DeploymentStatusInProgress, DeploymentStatusComplete, DeploymentStatusFailed:
		return t, nil
	}
	return DeploymentStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t DeploymentStatus) String() string {
	return string(t)
}

// Scan allows for scanning of DeploymentStatus from a database conforming to the sql.Scanner interface
func (t *DeploymentStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseDeploymentStatus(ts)
	case []byte:
		*t, err = ParseDeploymentStatus(string(ts))
	}
	return errors.Trace(err)
}

// DeploymentType represents the type associated with the type column of the deployment table
type DeploymentType string

const (
	// DeploymentTypeEcs represents the ECS state of the type field on a deployment record
	DeploymentTypeEcs DeploymentType = "ECS"
)

// ParseDeploymentType converts a string into the correcponding enum value
func ParseDeploymentType(s string) (DeploymentType, error) {
	switch t := DeploymentType(strings.ToUpper(s)); t {
	case DeploymentTypeEcs:
		return t, nil
	}
	return DeploymentType(""), errors.Trace(fmt.Errorf("Unknown type:%s", s))
}

func (t DeploymentType) String() string {
	return string(t)
}

// Scan allows for scanning of DeploymentType from a database conforming to the sql.Scanner interface
func (t *DeploymentType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseDeploymentType(ts)
	case []byte:
		*t, err = ParseDeploymentType(string(ts))
	}
	return errors.Trace(err)
}

// DeployableConfigStatus represents the type associated with the status column of the deployable_config table
type DeployableConfigStatus string

const (
	// DeployableConfigStatusActive represents the ACTIVE state of the status field on a deployable_config record
	DeployableConfigStatusActive DeployableConfigStatus = "ACTIVE"
	// DeployableConfigStatusDeprecated represents the DEPRECATED state of the status field on a deployable_config record
	DeployableConfigStatusDeprecated DeployableConfigStatus = "DEPRECATED"
)

// ParseDeployableConfigStatus converts a string into the correcponding enum value
func ParseDeployableConfigStatus(s string) (DeployableConfigStatus, error) {
	switch t := DeployableConfigStatus(strings.ToUpper(s)); t {
	case DeployableConfigStatusActive, DeployableConfigStatusDeprecated:
		return t, nil
	}
	return DeployableConfigStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t DeployableConfigStatus) String() string {
	return string(t)
}

// Scan allows for scanning of DeployableConfigStatus from a database conforming to the sql.Scanner interface
func (t *DeployableConfigStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseDeployableConfigStatus(ts)
	case []byte:
		*t, err = ParseDeployableConfigStatus(string(ts))
	}
	return errors.Trace(err)
}

// EnvironmentConfigStatus represents the type associated with the status column of the environment_config table
type EnvironmentConfigStatus string

const (
	// EnvironmentConfigStatusActive represents the ACTIVE state of the status field on a environment_config record
	EnvironmentConfigStatusActive EnvironmentConfigStatus = "ACTIVE"
	// EnvironmentConfigStatusDeprecated represents the DEPRECATED state of the status field on a environment_config record
	EnvironmentConfigStatusDeprecated EnvironmentConfigStatus = "DEPRECATED"
)

// ParseEnvironmentConfigStatus converts a string into the correcponding enum value
func ParseEnvironmentConfigStatus(s string) (EnvironmentConfigStatus, error) {
	switch t := EnvironmentConfigStatus(strings.ToUpper(s)); t {
	case EnvironmentConfigStatusActive, EnvironmentConfigStatusDeprecated:
		return t, nil
	}
	return EnvironmentConfigStatus(""), errors.Trace(fmt.Errorf("Unknown status:%s", s))
}

func (t EnvironmentConfigStatus) String() string {
	return string(t)
}

// Scan allows for scanning of EnvironmentConfigStatus from a database conforming to the sql.Scanner interface
func (t *EnvironmentConfigStatus) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseEnvironmentConfigStatus(ts)
	case []byte:
		*t, err = ParseEnvironmentConfigStatus(string(ts))
	}
	return errors.Trace(err)
}

// DeployableVectorSourceType represents the type associated with the source_type column of the deployable_vector table
type DeployableVectorSourceType string

const (
	// DeployableVectorSourceTypeBuild represents the BUILD state of the source_type field on a deployable_vector record
	DeployableVectorSourceTypeBuild DeployableVectorSourceType = "BUILD"
	// DeployableVectorSourceTypeEnvironmentID represents the ENVIRONMENT_ID state of the source_type field on a deployable_vector record
	DeployableVectorSourceTypeEnvironmentID DeployableVectorSourceType = "ENVIRONMENT_ID"
)

// ParseDeployableVectorSourceType converts a string into the correcponding enum value
func ParseDeployableVectorSourceType(s string) (DeployableVectorSourceType, error) {
	switch t := DeployableVectorSourceType(strings.ToUpper(s)); t {
	case DeployableVectorSourceTypeBuild, DeployableVectorSourceTypeEnvironmentID:
		return t, nil
	}
	return DeployableVectorSourceType(""), errors.Trace(fmt.Errorf("Unknown source_type:%s", s))
}

func (t DeployableVectorSourceType) String() string {
	return string(t)
}

// Scan allows for scanning of DeployableVectorSourceType from a database conforming to the sql.Scanner interface
func (t *DeployableVectorSourceType) Scan(src interface{}) error {
	var err error
	switch ts := src.(type) {
	case string:
		*t, err = ParseDeployableVectorSourceType(ts)
	case []byte:
		*t, err = ParseDeployableVectorSourceType(string(ts))
	}
	return errors.Trace(err)
}

// Deployable represents a deployable record
type Deployable struct {
	Created           time.Time
	Modified          time.Time
	ID                DeployableID
	DeployableGroupID DeployableGroupID
	Name              string
	Description       string
	GitURL            string
}

// DeployableUpdate represents the mutable aspects of a deployable record
type DeployableUpdate struct {
	DeployableGroupID DeployableGroupID
	Name              *string
	Description       *string
}

// DeployableGroup represents a deployable_group record
type DeployableGroup struct {
	Name        string
	Description string
	Created     time.Time
	Modified    time.Time
	ID          DeployableGroupID
}

// DeployableGroupUpdate represents the mutable aspects of a deployable_group record
type DeployableGroupUpdate struct {
	Description *string
	Name        *string
}

// DeployableVector represents a deployable_vector record
type DeployableVector struct {
	SourceEnvironmentID EnvironmentID
	TargetEnvironmentID EnvironmentID
	Created             time.Time
	ID                  DeployableVectorID
	DeployableID        DeployableID
	SourceType          DeployableVectorSourceType
}

// EnvironmentConfigValue represents a environment_config_value record
type EnvironmentConfigValue struct {
	EnvironmentConfigID EnvironmentConfigID
	Name                string
	Value               string
	Created             time.Time
}

// DeployableConfigValue represents a deployable_config_value record
type DeployableConfigValue struct {
	DeployableConfigID DeployableConfigID
	Name               string
	Value              string
	Created            time.Time
}

// DeployableConfig represents a deployable_config record
type DeployableConfig struct {
	Status        DeployableConfigStatus
	Created       time.Time
	ID            DeployableConfigID
	DeployableID  DeployableID
	EnvironmentID EnvironmentID
}

// DeployableConfigUpdate represents the mutable aspects of a deployable_config record
type DeployableConfigUpdate struct {
	DeployableID  DeployableID
	EnvironmentID EnvironmentID
	Status        *DeployableConfigStatus
}

// Deployment represents a deployment record
type Deployment struct {
	ID                 DeploymentID
	DeploymentNumber   uint64
	Type               DeploymentType
	Data               []byte
	Status             DeploymentStatus
	BuildNumber        string
	DeployableID       DeployableID
	EnvironmentID      EnvironmentID
	DeployableConfigID DeployableConfigID
	DeployableVectorID DeployableVectorID
	GitHash            string
	Created            time.Time
}

// EnvironmentConfig represents a environment_config record
type EnvironmentConfig struct {
	ID            EnvironmentConfigID
	EnvironmentID EnvironmentID
	Status        EnvironmentConfigStatus
	Created       time.Time
}

// EnvironmentConfigUpdate represents the mutable aspects of a environment_config record
type EnvironmentConfigUpdate struct {
	EnvironmentID EnvironmentID
	Status        *EnvironmentConfigStatus
}

// Environment represents a environment record
type Environment struct {
	Name              string
	Description       string
	IsProd            bool
	Created           time.Time
	Modified          time.Time
	ID                EnvironmentID
	DeployableGroupID DeployableGroupID
}

// EnvironmentUpdate represents the mutable aspects of a environment record
type EnvironmentUpdate struct {
	Description       *string
	DeployableGroupID DeployableGroupID
	Name              *string
}

// InsertDeployableVector inserts a deployable_vector record
func (d *dal) InsertDeployableVector(model *DeployableVector) (DeployableVectorID, error) {
	if !model.ID.IsValid {
		id, err := NewDeployableVectorID()
		if err != nil {
			return EmptyDeployableVectorID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO deployable_vector
          (id, deployable_id, source_type, source_environment_id, target_environment_id)
          VALUES (?, ?, ?, ?, ?)`, model.ID, model.DeployableID, model.SourceType.String(), model.SourceEnvironmentID, model.TargetEnvironmentID)
	if err != nil {
		return EmptyDeployableVectorID(), errors.Trace(err)
	}

	return model.ID, nil
}

// DeployableVector retrieves a deployable_vector record
func (d *dal) DeployableVector(id DeployableVectorID) (*DeployableVector, error) {
	row := d.db.QueryRow(
		selectDeployableVector+` WHERE id = ?`, id.Val)
	model, err := scanDeployableVector(row)
	return model, errors.Trace(err)
}

// DeployableVectorsForDeployable retrieves all deployable vector records for a given deployable id
func (d *dal) DeployableVectorsForDeployable(depID DeployableID) ([]*DeployableVector, error) {
	rows, err := d.db.Query(
		selectDeployableVector+` WHERE deployable_id = ?`, depID.Val)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*DeployableVector
	for rows.Next() {
		model, err := scanDeployableVector(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

// DeployableVectorsForDeployableAndSource retrieves all deployable vector records for a given deployable id and source
func (d *dal) DeployableVectorsForDeployableAndSource(depID DeployableID, source DeployableVectorSourceType) ([]*DeployableVector, error) {
	rows, err := d.db.Query(
		selectDeployableVector+` WHERE deployable_id = ? AND source_type = ?`, depID.Val, source.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*DeployableVector
	for rows.Next() {
		model, err := scanDeployableVector(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

// DeployableVectorsForDeployableAndSourceEnvironment retrieves all deployable vector records for a given deployable id and source en
func (d *dal) DeployableVectorsForDeployableAndSourceEnvironment(depID DeployableID, sourceEnv EnvironmentID) ([]*DeployableVector, error) {
	rows, err := d.db.Query(
		selectDeployableVector+` WHERE deployable_id = ? AND source_environment_id = ?`, depID.Val, sourceEnv.Val)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*DeployableVector
	for rows.Next() {
		model, err := scanDeployableVector(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

func (d *dal) DeployableVectorForDeployableSourceTarget(depID DeployableID, sourceType DeployableVectorSourceType, sourceEnvID, targetEnvID EnvironmentID) (*DeployableVector, error) {
	row := d.db.QueryRow(
		selectDeployableVector+` WHERE deployable_id = ? AND source_environment_id = ? AND target_environment_id = ? AND source_type = ?`, depID.Val, sourceEnvID, targetEnvID, sourceType.String())
	model, err := scanDeployableVector(row)
	return model, errors.Trace(err)
}

// DeleteDeployableVector deletes a deployable_vector record
func (d *dal) DeleteDeployableVector(id DeployableVectorID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM deployable_vector
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertDeployable inserts a deployable record
func (d *dal) InsertDeployable(model *Deployable) (DeployableID, error) {
	if !model.ID.IsValid {
		id, err := NewDeployableID()
		if err != nil {
			return EmptyDeployableID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO deployable
          (id, deployable_group_id, name, description, git_url)
          VALUES (?, ?, ?, ?, ?)`, model.ID, model.DeployableGroupID, model.Name, model.Description, model.GitURL)
	if err != nil {
		return EmptyDeployableID(), errors.Trace(err)
	}

	return model.ID, nil
}

// Deployable retrieves a deployable record
func (d *dal) Deployable(id DeployableID) (*Deployable, error) {
	row := d.db.QueryRow(
		selectDeployable+` WHERE id = ?`, id.Val)
	model, err := scanDeployable(row)
	return model, errors.Trace(err)
}

// DeployableForNameAndGroup retrieves a deployable record for the provided name and group id
func (d *dal) DeployableForNameAndGroup(name string, groupID DeployableGroupID) (*Deployable, error) {
	row := d.db.QueryRow(
		selectDeployable+` WHERE deployable_group_id = ? AND name = ?`, groupID, name)
	model, err := scanDeployable(row)
	return model, errors.Trace(err)
}

// DeployablesForGroup retrieves all deployable records for a given deployable group
func (d *dal) DeployablesForGroup(groupID DeployableGroupID) ([]*Deployable, error) {
	rows, err := d.db.Query(
		selectDeployable+` WHERE deployable_group_id = ?`, groupID.Val)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*Deployable
	for rows.Next() {
		model, err := scanDeployable(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

// UpdateDeployable updates the mutable aspects of a deployable record
func (d *dal) UpdateDeployable(id DeployableID, update *DeployableUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.DeployableGroupID.IsValid {
		args.Append("deployable_group_id", update.DeployableGroupID)
	}
	if update.Name != nil {
		args.Append("name", *update.Name)
	}
	if update.Description != nil {
		args.Append("description", *update.Description)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE deployable
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteDeployable deletes a deployable record
func (d *dal) DeleteDeployable(id DeployableID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM deployable
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertEnvironmentConfig inserts a environment_config record
func (d *dal) InsertEnvironmentConfig(model *EnvironmentConfig) (EnvironmentConfigID, error) {
	if !model.ID.IsValid {
		id, err := NewEnvironmentConfigID()
		if err != nil {
			return EmptyEnvironmentConfigID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT IGNORE INTO environment_config
          (id, environment_id, status)
          VALUES (?, ?, ?)`, model.ID, model.EnvironmentID, model.Status.String())
	if err != nil {
		return EmptyEnvironmentConfigID(), errors.Trace(err)
	}

	return model.ID, nil
}

// DeprecateActiveEnvironmentConfig updates the mutable aspects of a environment_config record
func (d *dal) DeprecateActiveEnvironmentConfig(envID EnvironmentID) (int64, error) {
	res, err := d.db.Exec(
		`UPDATE environment_config SET status = 'DEPRECATED' WHERE environment_id = ? AND status = 'ACTIVE'`, envID)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// EnvironmentConfig retrieves a environment_config record
func (d *dal) EnvironmentConfig(id EnvironmentConfigID) (*EnvironmentConfig, error) {
	row := d.db.QueryRow(
		selectEnvironmentConfig+` WHERE id = ?`, id.Val)
	model, err := scanEnvironmentConfig(row)
	return model, errors.Trace(err)
}

// EnvironmentConfig retrieves a environment_config record
func (d *dal) EnvironmentConfigsForStatus(environmentID EnvironmentID, status EnvironmentConfigStatus) ([]*EnvironmentConfig, error) {
	rows, err := d.db.Query(
		selectEnvironmentConfig+` WHERE environment_id = ? AND status = ?`, environmentID.Val, status.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*EnvironmentConfig
	for rows.Next() {
		model, err := scanEnvironmentConfig(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

// UpdateEnvironmentConfig updates the mutable aspects of a environment_config record
func (d *dal) UpdateEnvironmentConfig(id EnvironmentConfigID, update *EnvironmentConfigUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.EnvironmentID.IsValid {
		args.Append("environment_id", update.EnvironmentID)
	}
	if update.Status != nil {
		args.Append("status", *update.Status)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE environment_config
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteEnvironmentConfig deletes a environment_config record
func (d *dal) DeleteEnvironmentConfig(id EnvironmentConfigID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM environment_config
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// EnvironmentConfigValues retrieves all environment_config_value records for the provided config id
func (d *dal) EnvironmentConfigValues(id EnvironmentConfigID) ([]*EnvironmentConfigValue, error) {
	rows, err := d.db.Query(
		selectEnvironmentConfigValue+` WHERE environment_config_id = ?`, id.Val)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*EnvironmentConfigValue
	for rows.Next() {
		model, err := scanEnvironmentConfigValue(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

// InsertEnvironmentConfigValue inserts a environment_config_value record
func (d *dal) InsertEnvironmentConfigValue(model *EnvironmentConfigValue) error {
	_, err := d.db.Exec(
		`INSERT INTO environment_config_value
          (environment_config_id, name, value)
          VALUES (?, ?, ?)`, model.EnvironmentConfigID, model.Name, model.Value)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// InsertEnvironmentConfigValues inserts a set of environment_config_value records
func (d *dal) InsertEnvironmentConfigValues(models []*EnvironmentConfigValue) error {
	in := dbutil.MySQLMultiInsert(len(models))
	for _, m := range models {
		in.Append(m.EnvironmentConfigID, m.Name, m.Value)
	}
	_, err := d.db.Exec(
		`INSERT INTO environment_config_value
          (environment_config_id, name, value)
          VALUES `+in.Query(), in.Values()...)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// InsertDeployableConfig inserts a deployable_config record
func (d *dal) InsertDeployableConfig(model *DeployableConfig) (DeployableConfigID, error) {
	if !model.ID.IsValid {
		id, err := NewDeployableConfigID()
		if err != nil {
			return EmptyDeployableConfigID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT IGNORE INTO deployable_config
          (id, deployable_id, environment_id, status)
          VALUES (?, ?, ?, ?)`, model.ID, model.DeployableID, model.EnvironmentID, model.Status.String())
	if err != nil {
		return EmptyDeployableConfigID(), errors.Trace(err)
	}

	return model.ID, nil
}

// DeprecateActiveDeployableConfig updates the mutable aspects of a environment_config record
func (d *dal) DeprecateActiveDeployableConfig(deployableID DeployableID, envID EnvironmentID) (int64, error) {
	res, err := d.db.Exec(
		`UPDATE deployable_config SET status = 'DEPRECATED' WHERE environment_id = ? AND deployable_id = ? AND status = 'ACTIVE'`, envID, deployableID)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertDeployableConfigValues inserts a set of deployable_config_value records
func (d *dal) InsertDeployableConfigValues(models []*DeployableConfigValue) error {
	in := dbutil.MySQLMultiInsert(len(models))
	for _, m := range models {
		in.Append(m.DeployableConfigID, m.Name, m.Value)
	}
	_, err := d.db.Exec(
		`INSERT INTO deployable_config_value
          (deployable_config_id, name, value)
          VALUES `+in.Query(), in.Values()...)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// DeployableConfigValues retrieves all deployable_config_value records for the provided config id
func (d *dal) DeployableConfigValues(id DeployableConfigID) ([]*DeployableConfigValue, error) {
	rows, err := d.db.Query(
		selectDeployableConfigValue+` WHERE deployable_config_id = ?`, id.Val)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*DeployableConfigValue
	for rows.Next() {
		model, err := scanDeployableConfigValue(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

// DeployableConfig retrieves a deployable_config record
func (d *dal) DeployableConfig(id DeployableConfigID) (*DeployableConfig, error) {
	row := d.db.QueryRow(
		selectDeployableConfig+` WHERE id = ?`, id.Val)
	model, err := scanDeployableConfig(row)
	return model, errors.Trace(err)
}

// Deployable retrieves a deployable_config record
func (d *dal) DeployableConfigsForStatus(depID DeployableID, environmentID EnvironmentID, status DeployableConfigStatus) ([]*DeployableConfig, error) {
	rows, err := d.db.Query(
		selectDeployableConfig+` WHERE deployable_id = ? AND environment_id = ? AND status = ?`, depID.Val, environmentID.Val, status.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*DeployableConfig
	for rows.Next() {
		model, err := scanDeployableConfig(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

// UpdateDeployableConfig updates the mutable aspects of a deployable_config record
func (d *dal) UpdateDeployableConfig(id DeployableConfigID, update *DeployableConfigUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.DeployableID.IsValid {
		args.Append("deployable_id", update.DeployableID)
	}
	if update.EnvironmentID.IsValid {
		args.Append("environment_id", update.EnvironmentID)
	}
	if update.Status != nil {
		args.Append("status", *update.Status)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE deployable_config
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteDeployableConfig deletes a deployable_config record
func (d *dal) DeleteDeployableConfig(id DeployableConfigID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM deployable_config
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertDeployment inserts a deployment record
func (d *dal) InsertDeployment(model *Deployment) (DeploymentID, error) {
	if !model.ID.IsValid {
		id, err := NewDeploymentID()
		if err != nil {
			return EmptyDeploymentID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO deployment
          (id, type, data, status, build_number, deployable_id, environment_id, deployable_config_id, deployable_vector_id, git_hash)
          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, model.ID, model.Type.String(), model.Data, model.Status.String(), model.BuildNumber, model.DeployableID, model.EnvironmentID, model.DeployableConfigID, model.DeployableVectorID, model.GitHash)
	if err != nil {
		return EmptyDeploymentID(), errors.Trace(err)
	}

	return model.ID, nil
}

// Deployment retrieves a deployment record
func (d *dal) Deployment(id DeploymentID) (*Deployment, error) {
	row := d.db.QueryRow(
		selectDeployment+` WHERE id = ?`, id.Val)
	model, err := scanDeployment(row)
	return model, errors.Trace(err)
}

func (d *dal) ActiveDeployment(depID DeployableID, envID EnvironmentID) (*Deployment, error) {
	row := d.db.QueryRow(
		selectDeployment+` WHERE deployable_id = ? AND environment_id = ? AND status = 'COMPLETE' ORDER BY deployment_number DESC LIMIT 1`, depID, envID)
	model, err := scanDeployment(row)
	return model, errors.Trace(err)
}

func (d *dal) DeploymentsForDeploymentGroup(depID DeployableGroupID, envID EnvironmentID, buildNumber string) ([]*Deployment, error) {
	rows, err := d.db.Query(
		selectDeployment+` WHERE status = 'COMPLETE' environment_id = ? AND build_number = ? AND deployable_id IN (SELECT id FROM deployable WHERE deployable_group_id = ?);`, envID.Val, buildNumber, depID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*Deployment
	// do a little janky deduping incase multiple instances of the same build number have been deployed. Can likely fix this in the query somehow
	foundDeployables := make(map[uint64]struct{})
	for rows.Next() {
		model, err := scanDeployment(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if _, ok := foundDeployables[model.DeployableID.Val]; !ok {
			models = append(models, model)
			foundDeployables[model.DeployableID.Val] = struct{}{}
		}
	}
	return models, errors.Trace(err)
}

// TODO: Lock retrieval of pending deployments on a deployable if there is one IN_PROGRESS
// NextPendingDeployment retrieves the next pending deployment to execute
func (d *dal) NextPendingDeployment() (*Deployment, error) {
	row := d.db.QueryRow(
		selectDeployment + ` WHERE status = 'PENDING' ORDER BY deployment_number ASC LIMIT 1 FOR UPDATE`)
	model, err := scanDeployment(row)
	return model, errors.Trace(err)
}

// SetDeploymentStatus sets the specified deployment to the provided status
func (d *dal) SetDeploymentStatus(id DeploymentID, s DeploymentStatus) error {
	_, err := d.db.Exec(`UPDATE deployment SET status = ? WHERE id = ?`, s.String(), id)
	return errors.Trace(err)
}

// Deployments retrieves all deployment records for the provided deployable id
func (d *dal) Deployments(depID DeployableID) ([]*Deployment, error) {
	rows, err := d.db.Query(
		selectDeployment+` WHERE deployable_id = ?`, depID.Val)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*Deployment
	for rows.Next() {
		model, err := scanDeployment(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

// DeploymentsForStatus retrieves all deployment records for the provided deployable id and status
func (d *dal) DeploymentsForStatus(depID DeployableID, status DeploymentStatus) ([]*Deployment, error) {
	rows, err := d.db.Query(
		selectDeployment+` WHERE deployable_id = ? AND status = ?`, depID.Val, status.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*Deployment
	for rows.Next() {
		model, err := scanDeployment(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

// DeleteDeployment deletes a deployment record
func (d *dal) DeleteDeployment(id DeploymentID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM deployment
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertDeployableGroup inserts a deployable_group record
func (d *dal) InsertDeployableGroup(model *DeployableGroup) (DeployableGroupID, error) {
	if !model.ID.IsValid {
		id, err := NewDeployableGroupID()
		if err != nil {
			return EmptyDeployableGroupID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO deployable_group
          (name, description, id)
          VALUES (?, ?, ?)`, model.Name, model.Description, model.ID)
	if err != nil {
		return EmptyDeployableGroupID(), errors.Trace(err)
	}

	return model.ID, nil
}

// DeployableGroup retrieves a deployable_group record
func (d *dal) DeployableGroup(id DeployableGroupID) (*DeployableGroup, error) {
	row := d.db.QueryRow(
		selectDeployableGroup+` WHERE id = ?`, id.Val)
	model, err := scanDeployableGroup(row)
	return model, errors.Trace(err)
}

// DeployableGroups retrieves all deployable_group records
func (d *dal) DeployableGroups() ([]*DeployableGroup, error) {
	rows, err := d.db.Query(selectDeployableGroup)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*DeployableGroup
	for rows.Next() {
		model, err := scanDeployableGroup(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

// DeployableGroup retrieves a deployable_group record with the provided name
func (d *dal) DeployableGroupForName(name string) (*DeployableGroup, error) {
	row := d.db.QueryRow(
		selectDeployableGroup+` WHERE name = ?`, name)
	model, err := scanDeployableGroup(row)
	return model, errors.Trace(err)
}

// UpdateDeployableGroup updates the mutable aspects of a deployable_group record
func (d *dal) UpdateDeployableGroup(id DeployableGroupID, update *DeployableGroupUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.Name != nil {
		args.Append("name", *update.Name)
	}
	if update.Description != nil {
		args.Append("description", *update.Description)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE deployable_group
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteDeployableGroup deletes a deployable_group record
func (d *dal) DeleteDeployableGroup(id DeployableGroupID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM deployable_group
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertEnvironment inserts a environment record
func (d *dal) InsertEnvironment(model *Environment) (EnvironmentID, error) {
	if !model.ID.IsValid {
		id, err := NewEnvironmentID()
		if err != nil {
			return EmptyEnvironmentID(), errors.Trace(err)
		}
		model.ID = id
	}
	_, err := d.db.Exec(
		`INSERT INTO environment
          (description, id, deployable_group_id, name, is_prod)
          VALUES (?, ?, ?, ?, ?)`, model.Description, model.ID, model.DeployableGroupID, model.Name, model.IsProd)
	if err != nil {
		return EmptyEnvironmentID(), errors.Trace(err)
	}

	return model.ID, nil
}

// Environment retrieves a environment record
func (d *dal) Environment(id EnvironmentID) (*Environment, error) {
	row := d.db.QueryRow(
		selectEnvironment+` WHERE id = ?`, id.Val)
	model, err := scanEnvironment(row)
	return model, errors.Trace(err)
}

// EnvironmentsForGroup retrieves all environment records for a given deployable group
func (d *dal) EnvironmentsForGroup(groupID DeployableGroupID) ([]*Environment, error) {
	rows, err := d.db.Query(
		selectEnvironment+` WHERE deployable_group_id = ?`, groupID.Val)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var models []*Environment
	for rows.Next() {
		model, err := scanEnvironment(rows)
		if err != nil {
			return nil, errors.Trace(err)
		}
		models = append(models, model)
	}
	return models, errors.Trace(err)
}

// Environment retrieves a environment record
func (d *dal) EnvironmentForNameAndGroup(name string, groupID DeployableGroupID) (*Environment, error) {
	row := d.db.QueryRow(
		selectEnvironment+` WHERE deployable_group_id = ? AND name = ?`, groupID, name)
	model, err := scanEnvironment(row)
	return model, errors.Trace(err)
}

// UpdateEnvironment updates the mutable aspects of a environment record
func (d *dal) UpdateEnvironment(id EnvironmentID, update *EnvironmentUpdate) (int64, error) {
	args := dbutil.MySQLVarArgs()
	if update.DeployableGroupID.IsValid {
		args.Append("deployable_group_id", update.DeployableGroupID)
	}
	if update.Name != nil {
		args.Append("name", *update.Name)
	}
	if update.Description != nil {
		args.Append("description", *update.Description)
	}
	if args.IsEmpty() {
		return 0, nil
	}

	res, err := d.db.Exec(
		`UPDATE environment
          SET `+args.ColumnsForUpdate()+` WHERE id = ?`, append(args.Values(), id.Val)...)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// DeleteEnvironment deletes a environment record
func (d *dal) DeleteEnvironment(id EnvironmentID) (int64, error) {
	res, err := d.db.Exec(
		`DELETE FROM environment
          WHERE id = ?`, id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	aff, err := res.RowsAffected()
	return aff, errors.Trace(err)
}

// InsertDeployableConfigValue inserts a deployable_config_value record
func (d *dal) InsertDeployableConfigValue(model *DeployableConfigValue) error {
	_, err := d.db.Exec(
		`INSERT INTO deployable_config_value
          (deployable_config_id, name, value)
          VALUES (?, ?, ?)`, model.DeployableConfigID, model.Name, model.Value)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

const selectDeployableVector = `
    SELECT deployable_vector.deployable_id, deployable_vector.source_type, deployable_vector.source_environment_id, deployable_vector.target_environment_id, deployable_vector.created, deployable_vector.id
      FROM deployable_vector`

func scanDeployableVector(row dbutil.Scanner) (*DeployableVector, error) {
	var m DeployableVector
	m.DeployableID = EmptyDeployableID()
	m.SourceEnvironmentID = EmptyEnvironmentID()
	m.TargetEnvironmentID = EmptyEnvironmentID()
	m.ID = EmptyDeployableVectorID()

	err := row.Scan(&m.DeployableID, &m.SourceType, &m.SourceEnvironmentID, &m.TargetEnvironmentID, &m.Created, &m.ID)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	}
	return &m, errors.Trace(err)
}

const selectDeployableGroup = `
    SELECT deployable_group.description, deployable_group.created, deployable_group.modified, deployable_group.id, deployable_group.name
      FROM deployable_group`

func scanDeployableGroup(row dbutil.Scanner) (*DeployableGroup, error) {
	var m DeployableGroup
	m.ID = EmptyDeployableGroupID()

	err := row.Scan(&m.Description, &m.Created, &m.Modified, &m.ID, &m.Name)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	}
	return &m, errors.Trace(err)
}

const selectDeployable = `
    SELECT deployable.id, deployable.deployable_group_id, deployable.name, deployable.description, deployable.git_url, deployable.created, deployable.modified
      FROM deployable`

func scanDeployable(row dbutil.Scanner) (*Deployable, error) {
	var m Deployable
	m.ID = EmptyDeployableID()
	m.DeployableGroupID = EmptyDeployableGroupID()

	err := row.Scan(&m.ID, &m.DeployableGroupID, &m.Name, &m.Description, &m.GitURL, &m.Created, &m.Modified)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	}
	return &m, errors.Trace(err)
}

const selectEnvironment = `
    SELECT environment.id, environment.deployable_group_id, environment.name, environment.description, environment.created, environment.modified, environment.is_prod
      FROM environment`

func scanEnvironment(row dbutil.Scanner) (*Environment, error) {
	var m Environment
	m.ID = EmptyEnvironmentID()
	m.DeployableGroupID = EmptyDeployableGroupID()

	err := row.Scan(&m.ID, &m.DeployableGroupID, &m.Name, &m.Description, &m.Created, &m.Modified, &m.IsProd)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	}
	return &m, errors.Trace(err)
}

const selectEnvironmentConfig = `
    SELECT environment_config.id, environment_config.environment_id, environment_config.status, environment_config.created
      FROM environment_config`

func scanEnvironmentConfig(row dbutil.Scanner) (*EnvironmentConfig, error) {
	var m EnvironmentConfig
	m.ID = EmptyEnvironmentConfigID()
	m.EnvironmentID = EmptyEnvironmentID()

	err := row.Scan(&m.ID, &m.EnvironmentID, &m.Status, &m.Created)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	}
	return &m, errors.Trace(err)
}

const selectEnvironmentConfigValue = `
    SELECT environment_config_value.environment_config_id, environment_config_value.name, environment_config_value.value, environment_config_value.created
      FROM environment_config_value`

func scanEnvironmentConfigValue(row dbutil.Scanner) (*EnvironmentConfigValue, error) {
	var m EnvironmentConfigValue
	m.EnvironmentConfigID = EmptyEnvironmentConfigID()

	err := row.Scan(&m.EnvironmentConfigID, &m.Name, &m.Value, &m.Created)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	}
	return &m, errors.Trace(err)
}

const selectDeployableConfig = `
    SELECT deployable_config.deployable_id, deployable_config.environment_id, deployable_config.status, deployable_config.created, deployable_config.id
      FROM deployable_config`

func scanDeployableConfig(row dbutil.Scanner) (*DeployableConfig, error) {
	var m DeployableConfig
	m.DeployableID = EmptyDeployableID()
	m.EnvironmentID = EmptyEnvironmentID()
	m.ID = EmptyDeployableConfigID()

	err := row.Scan(&m.DeployableID, &m.EnvironmentID, &m.Status, &m.Created, &m.ID)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	}
	return &m, errors.Trace(err)
}

const selectDeployableConfigValue = `
    SELECT deployable_config_value.deployable_config_id, deployable_config_value.name, deployable_config_value.value, deployable_config_value.created
      FROM deployable_config_value`

func scanDeployableConfigValue(row dbutil.Scanner) (*DeployableConfigValue, error) {
	var m DeployableConfigValue
	m.DeployableConfigID = EmptyDeployableConfigID()

	err := row.Scan(&m.DeployableConfigID, &m.Name, &m.Value, &m.Created)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	}
	return &m, errors.Trace(err)
}

const selectDeployment = `
    SELECT deployment.id, deployment.deployment_number, deployment.type, deployment.data, deployment.status, deployment.build_number, deployment.deployable_id, deployment.environment_id, deployment.deployable_config_id, deployment.deployable_vector_id, deployment.git_hash, deployment.created
      FROM deployment`

func scanDeployment(row dbutil.Scanner) (*Deployment, error) {
	var m Deployment
	m.ID = EmptyDeploymentID()
	m.DeployableID = EmptyDeployableID()
	m.EnvironmentID = EmptyEnvironmentID()
	m.DeployableConfigID = EmptyDeployableConfigID()
	m.DeployableVectorID = EmptyDeployableVectorID()

	err := row.Scan(&m.ID, &m.DeploymentNumber, &m.Type, &m.Data, &m.Status, &m.BuildNumber, &m.DeployableID, &m.EnvironmentID, &m.DeployableConfigID, &m.DeployableVectorID, &m.GitHash, &m.Created)
	if err == sql.ErrNoRows {
		return nil, errors.Trace(ErrNotFound)
	}
	return &m, errors.Trace(err)
}
