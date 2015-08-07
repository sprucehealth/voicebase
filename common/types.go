package common

import (
	"errors"
	"fmt"
	"strings"
)

type ByStatusTimestamp []StatusEvent

func (a ByStatusTimestamp) Len() int      { return len(a) }
func (a ByStatusTimestamp) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByStatusTimestamp) Less(i, j int) bool {
	return a[i].StatusTimestamp.Before(a[j].StatusTimestamp)
}

type ByCreationDate []*Card

func (c ByCreationDate) Len() int           { return len(c) }
func (c ByCreationDate) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
func (c ByCreationDate) Less(i, j int) bool { return c[i].CreationDate.Before(c[j].CreationDate) }

type Platform string

const (
	Android Platform = "android"
	IOS     Platform = "iOS"
)

func (p Platform) String() string {
	return string(p)
}

func ParsePlatform(p string) (Platform, error) {
	switch p {
	case "android":
		return Android, nil
	case "iOS":
		return IOS, nil
	}
	return Platform(""), fmt.Errorf("Unable to determine platform type from %s", p)
}

func (p *Platform) UnmarshalText(text []byte) error {
	var err error
	*p, err = ParsePlatform(string(text))
	return err
}

func (p *Platform) Scan(src interface{}) error {

	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into Platform when string expected", src)
	}

	var err error
	*p, err = ParsePlatform(string(str))

	return err
}

const (
	SMS   CommunicationType = "SMS"
	Email CommunicationType = "EMAIL"
	Push  CommunicationType = "PUSH"
)

type CommunicationType string

func (c CommunicationType) String() string {
	return string(c)
}

func (c *CommunicationType) Scan(src interface{}) error {

	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into CommunicationType when string expected", src)
	}

	var err error
	*c, err = ParseCommunicationType(string(str))

	return err
}

func ParseCommunicationType(c string) (CommunicationType, error) {
	switch c {
	case "SMS":
		return SMS, nil
	case "EMAIL":
		return Email, nil
	case "PUSH":
		return Push, nil
	}
	return CommunicationType(""), fmt.Errorf("Unable to determine communication type for %s", c)
}

type PushPromptStatus string

func (p PushPromptStatus) String() string {
	return string(p)
}

const (
	Unprompted PushPromptStatus = "UNPROMPTED"
	Accepted   PushPromptStatus = "ACCEPTED"
	Declined   PushPromptStatus = "DECLINED"
)

func ParsePushPromptStatus(promptStatus string) (PushPromptStatus, error) {
	switch strings.ToUpper(promptStatus) {
	case "UNPROMPTED":
		return Unprompted, nil
	case "ACCEPTED":
		return Accepted, nil
	case "DECLINED":
		return Declined, nil
	}
	return PushPromptStatus(""), errors.New("Unknown prompt status: " + promptStatus)
}

type MedicalLicenseStatus string

const (
	MLActive    MedicalLicenseStatus = "ACTIVE"
	MLInactive  MedicalLicenseStatus = "INACTIVE"
	MLTemporary MedicalLicenseStatus = "TEMPORARY"
	MLPending   MedicalLicenseStatus = "PENDING"
)

func (l MedicalLicenseStatus) String() string {
	return string(l)
}

func (l *MedicalLicenseStatus) Scan(src interface{}) error {
	var err error
	switch s := src.(type) {
	case string:
		*l, err = ParseMedicalLicenseStatus(s)
	case []byte:
		*l, err = ParseMedicalLicenseStatus(string(s))
	default:
		return fmt.Errorf("common: can't scan type %T into MedicalLicenseStatus", src)
	}
	return err
}

func ParseMedicalLicenseStatus(s string) (MedicalLicenseStatus, error) {
	switch l := MedicalLicenseStatus(strings.ToUpper(s)); l {
	case MLActive, MLInactive, MLTemporary, MLPending:
		return l, nil
	}
	return "", errors.New("common: unknown medical license status: " + s)
}

type MedicalRecordStatus string

const (
	MRPending MedicalRecordStatus = "PENDING"
	MRError   MedicalRecordStatus = "ERROR"
	MRSuccess MedicalRecordStatus = "SUCCESS"
)

func (s MedicalRecordStatus) String() string {
	return string(s)
}

func (s *MedicalRecordStatus) Scan(src interface{}) error {
	var err error
	switch v := src.(type) {
	case string:
		*s, err = ParseMedicalRecordStatus(v)
	case []byte:
		*s, err = ParseMedicalRecordStatus(string(v))
	default:
		return fmt.Errorf("common: can't scan type %T into MedicalRecordStatus", src)
	}
	return err
}

func ParseMedicalRecordStatus(str string) (MedicalRecordStatus, error) {
	switch s := MedicalRecordStatus(strings.ToUpper(str)); s {
	case MRPending, MRError, MRSuccess:
		return s, nil
	}
	return "", errors.New("common: unknown medical record status: " + str)
}
