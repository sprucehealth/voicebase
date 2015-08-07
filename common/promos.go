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

// PromotionUpdate represents the data to be applied to an UPDATE statement of a promotion record matching the provided promotion_code_id
type PromotionUpdate struct {
	CodeID  int64
	Expires *time.Time
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

// ReferralProgram represents a mapping between a promotion and the account allowed to refer that promotion
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

// ReferralTrackingEntry represents a historical tracking of referral links/codes
type ReferralTrackingEntry struct {
	CodeID             int64
	ClaimingAccountID  int64
	ReferringAccountID int64
	Created            time.Time
	Status             ReferralTrackingStatus
}

// PromotionReferralRoute represents a routing between a set of promotion routing criteria and a given promotin
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

// PromotionReferralRouteUpdate represents the mutable fields on a promotion_referral_route record
type PromotionReferralRouteUpdate struct {
	ID        int64
	Lifecycle PRRLifecycle
}

// AccountCredit represents the attributes of credit to be applied to an account
type AccountCredit struct {
	AccountID int64
	Credit    int
}

// ParkedAccount represents a known email and account that has yet to be activated but can have attributes associated with it
type ParkedAccount struct {
	ID             int64
	CodeID         int64
	Code           string
	IsReferral     bool
	Email          string
	State          string
	AccountCreated bool
}

// PRRLifecycle represents the lifecycle of a promotion_referral_route
type PRRLifecycle string

const (
	// PRRLifecycleActive represents the lifecycle where a promotion_referral_route will actively route new users to it
	PRRLifecycleActive PRRLifecycle = "ACTIVE"

	// PRRLifecycleNoNewUsers represents the lifecycle where a promotion_referral_route will actively not route new users but will allow existing users to retain state
	PRRLifecycleNoNewUsers PRRLifecycle = "NO_NEW_USERS"

	// PRRLifecycleDeprecated represents the lifecycle where a promotion_referral_route has been disabled and all owners will be moved off
	PRRLifecycleDeprecated PRRLifecycle = "DEPRECATED"
)

func (p PRRLifecycle) String() string {
	return string(p)
}

// ParsePRRLifecycle returns the PRRLifecycle that maps to the provided string
func ParsePRRLifecycle(s string) (PRRLifecycle, error) {
	switch rs := PRRLifecycle(strings.ToUpper(s)); rs {
	case PRRLifecycleActive, PRRLifecycleNoNewUsers, PRRLifecycleDeprecated:
		return rs, nil
	}

	return PRRLifecycle(""), fmt.Errorf("%s is not a PRRLifecycle", s)
}

// Scan allows for PRRLifecycle to be utilized in database queries and confirms the sql.Scanner interface
func (p *PRRLifecycle) Scan(src interface{}) error {

	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into PRRLifecycle when string expected", src)
	}

	var err error
	*p, err = ParsePRRLifecycle(string(str))

	return err
}

// PRRGender represents the available set of gender states for a promotion_referral_route
type PRRGender string

const (
	// PRRGenderMale represents the Male gender to match for a promotion_referral_route
	PRRGenderMale PRRGender = "M"

	// PRRGenderFemale represents the Female gender to match for a promotion_referral_route
	PRRGenderFemale PRRGender = "F"
)

func (p PRRGender) String() string {
	return string(p)
}

// ParsePRRGender returns the PRRGender that maps to the provided string
func ParsePRRGender(s string) (PRRGender, error) {
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

// Scan allows for PRRGender to be utilized in database queries and confirms the sql.Scanner interface
func (p *PRRGender) Scan(src interface{}) error {
	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into PRRGender when string expected", src)
	}

	var err error
	*p, err = ParsePRRGender(string(str))

	return err
}

// ReferralProgramStatus represents the available status field values for a refferal_program record
type ReferralProgramStatus string

// ReferralProgramStatusList is an alias for a list of ReferralProgramStatus
type ReferralProgramStatusList []string

