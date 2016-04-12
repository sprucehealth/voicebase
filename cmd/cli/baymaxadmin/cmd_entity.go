package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type entityCmd struct {
	cnf    *config
	dirCli directory.DirectoryClient
}

func newEntityCmd(cnf *config) (command, error) {
	dirCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}
	return &entityCmd{
		cnf:    cnf,
		dirCli: dirCli,
	}, nil
}

func (c *entityCmd) run(args []string) error {
	fs := flag.NewFlagSet("entity", flag.ExitOnError)
	entityID := fs.String("entity_id", "", "ID of entity")
	withMembers := fs.Bool("members", false, "Display members")
	withMemberships := fs.Bool("memberships", false, "Display memberships")
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
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	info := []directory.EntityInformation{directory.EntityInformation_CONTACTS, directory.EntityInformation_EXTERNAL_IDS}
	if *withMembers {
		info = append(info, directory.EntityInformation_MEMBERS)
	}
	if *withMemberships {
		info = append(info, directory.EntityInformation_MEMBERSHIPS)
	}

	_, err := lookupAndDisplayEntity(ctx, c.dirCli, *entityID, info)
	return err
}

func lookupAndDisplayEntity(ctx context.Context, dirCli directory.DirectoryClient, entityID string, info []directory.EntityInformation) (*directory.Entity, error) {
	req := &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: entityID,
		},
		RequestedInformation: &directory.RequestedInformation{
			EntityInformation: info,
		},
	}
	res, err := dirCli.LookupEntities(ctx, req)
	if err != nil && grpc.Code(err) != codes.NotFound {
		return nil, err
	}

	switch len(res.Entities) {
	case 0:
		return nil, errors.New("Entity not found")
	case 1:
	default:
		fmt.Fprintf(os.Stderr, "Expected 1 entity, got %d\n", len(res.Entities))
	}
	ent := res.Entities[0]

	displayEntity("", ent)

	return ent, nil
}

func displayEntity(indent string, ent *directory.Entity) {
	fmt.Printf("%sEntity %s (type %s) (status %s)\n", indent, ent.ID, ent.Type, ent.Status)
	if ent.AccountID != "" {
		fmt.Printf("%s    Account ID: %s\n", indent, ent.AccountID)
	}
	if len(ent.ExternalIDs) != 0 {
		fmt.Printf("%s    External IDs: %s\n", indent, strings.Join(ent.ExternalIDs, ", "))
	}
	displayEntityInfo(indent+"    ", ent.Info)
	displayEntityContacts(indent+"    ", ent.Contacts)
	if len(ent.Members) != 0 {
		fmt.Printf("%sMembers:\n", indent)
		displayEntities(indent+"    ", ent.Members)
	}
	if len(ent.Memberships) != 0 {
		fmt.Printf("%sMemberships:\n", indent)
		displayEntities(indent+"    ", ent.Memberships)
	}
}

func displayEntities(indent string, entities []*directory.Entity) {
	w := tabwriter.NewWriter(os.Stdout, 4, 8, 4, ' ', 0)
	fmt.Fprintf(w, indent+"ID\tDisplay Name\tType\tStatus\n")
	for _, e := range entities {
		fmt.Fprintf(w, indent+"%s\t%s\t%s\t%s\n", e.ID, e.Info.DisplayName, e.Type, e.Status)
	}
	if err := w.Flush(); err != nil {
		golog.Fatalf(err.Error())
	}
}

func displayEntityInfo(indent string, info *directory.EntityInfo) {
	if info == nil {
		return
	}
	fmt.Printf("%sDisplay Name: %s\n", indent, info.DisplayName)
	if info.GroupName != "" {
		fmt.Printf("%sGroup Name: %s\n", indent, info.GroupName)
	}
	if info.ShortTitle != "" {
		fmt.Printf("%sShort Title: %s\n", indent, info.ShortTitle)
	}
	if info.LongTitle != "" {
		fmt.Printf("%sLongTitle: %s\n", indent, info.LongTitle)
	}
	if info.FirstName != "" {
		fmt.Printf("%sFirst Name: %s\n", indent, info.FirstName)
	}
	if info.LastName != "" {
		fmt.Printf("%sLast Name: %s\n", indent, info.LastName)
	}
	if info.Gender != directory.EntityInfo_UNKNOWN {
		fmt.Printf("%sGender: %s\n", indent, info.Gender)
	}
	if info.DOB != nil {
		fmt.Printf("%sDOB: %s\n", indent, time.Date(int(info.DOB.Year), time.Month(info.DOB.Month), int(info.DOB.Day), 12, 0, 0, 0, nil).Format("Jan _2, 2006"))
	}
	if info.Note != "" {
		fmt.Printf("%sNote: %s\n", indent, info.Note)
	}
}

func displayEntityContacts(indent string, contacts []*directory.Contact) {
	if len(contacts) == 0 {
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 4, 8, 4, ' ', 0)
	fmt.Println("Contacts:")
	fmt.Fprintf(w, indent+"ID\tType\tValue\tLabel\tProvisioned\n")
	for _, c := range contacts {
		fmt.Fprintf(w, indent+"%s\t%s\t%s\t%s\t%t\n", c.ID, c.ContactType, c.Value, c.Label, c.Provisioned)
	}
	if err := w.Flush(); err != nil {
		golog.Fatalf(err.Error())
	}
}
