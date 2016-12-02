package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/excomms"
)

type provisionNumberCmd struct {
	cnf       *config
	dirCli    directory.DirectoryClient
	excommCli excomms.ExCommsClient
}

func newProvisionNumberCmd(cnf *config) (command, error) {
	dirCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}
	excommCli, err := cnf.exCommsClient()
	if err != nil {
		return nil, err
	}
	return &provisionNumberCmd{
		cnf:       cnf,
		dirCli:    dirCli,
		excommCli: excommCli,
	}, nil
}

func (c *provisionNumberCmd) run(args []string) error {
	fs := flag.NewFlagSet("provisionnumber", flag.ExitOnError)
	orgID := fs.String("org_id", "", "ID of the organization to provision a number for")
	areaCode := fs.String("area_code", "", "Area code in which to provision the number")
	number := fs.String("number", "", "The exact number to provision")
	uuid := fs.String("uuid", "", "UUID for provisioning request")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *orgID == "" {
		*orgID = prompt(scn, "Organization entity ID: ")
	}

	ctx := context.Background()

	fmt.Printf("Organization:\n\n")
	org, err := lookupAndDisplayEntity(ctx, c.dirCli, *orgID, []directory.EntityInformation{
		directory.EntityInformation_CONTACTS,
	})
	if err != nil {
		return err
	}
	if org.Type != directory.EntityType_ORGANIZATION {
		return errors.Errorf("Can only provision numbers for organizations, not %s", org.Type)
	}
	if org.Status != directory.EntityStatus_ACTIVE {
		return errors.Errorf("Expected organization to be actove, got %s", org.Status)
	}

	if *areaCode == "" && *number == "" {
		*areaCode = prompt(scn, "Area code: ")
	}
	if *areaCode == "" && *number == "" {
		*areaCode = prompt(scn, "Phone number: ")
	}
	if *areaCode == "" && *number == "" {
		return errors.Errorf("Either area code or phone number required")
	}
	if *areaCode != "" && *number != "" {
		return errors.Errorf("Only one of area code or phone number may be provided")
	}

	if *uuid == "" {
		*uuid = prompt(scn, "UUID: ")
	}
	if *uuid == "" {
		return errors.Errorf("UUID required")
	}
	*uuid = org.ID + ":" + *uuid

	var req *excomms.ProvisionPhoneNumberRequest
	if *areaCode != "" {
		req = &excomms.ProvisionPhoneNumberRequest{
			UUID:         *uuid,
			ProvisionFor: org.ID,
			Number: &excomms.ProvisionPhoneNumberRequest_AreaCode{
				AreaCode: *areaCode,
			},
		}
	} else {
		req = &excomms.ProvisionPhoneNumberRequest{
			UUID:         *uuid,
			ProvisionFor: org.ID,
			Number: &excomms.ProvisionPhoneNumberRequest_PhoneNumber{
				PhoneNumber: *number,
			},
		}
	}
	res, err := c.excommCli.ProvisionPhoneNumber(ctx, req)
	if err != nil {
		return errors.Errorf("failed to provision number: %s", err)
	}
	fmt.Printf("Provisioned %s\n", res.PhoneNumber)

	dres, err := c.dirCli.CreateContact(ctx, &directory.CreateContactRequest{
		EntityID: org.ID,
		Contact: &directory.Contact{
			Provisioned: true,
			ContactType: directory.ContactType_PHONE,
			Value:       res.PhoneNumber,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
	})
	if err != nil {
		return errors.Errorf("failed to add contact to organization: %s", err)
	}
	displayEntity("", dres.Entity)

	return nil
}