const (
	// RSActive is the ReferralProgramStatus associated with a referral_program that can be actively shared and claimed by users
	RSActive ReferralProgramStatus = "ACTIVE"

	// RSInactive is the ReferralProgramStatus associated with a referral_program that can be actively shared and claimed by users
	RSInactive ReferralProgramStatus = "INACTIVE"

	// RSDefault represents the default referral program to apply to users who do not match a route
	RSDefault ReferralProgramStatus = "DEFAULT"
)

func (p ReferralProgramStatus) String() string {
	return string(p)
}

// ParseReferralProgramStatus returns the ReferralProgramStatus that maps to the provided string
func ParseReferralProgramStatus(s string) (ReferralProgramStatus, error) {
	switch rs := ReferralProgramStatus(strings.ToUpper(s)); rs {
	case RSActive, RSInactive, RSDefault:
		return rs, nil
	}

	return ReferralProgramStatus(""), fmt.Errorf("%s is not a ReferralProgramStatus", s)
}

// Scan allows for ReferralProgramStatus to be utilized in database queries and confirms the sql.Scanner interface
func (p *ReferralProgramStatus) Scan(src interface{}) error {

	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into ReferralProgramStatus when string expected", src)
	}

	var err error
	*p, err = ParseReferralProgramStatus(string(str))

	return err
}

// PromotionStatus represents the available status field values for a promotion record
type PromotionStatus string

const (
	// PSPending represents that a promotion has been applied to an account but not consumed
	PSPending PromotionStatus = "PENDING"

	// PSCompleted represents that a promotion has been consumed by an account
	PSCompleted PromotionStatus = "COMPLETED"

	// PSExpired represents that a promotion has passed it's expiration
	PSExpired PromotionStatus = "EXPIRED"

	// PSDeleted represents that a promotion has been removed from an account before being cosumed
	PSDeleted PromotionStatus = "DELETED"
)

func (p PromotionStatus) String() string {
	return string(p)
}

// ParsePromotionStatus returns the PromotionStatus that maps to the provided string
func ParsePromotionStatus(s string) (PromotionStatus, error) {
	switch ps := PromotionStatus(strings.ToUpper(s)); ps {
	case PSPending, PSCompleted, PSExpired:
		return ps, nil
	}

	return PromotionStatus(""), fmt.Errorf("%s is not a PromotionStatus", s)
}

// Scan allows for PromotionStatus to be utilized in database queries and confirms the sql.Scanner interface
func (p *PromotionStatus) Scan(src interface{}) error {
	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into PromotionStatus when string expected", src)
	}

	var err error
	*p, err = ParsePromotionStatus(string(str))

	return err
}

// ReferralTrackingStatus represents the status field associated with a referral_tracking_entry
type ReferralTrackingStatus string

const (
	// RTSPending represents that a referal_program has been shared but no visit completed
	RTSPending ReferralTrackingStatus = "PENDING"

	// RTSCompleted represents that a referal_program has been shared and a visit completed
	RTSCompleted ReferralTrackingStatus = "COMPLETED"
)

func (r ReferralTrackingStatus) String() string {
	return string(r)
}

// ParseReferralTrackingStatus returns the ReferralTrackingStatus that maps to the provided string
func ParseReferralTrackingStatus(s string) (ReferralTrackingStatus, error) {
	switch rt := ReferralTrackingStatus(strings.ToUpper(s)); rt {
	case RTSPending, RTSCompleted:
		return rt, nil
	}

	return ReferralTrackingStatus(""), fmt.Errorf("%s is not a ReferralTrackingStatus", s)
}

// Scan allows for ReferralTrackingStatus to be utilized in database queries and confirms the sql.Scanner interface
func (r *ReferralTrackingStatus) Scan(src interface{}) error {
	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into ReferralTrackingStatus when string expected", src)
	}

	var err error
	*r, err = ParseReferralTrackingStatus(string(str))

	return err
}
