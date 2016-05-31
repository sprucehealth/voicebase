package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/directory"
	"golang.org/x/net/context"
)

type addContactCmd struct {
	cnf    *config
	dirCli directory.DirectoryClient
}

func newAddContactCmd(cnf *config) (command, error) {
	dirCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}

	return &addContactCmd{
		cnf:    cnf,
		dirCli: dirCli,
	}, nil
}

func (c *addContactCmd) run(args []string) error {
	fs := flag.NewFlagSet("addcontact", flag.ExitOnError)
	entityID := fs.String("entity_id", "", "ID of the entity that has the contact")
	provisioned := fs.Bool("provisioned", false, "whether or not number is provisioned")
	contactType := fs.String("contact_type", "", "phone or email")
	contactValue := fs.String("contact_value", "", "value of contact")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx := context.Background()
	scn := bufio.NewScanner(os.Stdin)

	if *entityID == "" {
		*entityID = prompt(scn, "Entity ID: ")
	}
	if *entityID == "" {
		return errors.New("Entity ID required")
	}

	if *contactType == "" {
		*contactType = prompt(scn, "Contact Type: ")
	}
	if *contactType == "" {
		return errors.New("Contact Value required")
	}

	if *contactValue == "" {
		*contactValue = prompt(scn, "Contact Value: ")
	}
	if *contactValue == "" {
		return errors.New("Contact Value required")
	}

	var cType directory.ContactType
	var value string
	var err error
	switch *contactType {
	case "phone":
		cType = directory.ContactType_PHONE
		value, err = phone.Format(*contactValue, phone.E164)
		if err != nil {
			return errors.Trace(err)
		}
	case "email":
		cType = directory.ContactType_EMAIL
		if !validate.Email(*contactValue) {
			return errors.Trace(fmt.Errorf("Invalid email %s : %s", *contactValue, err))
		}
	default:
		return fmt.Errorf("unknown contact type %s (can only be phone or email)", *contactType)

	}

	req := &directory.CreateContactRequest{
		Contact: &directory.Contact{
			ContactType: cType,
			Value:       value,
			Provisioned: *provisioned,
		},
		EntityID: *entityID,
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: []directory.EntityInformation{directory.EntityInformation_CONTACTS},
		},
	}

	res, err := c.dirCli.CreateContact(ctx, req)
	if err != nil {
		return errors.Trace(err)
	}

	// success!
	fmt.Println("Contact Added")
	displayEntity("", res.Entity)

	return nil
}
