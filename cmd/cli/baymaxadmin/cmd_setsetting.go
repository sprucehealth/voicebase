package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type setSettingCmd struct {
	cnf         *config
	settingsCli settings.SettingsClient
}

func newSetSettingCmd(cnf *config) (command, error) {
	settingsCli, err := cnf.settingsClient()
	if err != nil {
		return nil, err
	}
	return &setSettingCmd{
		cnf:         cnf,
		settingsCli: settingsCli,
	}, nil
}

func (c *setSettingCmd) run(args []string) error {
	fs := flag.NewFlagSet("setsetting", flag.ExitOnError)
	nodeID := fs.String("node_id", "", "ID of the node who owns the setting")
	key := fs.String("key", "", "Setting key")
	subkey := fs.String("subkey", "", "Optional setting sub-key")
	value := fs.String("value", "", "Value for the setting")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	ctx := context.Background()

	scn := bufio.NewScanner(os.Stdin)

	if *nodeID == "" {
		*nodeID = prompt(scn, "Node ID: ")
	}
	if *nodeID == "" {
		return errors.New("Node ID required")
	}

	if *key == "" {
		*key = prompt(scn, "Key: ")
	}
	if *key == "" {
		return errors.New("Key required")
	}

	// TODO: for now requiring a value. need to consider how to set values to empty strings and such
	if *value == "" {
		return errors.New("Value required")
	}

	cres, err := c.settingsCli.GetConfigs(ctx, &settings.GetConfigsRequest{
		Keys: []string{*key},
	})
	if grpc.Code(err) == codes.NotFound {
		return errors.New("Setting config not found")
	} else if err != nil {
		return fmt.Errorf("Failed to fetch setting config: %s", err)
	} else if len(cres.Configs) == 0 {
		return errors.New("Setting config not found")
	} else if len(cres.Configs) != 1 {
		return fmt.Errorf("Expected 1 config, got %d", len(cres.Configs))
	}
	config := cres.Configs[0]

	val := &settings.Value{
		Key: &settings.ConfigKey{
			Key:    *key,
			Subkey: *subkey,
		},
	}

	// TODO: implement more types. how to pass in complex types?
	// TODO: validate values where appropriate
	switch config.Type {
	case settings.ConfigType_BOOLEAN:
		b, err := strconv.ParseBool(*value)
		if err != nil {
			return fmt.Errorf("Failed to parse value as boolean: %s", err)
		}
		val.Type = settings.ConfigType_BOOLEAN
		val.Value = &settings.Value_Boolean{
			Boolean: &settings.BooleanValue{
				Value: b,
			},
		}
	case settings.ConfigType_INTEGER:
		i, err := strconv.ParseInt(*value, 10, 64)
		if err != nil {
			return fmt.Errorf("Failed to parse value as int: %s", err)
		}
		val.Type = settings.ConfigType_INTEGER
		val.Value = &settings.Value_Integer{
			Integer: &settings.IntegerValue{
				Value: i,
			},
		}
	case settings.ConfigType_SINGLE_SELECT:
		val.Type = settings.ConfigType_SINGLE_SELECT
		val.Value = &settings.Value_SingleSelect{
			SingleSelect: &settings.SingleSelectValue{
				Item: &settings.ItemValue{
					ID: *value,
				},
			},
		}
	default:
		return fmt.Errorf("Unsupported type %s", config.Type)
	}

	if _, err := c.settingsCli.SetValue(ctx, &settings.SetValueRequest{
		NodeID: *nodeID,
		Value:  val,
	}); err != nil {
		return fmt.Errorf("Failed to set value: %s", err)
	}

	return nil
}
