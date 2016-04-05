package main

import (
	"encoding/csv"
	"flag"
	"io"
	"os"
	"strconv"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/svc/directory"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	flagDirectoryAddr = flag.String("directory_addr", "", "`host:port` of directory service")
	flagEntityListCSV = flag.String("entity_csv", "", "filename of file containing list of entityIDs that need to have their displayname determined")
)

func main() {
	flag.Parse()

	conn, err := grpc.Dial(
		*flagDirectoryAddr,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with directory service: %s", err.Error())
	}
	defer conn.Close()
	directoryClient := directory.NewDirectoryClient(conn)

	entityIDs, err := getEntityIDs()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	for _, entityID := range entityIDs {

		// lookup the entity to build the display name
		res, err := directoryClient.LookupEntities(
			context.Background(),
			&directory.LookupEntitiesRequest{
				LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
				LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
					EntityID: entityID,
				},
				RequestedInformation: &directory.RequestedInformation{
					Depth: 1,
					EntityInformation: []directory.EntityInformation{
						directory.EntityInformation_CONTACTS,
					},
				},
			})
		if err != nil {
			golog.Fatalf("Unable to lookup entity information for %s: %s", entityID, err.Error())
		} else if len(res.Entities) != 0 {
			golog.Fatalf("Expected 1 entity for %s but got %d", entityID, len(res.Entities))
		}

		entity := res.Entities[0]
		entity.Info.DisplayName = buildDisplayName(entity.Info, entity.Contacts)
		if entity.Info.DisplayName == "" {
			golog.Errorf("Unable to build display name for %s", entityID)
			continue
		}

		// update the display name
		if _, err := directoryClient.UpdateEntity(context.Background(), &directory.UpdateEntityRequest{
			EntityID:   entityID,
			EntityInfo: entity.Info,
			Contacts:   entity.Contacts,
		}); err != nil {
			golog.Fatalf("Unable to update entity information for %s: %s", entityID, err.Error())
		}
		golog.Infof("Successfully updated display name for entity %s", entityID)
	}
}

func getEntityIDs() ([]string, error) {
	csvFile, err := os.Open(*flagEntityListCSV)
	if err != nil {
		return nil, err
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = '\n'

	var entityIDs []string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		entityIDInt64, err := strconv.ParseUint(row[0], 10, 64)
		if err != nil {
			return nil, err
		}

		entityID := model.ObjectID{
			Prefix:  directory.EntityIDPrefix,
			Val:     entityIDInt64,
			IsValid: true,
		}

		entityIDs = append(entityIDs, entityID.String())
	}

	return entityIDs, nil
}

func buildDisplayName(info *directory.EntityInfo, contacts []*directory.Contact) string {
	if info.FirstName != "" || info.LastName != "" {
		var displayName string
		if info.FirstName != "" {
			displayName = info.FirstName
		}
		if info.MiddleInitial != "" {
			displayName += " " + info.MiddleInitial
		}
		if info.LastName != "" {
			displayName += " " + info.LastName
		}

		if info.ShortTitle != "" {
			displayName += ", " + info.ShortTitle
		}
		return displayName
	} else if info.GroupName != "" {
		return info.GroupName
	}

	// pick the display name to be the first contact value
	for _, c := range contacts {
		if c.ContactType == directory.ContactType_PHONE {
			pn, err := phone.Format(c.Value, phone.Pretty)
			if err != nil {
				return c.Value
			}
			return pn
		}
		return c.Value
	}

	return ""
}
