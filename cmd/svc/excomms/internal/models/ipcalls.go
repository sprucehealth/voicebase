package models

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/svc/excomms"
)

// IPCallType is the type of IP call (audio or video)
type IPCallType string

const (
	// IPCallTypeVideo signifies a video call
	IPCallTypeVideo IPCallType = "VIDEO"
	// IPCallTypeAudio signifies an audio call
	IPCallTypeAudio IPCallType = "AUDIO"
)

// Scan implements sql.Scanner and expects src to be nil or of type string or []byte
func (t *IPCallType) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch src := src.(type) {
	case []byte:
		*t = IPCallType(string(src))
	case string:
		*t = IPCallType(src)
	default:
		return errors.Trace(fmt.Errorf("unsupported type for IPCallType.Scan: %T", src))
	}
	if !t.Valid() {
		return errors.Trace(fmt.Errorf("'%s' is not a valid IPCallType", string(*t)))
	}
	return nil
}

// Value implements sql/driver.Valuer to allow an ObjectID to be used in an sql query
func (t IPCallType) Value() (driver.Value, error) {
	if !t.Valid() {
		return nil, errors.Trace(fmt.Errorf("'%s' is not a valid IPCallType", string(t)))
	}
	return string(t), nil
}

// Valid returns true iff the value if the value is valid
func (t IPCallType) Valid() bool {
	switch t {
	case IPCallTypeVideo, IPCallTypeAudio:
		return true
	}
	return false
}

// IPCallParticipantRole is a participant's role in an IP call
type IPCallParticipantRole string

const (
	// IPCallParticipantRoleCaller is the person that placed the call
	IPCallParticipantRoleCaller IPCallParticipantRole = "CALLER"
	// IPCallParticipantRoleRecipient is a person receiving the call
	IPCallParticipantRoleRecipient IPCallParticipantRole = "RECIPIENT"
)

// Scan implements sql.Scanner and expects src to be nil or of type string or []byte
func (r *IPCallParticipantRole) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch src := src.(type) {
	case []byte:
		*r = IPCallParticipantRole(string(src))
	case string:
		*r = IPCallParticipantRole(src)
	default:
		return errors.Errorf("unsupported type for IPCallParticipantRole.Scan: %T", src)
	}
	if !r.Valid() {
		return errors.Errorf("'%s' is not a valid IPCallParticipantRole", string(*r))
	}
	return nil
}

// Value implements sql/driver.Valuer to allow an ObjectID to be used in an sql query
func (r IPCallParticipantRole) Value() (driver.Value, error) {
	if !r.Valid() {
		return nil, errors.Errorf("'%s' is not a valid IPCallParticipantRole", string(r))
	}
	return string(r), nil
}

// Valid returns true iff the value if the value is valid
func (r IPCallParticipantRole) Valid() bool {
	switch r {
	case IPCallParticipantRoleCaller, IPCallParticipantRoleRecipient:
		return true
	}
	return false
}

// IPCallState is the state of an IP call participant
type IPCallState string

const (
	// IPCallStatePending means the call was initiated but is still pending any further activity from this participant
	IPCallStatePending IPCallState = "PENDING"
	// IPCallStateAccepted means the participant has accepted the call
	IPCallStateAccepted IPCallState = "ACCEPTED"
	// IPCallStateDeclined means the participant has decliedn the call
	IPCallStateDeclined IPCallState = "DECLINED"
	// IPCallStateConnected means the participant has successfully connected to the call
	IPCallStateConnected IPCallState = "CONNECTED"
	// IPCallStateFailed means the participant failed to connect to the call
	IPCallStateFailed IPCallState = "FAILED"
	// IPCallStateCompleted means the participant has completed the call
	IPCallStateCompleted IPCallState = "COMPLETED"
)

// Ptr returns a pointer to the call state (for use in dal updates)
func (s IPCallState) Ptr() *IPCallState {
	return &s
}

// Scan implements sql.Scanner and expects src to be nil or of type string or []byte
func (s *IPCallState) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch src := src.(type) {
	case []byte:
		*s = IPCallState(string(src))
	case string:
		*s = IPCallState(src)
	default:
		return errors.Errorf("unsupported type for IPCallState.Scan: %T", src)
	}
	if !s.Valid() {
		return errors.Errorf("'%s' is not a valid IPCallState", string(*s))
	}
	return nil
}

