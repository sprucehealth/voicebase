package common

import (
	"fmt"
	"time"
)

type PromotionGroup struct {
	ID               int64
	Name             string
	MaxAllowedPromos int
}

type PromoCode struct {
	ID         int64
	Code       string
	IsReferral bool
}

type Promotion struct {
	Code    string
	CodeID  int64
	Data    Typed
	Group   string
	Expires *time.Time
	Created time.Time
}

type PatientPromotion struct {
	PatientID int64
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

type ReferralProgramTemplate struct {
	ID         int64
	Role       string
	RoleTypeID int64
	Data       Typed
	Created    time.Time
	Status     ReferralProgramStatus
}

type ReferralProgram struct {
	TemplateID *int64
	AccountID  int64
	Code       string
	CodeID     int64
	Data       Typed
	Created    time.Time
	Status     ReferralProgramStatus
}

type ReferralTrackingEntry struct {
	CodeID             int64
	ClaimingPatientID  int64
	ReferringAccountID int64
	Created            time.Time
	Status             ReferralTrackingStatus
}

type PatientCredit struct {
	PatientID int64
	Credit    int
}

type ParkedAccount struct {
	ID             int64
	CodeID         int64
	Code           string
	IsReferral     bool
	Email          string
	State          string
	PatientCreated bool
}

type ReferralProgramStatus string

const (
	RSActive   ReferralProgramStatus = "Active"
	RSInactive ReferralProgramStatus = "Inactive"
)

func (p ReferralProgramStatus) String() string {
	return string(p)
}

func GetReferralProgramStatus(s string) (ReferralProgramStatus, error) {
	switch rs := ReferralProgramStatus(s); rs {
	case RSActive, RSInactive:
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
