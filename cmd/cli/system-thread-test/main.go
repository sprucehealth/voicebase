package main

import (
	"flag"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/threading"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	flagDirectoryAddr = flag.String("directory_addr", "127.0.0.1:5002", "host:port of directory service")
	flagThreadingAddr = flag.String("threading_addr", "127.0.0.1:5001", "host:port of threading service")
)

func main() {
	boot.ParseFlags("")
	orgID := flag.Arg(0)
	if orgID == "" {
		golog.Fatalf("Organization ID required")
	}
	name := flag.Arg(1)
	if name == "" {
		golog.Fatalf("Name is required")
	}

	conn, err := grpc.Dial(*flagThreadingAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to threading service: %s", err)
	}
	threadingCli := threading.NewThreadsClient(conn)

	conn, err = grpc.Dial(*flagDirectoryAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to directory service: %s", err)
	}
	directoryCli := directory.NewDirectoryClient(conn)

	ctx := context.Background()

	dres, err := directoryCli.LookupEntities(ctx, &directory.LookupEntitiesRequest{
		LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
		LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
			EntityID: orgID,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{},
		},
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}
	if len(dres.Entities) != 1 {
		golog.Fatalf("Expected 1 entity got %d", len(dres.Entities))
	}
	if dres.Entities[0].Type != directory.EntityType_ORGANIZATION {
		golog.Fatalf("Expected organization entity got %s", dres.Entities[0].Type)
	}

	dres2, err := directoryCli.CreateEntity(ctx, &directory.CreateEntityRequest{
		Type: directory.EntityType_SYSTEM,
		InitialMembershipEntityID: orgID,
		Contacts:                  nil,
		EntityInfo: &directory.EntityInfo{
			DisplayName: name,
			GroupName:   name,
		},
		RequestedInformation: &directory.RequestedInformation{
			Depth:             0,
			EntityInformation: []directory.EntityInformation{},
		},
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}
	entID := dres2.Entity.ID

	messages := []string{
		`Welcome to Spruce! Let’s get you set up with your own Spruce phone number so you can start receiving calls, voicemails, and texts from patients without disclosing your personal number.

<a href="https://dev-baymax.carefront.net/org/` + orgID + `/settings/phone">Get your Spruce number</a>
or type "skip" to get it later`,
		`Success! Your patients can now reach you at {number}. Next let’s set up you up to send and receive email through Spruce.

<a href="https://dev-baymax.carefront.net/org/` + orgID + `/settings/email">Set up email support</a>
or type "skip" to set it up later`,
		`Success! Your patients can now reach you at {email address}. Would you like to collaborate with colleagues around patient communication? Spruce can do that too.

<a href="https://dev-baymax.carefront.net/org/` + orgID + `/invite">Add a colleague to your organization</a>
or type "skip" to send invites later`,
		`We’ve sent your invite to colleague. Once they’ve joined, you can communicate with them about care, right from a patient’s conversation thread.

To send internal messages or notes in a patient thread, simply tap the lock icon while writing a message to mark it as internal. You can test it out right here.`,
		`That’s all for now. You’re well on your way to greater control in your communication with your patients. You can keep trying out other Spruce patient features in this conversation, and if you’re unsure about anything or need some help, message us on the @TeamSpruce conversation thread and a real human will respond.`,
	}

	ctres, err := threadingCli.CreateThread(context.Background(), &threading.CreateThreadRequest{
		OrganizationID: orgID,
		FromEntityID:   entID,
		Source: &threading.Endpoint{
			Channel: threading.Endpoint_APP,
			ID:      entID,
		},
		Internal: false,
		Summary:  messages[0][:100],
		Text:     messages[0],
	})
	if err != nil {
		golog.Fatalf("%s", err)
	}
	threadID := ctres.Thread.ID

	for _, m := range messages[1:] {
		summary := m
		if len(summary) > 100 {
			summary = summary[:100]
		}
		_, err = threadingCli.PostMessage(ctx, &threading.PostMessageRequest{
			ThreadID:     threadID,
			FromEntityID: entID,
			Source: &threading.Endpoint{
				Channel: threading.Endpoint_APP,
				ID:      entID,
			},
			Internal: false,
			Summary:  summary,
			Text:     m,
		})
		if err != nil {
			golog.Fatalf("%s", err)
		}
	}
}
