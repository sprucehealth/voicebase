package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/excomms"
)

type blockNumberCmd struct {
	cnf        *config
	excommsCli excomms.ExCommsClient
}

func newBlockNumberCmd(cnf *config) (command, error) {
	excommsCli, err := cnf.exCommsClient()
	if err != nil {
		return nil, err
	}
	return &blockNumberCmd{
		cnf:        cnf,
		excommsCli: excommsCli,
	}, nil
}

func (b *blockNumberCmd) run(args []string) error {
	fs := flag.NewFlagSet("blocknumber", flag.ExitOnError)
	orgEntityID := fs.String("org_entity_id", "", "EntityID of the organization")
	provisionedPhoneNumber := fs.String("provisioned_phone_number", "", "Spruce phone number")
	blockPhoneNumber := fs.String("phone_number", "", "Phone number to block or unblock")
	unblock := fs.Bool("unblock", false, "Unblock phone number")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	if *orgEntityID == "" {
		*orgEntityID = prompt(scn, "OrgEntityID: ")
	}
	if *orgEntityID == "" {
		return errors.New("EntityID for org required")
	}

	if *provisionedPhoneNumber == "" {
		*provisionedPhoneNumber = prompt(scn, "ProvisionedPhoneNumber: ")
	}
	if *provisionedPhoneNumber == "" {
		return errors.New("ProvisionedPhoneNumber required")
	}

	if *blockPhoneNumber == "" {
		*blockPhoneNumber = prompt(scn, "Phone Number To Block/Unblock: ")
	}
	if *blockPhoneNumber == "" {
		return errors.New("Phone number to unblock/block required")
	}

	var phoneNumbers []string
	if *unblock {
		res, err := b.excommsCli.UnblockNumber(ctx, &excomms.UnblockNumberRequest{
			OrgID:                  *orgEntityID,
			Number:                 *blockPhoneNumber,
			ProvisionedPhoneNumber: *provisionedPhoneNumber,
		})
		if err != nil {
			return errors.Trace(err)
		}
		phoneNumbers = res.Numbers
	} else {
		res, err := b.excommsCli.BlockNumber(ctx, &excomms.BlockNumberRequest{
			OrgID:                  *orgEntityID,
			Number:                 *blockPhoneNumber,
			ProvisionedPhoneNumber: *provisionedPhoneNumber,
		})
		if err != nil {
			return errors.Trace(err)
		}
		phoneNumbers = res.Numbers
	}

	fmt.Printf("Success!\nCurrent list of blocked numbers for %s: %+v\n", *provisionedPhoneNumber, phoneNumbers)
	return nil
}
