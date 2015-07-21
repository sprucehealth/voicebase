package common

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

var (
	// PromotionTypes is a global value that contains a mapping beteen type names and the concrete implementaion into which they should be cast
	PromotionTypes = make(map[string]reflect.Type)
)

// PromotionGroup represents a logical grouping for promotions to provide artificial limitations
type PromotionGroup struct {
	ID               int64
	Name             string
	MaxAllowedPromos int
}

// PromoCode represents the text code that maps to a promotin
type PromoCode struct {
	ID         int64
	Code       string
	IsReferral bool
}

// Promotion represents the information that makes up a user facing promotion of some or not value.
type Promotion struct {
	Code    string
	CodeID  int64
	Data    Typed
	Group   string
	Expires *time.Time
	Created time.Time
}

// AccountPromotion represents a promotion that has been associated with an account
type AccountPromotion struct {
	ID        int64
	AccountID int64
	Status    PromotionStatus
	Type      string
	Code      string
	CodeID    int64
	Group     string
	GroupID   int64
	Expires   *time.Time
	Created   time.Time
	Data      Typed
}

// AccountPromotionByCreation facilitates the sorting of promotions by creation date
type AccountPromotionByCreation []*AccountPromotion

func (a AccountPromotionByCreation) Len() int {
	return len(a)
}

func (a AccountPromotionByCreation) Less(i, j int) bool {
	return a[i].Created.Unix() < a[j].Created.Unix()
}

func (a AccountPromotionByCreation) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// ReferralProgramTemplate represents a base referral_program that is cloned when assigned to an account
type ReferralProgramTemplate struct {
	ID              int64
	Role            string
	RoleTypeID      int64
	Data            Typed
	Created         time.Time
	Status          ReferralProgramStatus
	PromotionCodeID *int64
}

// ReferralProgramTemplateUpdate represents the data available for an UPDATE of a ReferralProgramTemplate record
type ReferralProgramTemplateUpdate struct {
	ID     int64
	Status ReferralProgramStatus
}

type ReferralProgram struct {
	TemplateID               *int64
	AccountID                int64
	Code                     string
	CodeID                   int64
	Data                     Typed
	Created                  time.Time
	Status                   ReferralProgramStatus
	PromotionReferralRouteID *int64
}

type ReferralTrackingEntry struct {
	CodeID             int64
	ClaimingAccountID  int64
	ReferringAccountID int64
	Created            time.Time
	Status             ReferralTrackingStatus
}

type PromotionReferralRoute struct {
	ID              int64
	PromotionCodeID int64
	Created         time.Time
	Modified        time.Time
	Priority        int
	Lifecycle       PRRLifecycle
	Gender          *PRRGender
	AgeLower        *int
	AgeUpper        *int
	State           *string
	Pharmacy        *string
}

type PromotionReferralRouteUpdate struct {
	ID        int64
	Lifecycle PRRLifecycle
}

type AccountCredit struct {
	AccountID int64
	Credit    int
}

type ParkedAccount struct {
	ID             int64
	CodeID         int64
	Code           string
	IsReferral     bool
	Email          string
	State          string
	AccountCreated bool
}

type PRRLifecycle string

const (
	PRRLifecycleActive     PRRLifecycle = "ACTIVE"
	PRRLifecycleNoNewUsers PRRLifecycle = "NO_NEW_USERS"
	PRRLifecycleDeprecated PRRLifecycle = "DEPRECATED"
)

func (p PRRLifecycle) String() string {
	return string(p)
}

func GetPRRLifecycle(s string) (PRRLifecycle, error) {
	switch rs := PRRLifecycle(s); rs {
	case PRRLifecycleActive, PRRLifecycleNoNewUsers, PRRLifecycleDeprecated:
		return rs, nil
	}

	return PRRLifecycle(""), fmt.Errorf("%s is not a PRRLifecycle", s)
}

func (p *PRRLifecycle) Scan(src interface{}) error {

	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into PRRLifecycle when string expected", src)
	}

	var err error
	*p, err = GetPRRLifecycle(string(str))

	return err
}

type PRRGender string

const (
	PRRGenderMale   PRRGender = "M"
	PRRGenderFemale PRRGender = "F"
)

func (p PRRGender) String() string {
	return string(p)
}

func GetPRRGender(s string) (PRRGender, error) {
	switch rs := PRRGender(s); rs {
	case PRRGenderMale, PRRGenderFemale:
		return rs, nil
	}

	switch strings.ToLower(s) {
	case "male":
		return PRRGenderMale, nil
	case "female":
		return PRRGenderFemale, nil
	}

	return PRRGender(""), fmt.Errorf("%s is not a PRRGender", s)
}

func (p *PRRGender) Scan(src interface{}) error {
	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into PRRGender when string expected", src)
	}

	var err error
	*p, err = GetPRRGender(string(str))

	return err
}

type ReferralProgramStatus string
type ReferralProgramStatusList []string

const (
	RSActive   ReferralProgramStatus = "Active"
	RSInactive ReferralProgramStatus = "Inactive"
	RSDefault  ReferralProgramStatus = "Default"
)

func (p ReferralProgramStatus) String() string {
	return string(p)
}

func GetReferralProgramStatus(s string) (ReferralProgramStatus, error) {
	switch rs := ReferralProgramStatus(s); rs {
	case RSActive, RSInactive, RSDefault:
		return rs, nil
	}

	return ReferralProgramStatus(""), fmt.Errorf("%s is not a ReferralProgramStatus", s)
}

func (p *ReferralProgramStatus) Scan(src interface{}) error {

	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into ReferralProgramStatus when string expected", src)
	}

	var err error
	*p, err = GetReferralProgramStatus(string(str))

	return err
}

type PromotionStatus string

const (
	PSPending   PromotionStatus = "Pending"
	PSCompleted PromotionStatus = "Completed"
	PSExpired   PromotionStatus = "Expired"
	PSDeleted   PromotionStatus = "Deleted"
)

func (p PromotionStatus) String() string {
	return string(p)
}

func GetPromotionStatus(s string) (PromotionStatus, error) {
	switch ps := PromotionStatus(s); ps {
	case PSPending, PSCompleted, PSExpired:
		return ps, nil
	}

	return PromotionStatus(""), fmt.Errorf("%s is not a PromotionStatus", s)
}

func (p *PromotionStatus) Scan(src interface{}) error {

	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into PromotionStatus when string expected", src)
	}

	var err error
	*p, err = GetPromotionStatus(string(str))

	return err
}

type ReferralTrackingStatus string

const (
	RTSPending   ReferralTrackingStatus = "Pending"
	RTSCompleted ReferralTrackingStatus = "Completed"
)

func (r ReferralTrackingStatus) String() string {
	return string(r)
}

func GetReferralTrackingStatus(s string) (ReferralTrackingStatus, error) {
	switch rt := ReferralTrackingStatus(s); rt {
	case RTSPending, RTSCompleted:
		return rt, nil
	}

	return ReferralTrackingStatus(""), fmt.Errorf("%s is not a ReferralTrackingStatus", s)
}

func (r *ReferralTrackingStatus) Scan(src interface{}) error {

	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into ReferralTrackingStatus when string expected", src)
	}

	var err error
	*r, err = GetReferralTrackingStatus(string(str))

	return err
}