// Value implements sql/driver.Valuer to allow an ObjectID to be used in an sql query
func (s IPCallState) Value() (driver.Value, error) {
	if !s.Valid() {
		return nil, errors.Errorf("'%s' is not a valid IPCallState", string(s))
	}
	return string(s), nil
}

// Valid returns true iff the value if the value is valid
func (s IPCallState) Valid() bool {
	switch s {
	case IPCallStatePending, IPCallStateAccepted,
		IPCallStateDeclined, IPCallStateConnected,
		IPCallStateFailed, IPCallStateCompleted:
		return true
	}
	return false
}

// Pending returns true for any state that represents a non-connected and non-terminal call
func (s IPCallState) Pending() bool {
	switch s {
	case IPCallStateCompleted, IPCallStateDeclined, IPCallStateFailed, IPCallStateConnected:
		return false
	}
	return true
}

// Terminal returns true for any state that represents the end of a call
func (s IPCallState) Terminal() bool {
	switch s {
	case IPCallStateCompleted, IPCallStateDeclined, IPCallStateFailed:
		return true
	}
	return false
}

// NetworkType signifies a type of network connection
type NetworkType string

const (
	// NetworkTypeUnknown is an unknown network connection type
	NetworkTypeUnknown NetworkType = "UNKNOWN"
	// NetworkTypeCellular is a cellular network
	NetworkTypeCellular NetworkType = "CELLULAR"
	// NetworkTypeWiFi is a wi-fi network
	NetworkTypeWiFi NetworkType = "WIFI"
)

// Ptr returns a pointer to the network type (for use in dal updates)
func (s NetworkType) Ptr() *NetworkType {
	return &s
}

// Scan implements sql.Scanner and expects src to be nil or of type string or []byte
func (s *NetworkType) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch src := src.(type) {
	case []byte:
		*s = NetworkType(string(src))
	case string:
		*s = NetworkType(src)
	default:
		return errors.Trace(fmt.Errorf("unsupported type for NetworkType.Scan: %T", src))
	}
	if !s.Valid() {
		return errors.Trace(fmt.Errorf("'%s' is not a valid NetworkType", string(*s)))
	}
	return nil
}

// Value implements sql/driver.Valuer to allow an ObjectID to be used in an sql query
func (s NetworkType) Value() (driver.Value, error) {
	if !s.Valid() {
		return nil, errors.Trace(fmt.Errorf("'%s' is not a valid NetworkType", string(s)))
	}
	return string(s), nil
}

// Valid returns true iff the value if the value is valid
func (s NetworkType) Valid() bool {
	switch s {
	case NetworkTypeUnknown, NetworkTypeCellular, NetworkTypeWiFi:
		return true
	}
	return false
}

// IPCallID is the ID for an IPCall
type IPCallID struct{ model.ObjectID }

// NewIPCallID generates a new unique random IPCallID
func NewIPCallID() (IPCallID, error) {
	id, err := idgen.NewID()
	if err != nil {
		return IPCallID{}, errors.Trace(err)
	}
	return IPCallID{
		model.ObjectID{
			Prefix:  excomms.IPCallIDPrefix,
			Val:     id,
			IsValid: true,
		},
	}, nil
}

// ParseIPCallID parses an IPCallID from the string version
func ParseIPCallID(s string) (IPCallID, error) {
	t := EmptyIPCallID()
	err := t.UnmarshalText([]byte(s))
	return t, errors.Trace(err)
}

// EmptyIPCallID returns an new IPCallID that can be used when deserializing from the database
func EmptyIPCallID() IPCallID {
	return IPCallID{
		model.ObjectID{
			Prefix:  excomms.IPCallIDPrefix,
			IsValid: false,
		},
	}
}

// IPCall is a video or audio call placed over the network (non-POTS call)
type IPCall struct {
	ID            IPCallID
	Type          IPCallType
	Pending       bool
	InitiatedTime time.Time
	ConnectedTime *time.Time
	Participants  []*IPCallParticipant
}

// Active returns true iff no participant is in a terminal state
func (c *IPCall) Active() bool {
	for _, p := range c.Participants {
		if p.State.Terminal() {
			return false
		}
	}
	return true
}

// Connected returns true if the call is currently connected
func (c *IPCall) Connected() bool {
	return c.ConnectedTime != nil && c.Active()
}

// IPCallParticipant is a participant in an IP call
type IPCallParticipant struct {
	AccountID   string
	EntityID    string
	Identity    string
	Role        IPCallParticipantRole
	State       IPCallState
	NetworkType NetworkType
}
