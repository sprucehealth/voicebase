package main

import (
	"encoding/csv"
	"flag"
	"io"
	"os"
	"strings"

	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	flagSettingsAddr  = flag.String("settings_addr", "", "`host:port` of settings service")
	flagDirectoryAddr = flag.String("directory_addr", "", "`host:port` of directory service")
	flagEntityList    = flag.String("entities_csv", "", "csv of entities for which to migrate setting")
	flagOrgIgnoreList = flag.String("orgs_to_ignore_csv", "", "csv of orgs to ignore")
)

func main() {
	flag.Parse()

	conn, err := grpc.Dial(*flagSettingsAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to settings service: %s", err)
	}
	settingsClient := settings.NewSettingsClient(conn)

	conn, err = grpc.Dial(*flagDirectoryAddr, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to connect to settings service: %s", err)
	}
	directoryClient := directory.NewDirectoryClient(conn)

	entites, err := getEntities()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	orgsToIgnore, err := getOrgsToIgnore()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	// create a map of the orgs for quick access
	orgsToIgnoreMap := make(map[string]struct{}, len(orgsToIgnore))
	for _, o := range orgsToIgnore {
		orgsToIgnoreMap[o] = struct{}{}
	}

	for _, entityID := range entites {
		// lookup orgID and provisionedPhone
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
						directory.EntityInformation_MEMBERSHIPS,
						directory.EntityInformation_CONTACTS,
					},
				},
				Statuses: []directory.EntityStatus{directory.EntityStatus_ACTIVE},
			})
		if err != nil {
			golog.Fatalf(err.Error())
		}

		var orgID string
		var provisionedPhone string
		for _, membership := range res.Entities[0].Memberships {
			if membership.Type == directory.EntityType_ORGANIZATION {
				orgID = membership.ID

				for _, contact := range membership.Contacts {
					if contact.Provisioned && contact.ContactType == directory.ContactType_PHONE {
						provisionedPhone = contact.Value
					}
				}

				break
			}
		}

		// don't touch the org if in the ignoreMap
		if _, ok := orgsToIgnoreMap[orgID]; ok {
			golog.Infof("Ignore migrating setting for entity %s in org %s", entityID, orgID)
			continue
		}

		// ignore setting if no provisioned phone for the organization
		if provisionedPhone == "" {
			golog.Infof("Ignore entity migration for entity %s since no phone number set at org level", entityID)
			continue
		}

		// lookup setting for the entity and set that value for the org
		entitySettingValue, err := settings.GetBooleanValue(context.Background(), settingsClient, &settings.GetValuesRequest{
			NodeID: entityID,
			Keys: []*settings.ConfigKey{
				{
					Key: excommsSettings.ConfigKeySendCallsToVoicemail,
				},
			},
		})
		if err != nil {
			golog.Fatalf(err.Error())
		}

		// apply this value to the org the entity belongs to
		_, err = settingsClient.SetValue(context.Background(), &settings.SetValueRequest{
			NodeID: orgID,
			Value: &settings.Value{
				Key: &settings.ConfigKey{
					Key:    excommsSettings.ConfigKeySendCallsToVoicemail,
					Subkey: provisionedPhone,
				},
				Type: settings.ConfigType_BOOLEAN,
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: entitySettingValue.Value,
					},
				},
			},
		})
		if err != nil {
			golog.Fatalf(err.Error())
		}

		golog.Infof("SUCCESS! Migrated setting for entityID %s (value = %b) to orgID %s phoneNumber %s", entityID, entitySettingValue.Value, orgID, provisionedPhone)
	}

}

func getEntities() ([]string, error) {
	csvFile, err := os.Open(*flagEntityList)
	if err != nil {
		return nil, err
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = '\n'

	var entites []string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		entites = append(entites, row[0])
	}

	return entites, nil
}

func getOrgsToIgnore() ([]string, error) {
	csvFile, err := os.Open(*flagOrgIgnoreList)
	if err != nil {
		return nil, err
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = '\n'

	var orgList []string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		orgList = append(orgList, strings.TrimSpace(row[0]))
	}

	return orgList, nil
}
