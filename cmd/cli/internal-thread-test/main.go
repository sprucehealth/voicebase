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

	_, err = threadingCli.CreateEmptyThread(context.Background(), &threading.CreateEmptyThreadRequest{
		OrganizationID: orgID,
		Source: &threading.Endpoint{
			Channel: threading.Endpoint_APP,
			ID:      orgID,
		},
		PrimaryEntityID: orgID,
		Summary:         "No messages yet",
	})
	if err != nil {
		golog.Fatalf("%s", err)
	}
}
