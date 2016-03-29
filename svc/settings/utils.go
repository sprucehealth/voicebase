package settings

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"golang.org/x/net/context"
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
		return nil, fmt.Errorf("expected single select but got %T", res.Values[0])
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
		return nil, errors.Trace(fmt.Errorf("Expected boolean value for revealing sender instead got %T", res.Values[0].Value))
	}
	return res.Values[0].GetBoolean(), nil
}
