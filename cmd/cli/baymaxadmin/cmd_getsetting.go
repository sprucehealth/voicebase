package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type getSettingCmd struct {
	cnf         *config
	settingsCli settings.SettingsClient
}

func newGetSettingCmd(cnf *config) (command, error) {
	settingsCli, err := cnf.settingsClient()
	if err != nil {
		return nil, err
	}
	return &getSettingCmd{
		cnf:         cnf,
		settingsCli: settingsCli,
	}, nil
}

func (c *getSettingCmd) run(args []string) error {
	fs := flag.NewFlagSet("getsetting", flag.ExitOnError)
	nodeID := fs.String("node_id", "", "ID of the node who owns the setting")
	key := fs.String("key", "", "Setting key")
	subkey := fs.String("subkey", "", "Optional setting sub-key")
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

	res, err := c.settingsCli.GetValues(ctx, &settings.GetValuesRequest{
		Keys: []*settings.ConfigKey{
			{Key: *key, Subkey: *subkey},
		},
		NodeID: *nodeID,
	})
	if grpc.Code(err) == codes.NotFound {
		return errors.New("Value not found")
	} else if err != nil {
		return fmt.Errorf("Failed to get settings value: %s", err)
	}

	for _, v := range res.Values {
		fmt.Printf("Setting %s", v.Key.Key)
		if v.Key.Subkey != "" {
			fmt.Printf(":%s", v.Key.Subkey)
		}
		fmt.Printf(" (type %s)", v.Type)
		switch v.Type {
		case settings.ConfigType_BOOLEAN:
			fmt.Printf(" = %t\n", v.GetBoolean().Value)
		case settings.ConfigType_INTEGER:
			fmt.Printf(" = %d\n", v.GetInteger().Value)
		case settings.ConfigType_SINGLE_SELECT:
			item := v.GetSingleSelect().Item
			fmt.Printf(" = [%s] %s\n", item.ID, item.FreeTextResponse)
		case settings.ConfigType_MULTI_SELECT:
			fmt.Println()
			for _, item := range v.GetMultiSelect().Items {
				fmt.Printf("    [%s] %s\n", item.ID, item.FreeTextResponse)
			}
		case settings.ConfigType_STRING_LIST:
			fmt.Println()
			for _, s := range v.GetStringList().Values {
				fmt.Printf("    %s\n", s)
			}
		default:
			fmt.Printf(" !!! UNKNOWN TYPE\n")
		}
	}

	return nil
}
