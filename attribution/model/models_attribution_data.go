package model

import (
	"fmt"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
)

// AttributionData represents the external information to associate with an account/device
type AttributionData struct {
	ID           int64
	AccountID    *int64
	DeviceID     *string
	Data         map[string]interface{}
	CreationDate time.Time
	LastModified time.Time
}

// StringData returns attribution data marshalled into a string
func (a *AttributionData) StringData(key string) (string, bool, error) {
	iValue, ok := a.Data[key]
	if !ok {
		return "", ok, nil
	}
	switch t := iValue.(type) {
	case string:
		return t, ok, nil
	case *string:
		return *t, ok, nil
	case int:
		return strconv.FormatInt(int64(t), 10), ok, nil
	case int64:
		return strconv.FormatInt(t, 10), ok, nil
	}
	return "", ok, fmt.Errorf("Unsupported type %T for attribution taba conversion to string", iValue)
}

// Int64Data returns attribution data marshalled into an int64
func (a *AttributionData) Int64Data(key string) (int64, bool, error) {
	iValue, ok := a.Data[key]
	if !ok {
		return 0, ok, nil
	}
	switch t := iValue.(type) {
	case string:
		nValue, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return 0, ok, errors.Trace(err)
		}
		return nValue, ok, nil
	case *string:
		nValue, err := strconv.ParseInt(*t, 10, 64)
		if err != nil {
			return 0, ok, errors.Trace(err)
		}
		return nValue, ok, nil
	case int:
		return int64(t), ok, nil
	case int64:
		return t, ok, nil
	case *int:
		return int64(*t), ok, nil
	case *int64:
		return *t, ok, nil
	}
	return 0, ok, fmt.Errorf("Unsupported type %T for attribution taba conversion to int64", iValue)
}
