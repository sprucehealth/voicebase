package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sprucehealth/backend/svc/directory"
)

type updateEntityCmd struct {
	cnf    *config
	dirCli directory.DirectoryClient
}

func newUpdateEntityCmd(cnf *config) (command, error) {
	dirCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}

	return &updateEntityCmd{
		cnf:    cnf,
		dirCli: dirCli,
	}, nil
}

func (c *updateEntityCmd) run(args []string) error {
	fs := flag.NewFlagSet("entity", flag.ExitOnError)
	entityID := fs.String("entity_id", "", "ID of entity")
	firstName := fs.String("first_name", "", "first name of entity")
	lastName := fs.String("last_name", "", "last name of entity")
	groupName := fs.String("group_name", "", "group name of entity")
	shortTitle := fs.String("short_title", "", "short title of entity")
	longTitle := fs.String("long_title", "", "long title of entity")
	middleInitial := fs.String("middle_initial", "", "middle initial of entity")

	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *entityID == "" {
		*entityID = prompt(scn, "Entity ID: ")
	}
	if *entityID == "" {
		return errors.New("Entity ID required")
	}

	ctx := context.Background()
	entity, err := directory.SingleEntity(ctx, c.dirCli, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: *entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth: 0,
		},
	})
	if err != nil {
		return err
	} else if entity.Type != directory.EntityType_INTERNAL && entity.Type != directory.EntityType_ORGANIZATION {
		return fmt.Errorf("Expected entity of type INTERNAL or ORGANIZATION but got %s", entity.Type)
	}

	var update bool
	if *firstName != "" {
		entity.Info.FirstName = strings.TrimSpace(*firstName)
		update = true
	}
	if *lastName != "" {
		entity.Info.LastName = strings.TrimSpace(*lastName)
		update = true
	}
	if *middleInitial != "" {
		entity.Info.MiddleInitial = strings.TrimSpace(*middleInitial)
		update = true
	}
	if *groupName != "" {
		entity.Info.GroupName = strings.TrimSpace(*groupName)
		update = true
	}
	if *shortTitle != "" {
		entity.Info.ShortTitle = strings.TrimSpace(*shortTitle)
		update = true
	}
	if *longTitle != "" {
		entity.Info.LongTitle = strings.TrimSpace(*longTitle)
		update = true
	}

	if !update {
		return errors.New("specify at least one field to update")
	}

	_, err = c.dirCli.UpdateEntity(ctx, &directory.UpdateEntityRequest{
		EntityID:         *entityID,
		EntityInfo:       entity.Info,
		UpdateEntityInfo: true,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Entity updated!\n\n")
	if _, err := lookupAndDisplayEntity(ctx, c.dirCli, *entityID, []directory.EntityInformation{directory.EntityInformation_CONTACTS}); err != nil {
		return err
	}

	return nil
}
