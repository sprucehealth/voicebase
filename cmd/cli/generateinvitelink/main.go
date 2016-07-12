package main

// The purpose of this script is to talk to easily generate an invite link
// on behalf of a user.

import (
	"flag"

	"context"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/invite"
	"google.golang.org/grpc"
)

var (
	flagInviteAddr      = flag.String("invite_addr", "", "`host:port` of invite service")
	flagOrgID           = flag.String("org_id", "", "orgID into which to invite individual")
	flagInviterEntityID = flag.String("inviter_entity_id", "", "entityID of the inviter")
	flagEmailAddress    = flag.String("email_address", "", "email to send the invite to")
	flagPhoneNumber     = flag.String("phone_number", "", "phone number to use for the invite")
)

func main() {
	flag.Parse()

	if *flagInviteAddr == "" {
		golog.Fatalf("Invite service not configured")
	} else if *flagOrgID == "" {
		golog.Fatalf("OrgID not specified")
	} else if *flagInviterEntityID == "" {
		golog.Fatalf("InviterEntityID not specified")
	} else if *flagEmailAddress == "" {
		golog.Fatalf("email address not specified")
	} else if *flagPhoneNumber == "" {
		golog.Fatalf("phone number not specified")
	}

	conn, err := grpc.Dial(*flagInviteAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to invite service: %s", err)
	}
	inviteClient := invite.NewInviteClient(conn)

	_, err = inviteClient.InviteColleagues(context.Background(), &invite.InviteColleaguesRequest{
		OrganizationEntityID: *flagOrgID,
		InviterEntityID:      *flagInviterEntityID,
		Colleagues: []*invite.Colleague{
			{
				Email:       *flagEmailAddress,
				PhoneNumber: *flagPhoneNumber,
			},
		},
	})
	if err != nil {
		golog.Fatalf("Invite colleague request failed: %s", err.Error())
	}

	golog.Infof("Successfully invited %s to org %s", *flagEmailAddress, *flagOrgID)

}
