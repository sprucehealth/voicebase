package common

import (
	"errors"
	"fmt"
	"reflect"
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

var (
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
		return fmt.Errorf("Cannot scan type %s into CommunicationType when string expected", reflect.TypeOf(src))
	}

	var err error
	*c, err = GetCommunicationType(string(str))

	return err
}

func GetCommunicationType(c string) (CommunicationType, error) {
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

type PatientPromptStatus string

func (p PatientPromptStatus) String() string {
	return string(p)
}

var (
	Unprompted PatientPromptStatus = "UNPROMPTED"
	Accepted   PatientPromptStatus = "ACCEPTED"
	Declined   PatientPromptStatus = "DECLINED"
)

func GetPromptStatus(promptStatus string) (PatientPromptStatus, error) {
	switch promptStatus {
	case "UNPROMPTED":
		return Unprompted, nil
	case "ACCEPTED":
		return Accepted, nil
	case "DECLINED":
		return Declined, nil
	}
	return PatientPromptStatus(""), errors.New("Unknown prompt status: " + promptStatus)
}
