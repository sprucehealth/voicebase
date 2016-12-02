package settings

import (
	"fmt"
	"time"

	"context"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
)

// RegisterConfig attempts to register the provided config within the dealine of the context (by retrying every second),
// and returns an error and fails if deadline passes and registration was not possible.
func RegisterConfigs(ctx context.Context, client SettingsClient, configs []*Config) (*RegisterConfigsResponse, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
Timeout:
	for {
		res, err := client.RegisterConfigs(ctx, &RegisterConfigsRequest{
			Configs: configs,
		})
		if err != nil {
			golog.Errorf("Unable to register configs. Retrying in 5s. Error: %s", err.Error())
		} else {
			return res, nil
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			break Timeout
		}
	}

	return nil, fmt.Errorf("Deadline exceeded, unable to register configs.")
}

var (
	// ErrValueNotFound is returned when a value does not exist for a setting
	ErrValueNotFound = errors.New("no value specified for setting")
	// ErrMoreThanOneValueFound is returned when more than one value is found for a setting
	ErrMoreThanOneValueFound = errors.New("more than one value found for setting")
)

// GetSingleSelectValue is a helper method to return a single select value for the provided request.
func GetSingleSelectValue(ctx context.Context, client SettingsClient, req *GetValuesRequest) (*SingleSelectValue, error) {
	res, err := client.GetValues(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(res.Values) == 0 {
		return nil, errors.Trace(ErrValueNotFound)
	} else if len(res.Values) != 1 {
		return nil, errors.Trace(ErrMoreThanOneValueFound)
	} else if res.Values[0].GetSingleSelect() == nil {
		return nil, errors.Errorf("expected single select but got %T", res.Values[0])
	}

	return res.Values[0].GetSingleSelect(), nil
}

// GetBooleanValue is a helper method to return a boolean value for the provided request.
func GetBooleanValue(ctx context.Context, client SettingsClient, req *GetValuesRequest) (*BooleanValue, error) {
	res, err := client.GetValues(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(res.Values) == 0 {
		return nil, errors.Trace(ErrValueNotFound)
	} else if len(res.Values) != 1 {
		return nil, errors.Trace(ErrMoreThanOneValueFound)
	} else if res.Values[0].GetBoolean() == nil {
		return nil, errors.Errorf("Expected boolean value instead got %T", res.Values[0].Value)
	}
	return res.Values[0].GetBoolean(), nil
}

// GetStringListValue is a helper method to return a string list value for the provided request.
func GetStringListValue(ctx context.Context, client SettingsClient, req *GetValuesRequest) (*StringListValue, error) {
	res, err := client.GetValues(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(res.Values) == 0 {
		return nil, errors.Trace(ErrValueNotFound)
	} else if len(res.Values) != 1 {
		return nil, errors.Trace(ErrMoreThanOneValueFound)
	} else if res.Values[0].GetStringList() == nil {
		return nil, errors.Errorf("Expected string list instead got %T", res.Values[0].Value)
	}
	return res.Values[0].GetStringList(), nil
}

// GetIntegerValue is a helper method to return an integer for the provided request.
func GetIntegerValue(ctx context.Context, client SettingsClient, req *GetValuesRequest) (*IntegerValue, error) {
	res, err := client.GetValues(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(res.Values) == 0 {
		return nil, errors.Trace(ErrValueNotFound)
	} else if len(res.Values) != 1 {
		return nil, errors.Trace(ErrMoreThanOneValueFound)
	} else if res.Values[0].GetInteger() == nil {
		return nil, errors.Errorf("Expected integer value instead got %T", res.Values[0].Value)
	}
	return res.Values[0].GetInteger(), nil
}

// GetTextValue is a helper method to return a text value for the provided request.
func GetTextValue(ctx context.Context, client SettingsClient, req *GetValuesRequest) (*TextValue, error) {
	res, err := client.GetValues(ctx, req)
	if err != nil {
		return nil, errors.Trace(err)
	} else if len(res.Values) == 0 {
		return nil, errors.Trace(ErrValueNotFound)
	} else if len(res.Values) != 1 {
		return nil, errors.Trace(ErrMoreThanOneValueFound)
	} else if res.Values[0].GetText() == nil {
		return nil, errors.Errorf("expected text value instead got %T", res.Values[0].Value)
	}
	return res.Values[0].GetText(), nil
}
