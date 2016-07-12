package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"context"

	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/invite"
	"github.com/sprucehealth/backend/svc/settings"
)

type enableOrgCodeCmd struct {
	cnf          *config
	directoryCli directory.DirectoryClient
	inviteCli    invite.InviteClient
	settingsCli  settings.SettingsClient
}

func newEnableOrgCodeCmd(cnf *config) (command, error) {
	settingsCli, err := cnf.settingsClient()
	if err != nil {
		return nil, err
	}
	directoryCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}
	inviteCli, err := cnf.inviteClient()
	if err != nil {
		return nil, err
	}
	return &enableOrgCodeCmd{
		cnf:          cnf,
		directoryCli: directoryCli,
		settingsCli:  settingsCli,
		inviteCli:    inviteCli,
	}, nil
}

func (c *enableOrgCodeCmd) run(args []string) error {
	fs := flag.NewFlagSet("enableorgcode", flag.ExitOnError)
	entityID := fs.String("entity_id", "", "ID of the organization to enable org codes for")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	ctx := context.Background()

	scn := bufio.NewScanner(os.Stdin)

	if *entityID == "" {
		*entityID = prompt(scn, "Entity ID: ")
	}

	resp, err := c.directoryCli.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: *entityID,
		},
		RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
	})
	if err != nil {
		return err
	} else if len(resp.Entities) != 1 {
		return fmt.Errorf("Expected 1 entity but for %v", resp.Entities)
	}

	if _, err := c.settingsCli.SetValue(ctx, &settings.SetValueRequest{
		NodeID: *entityID,
		Value: &settings.Value{
			Type: settings.ConfigType_BOOLEAN,
			Key: &settings.ConfigKey{
				Key: invite.ConfigKeyOrganizationCode,
			},
			Value: &settings.Value_Boolean{
				Boolean: &settings.BooleanValue{
					Value: true,
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("Failed to set value: %s", err)
	}

	cResp, err := c.inviteCli.CreateOrganizationInvite(ctx, &invite.CreateOrganizationInviteRequest{
		OrganizationEntityID: *entityID,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Organization Link: https://invite.sprucehealth.com/%s\n", cResp.Organization.Token)
	return nil
}
