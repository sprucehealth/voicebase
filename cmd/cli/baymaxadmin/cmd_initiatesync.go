package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"os"

	"github.com/sprucehealth/backend/svc/patientsync"
)

type initiateSyncCmd struct {
	cnf            *config
	patientSyncCli patientsync.PatientSyncClient
}

func newInitiateSyncCmd(cnf *config) (command, error) {
	patientSyncCli, err := cnf.patientSyncClient()
	if err != nil {
		return nil, err
	}
	return &initiateSyncCmd{
		cnf:            cnf,
		patientSyncCli: patientSyncCli,
	}, nil
}

func (c *initiateSyncCmd) run(args []string) error {
	fs := flag.NewFlagSet("initiatesync", flag.ExitOnError)
	entityID := fs.String("entity_id", "", "ID of entity")
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

	_, err := c.patientSyncCli.InitiateSync(context.Background(), &patientsync.InitiateSyncRequest{
		OrganizationEntityID: *entityID,
		Source:               patientsync.SOURCE_HINT,
	})
	return err
}
