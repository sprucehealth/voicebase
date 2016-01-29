package settings

import (
	"fmt"
	"time"

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
