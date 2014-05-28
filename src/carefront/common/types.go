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

type Platform string

var (
	Android Platform = "android"
	IOS     Platform = "iOS"
)

func (p Platform) String() string {
	return string(p)
}

func GetPlatform(p string) (Platform, error) {
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
	*p, err = GetPlatform(string(text))
	return err
}

func (p *Platform) Scan(src interface{}) error {

	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %s into Platform when string expected", reflect.TypeOf(src))
	}

	var err error
	*p, err = GetPlatform(string(str))

	return err
}

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

type PushPromptStatus string

func (p PushPromptStatus) String() string {
	return string(p)
}

var (
	Unprompted PushPromptStatus = "UNPROMPTED"
	Accepted   PushPromptStatus = "ACCEPTED"
	Declined   PushPromptStatus = "DECLINED"
)

func GetPushPromptStatus(promptStatus string) (PushPromptStatus, error) {
	switch promptStatus {
	case "UNPROMPTED":
		return Unprompted, nil
	case "ACCEPTED":
		return Accepted, nil
	case "DECLINED":
		return Declined, nil
	}
	return PushPromptStatus(""), errors.New("Unknown prompt status: " + promptStatus)
}
