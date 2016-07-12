package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
)

// migrateVMSettingCmd provides a script to migrate the custom
// voicemail greeting setting for regular and afterhours
// to contain the mediaID in the freetext response rather than
// the internal URL (of the form s3://us-east-1/bucket/prefix-ID)
// for where the media object is stored. After the migration the setting will be in line with
// how self-service voicemail greetings will be uploaded by the clients.
// The custom voicemail greeting for the afterhours configuration is migrated to the general
// custom voicemail greeting setting to make it possible for providers to set a custom voicemail
// greeting in app and then inform us of whether they want to turn on the afterhours
// configuration. Driving the custom voicemail via a single setting provides
// for a simpler configuration.
type migrateVMSettingCmd struct {
	cnf          *config
	settingsCli  settings.SettingsClient
	directoryCli directory.DirectoryClient
}

func newMigrateVMSettingCmd(cnf *config) (command, error) {
	settingsCli, err := cnf.settingsClient()
	if err != nil {
		return nil, err
	}

	directoryCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}

	return &migrateVMSettingCmd{
		cnf:          cnf,
		directoryCli: directoryCli,
		settingsCli:  settingsCli,
	}, nil
}

func (c *migrateVMSettingCmd) run(args []string) error {
	fs := flag.NewFlagSet("migratevmsettings", flag.ExitOnError)
	afterHours := fs.Bool("afterhours", false, "migrate afterhours voicemail?")
	orgIDsFile := fs.String("org_ids_filename", "", "file containing orgIDs")
	bucket := fs.String("s3_bucket", "", "S3 bucket for where voicemails are to be stored")
	prefix := fs.String("s3_prefix", "", "S3 prefix for voicemails")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *orgIDsFile == "" {
		*orgIDsFile = prompt(scn, "Name of file containing orgIDs: ")
	}
	if *orgIDsFile == "" {
		return errors.New("Filename required")
	}

	if *bucket == "" {
		*bucket = prompt(scn, "S3 Bucket: ")
	}
	if *bucket == "" {
		return errors.New("S3 Bucket required")
	}

	if *prefix == "" {
		*prefix = prompt(scn, "Prefix: ")
	}
	if *prefix == "" {
		return errors.New("S3 Prefix for bucket required")
	}

	orgIDs, err := getOrgIDs(*orgIDsFile)
	if err != nil {
		return errors.Trace(err)
	}

	awsConfig, err := awsutil.Config("us-east-1", "", "", "")
	if err != nil {
		return errors.Trace(err)
	}

	awsSession := session.New(awsConfig)
	store := storage.NewS3(awsSession, *bucket, *prefix)

	// collect Spruce provisioned phone number for all orgs
	sprucePhoneNumbers := make(map[string]string, len(orgIDs))
	for _, orgID := range orgIDs {
		entity, err := directory.SingleEntity(context.Background(), c.directoryCli, &directory.LookupEntitiesRequest{
			LookupKeyType: directory.LookupEntitiesRequest_ENTITY_ID,
			LookupKeyOneof: &directory.LookupEntitiesRequest_EntityID{
				EntityID: orgID,
			},
			RootTypes: []directory.EntityType{directory.EntityType_ORGANIZATION},
			RequestedInformation: &directory.RequestedInformation{
				EntityInformation: []directory.EntityInformation{
					directory.EntityInformation_CONTACTS,
				},
			},
		})
		if err != nil {
			return errors.Trace(fmt.Errorf("Unable to get entity %s: %s", orgID, err))
		}

		var phoneNumber string
		for _, contact := range entity.Contacts {
			if contact.ContactType == directory.ContactType_PHONE && contact.Provisioned {
				phoneNumber = contact.Value
				break
			}
		}
		if phoneNumber == "" {
			return errors.Trace(fmt.Errorf("No spruce phone number found for %s", orgID))
		}

		sprucePhoneNumbers[orgID] = phoneNumber
	}

	// collect location of where voicemail is currently uploaded
	voicemailLocations := make(map[string]string, len(orgIDs))
	for _, orgID := range orgIDs {
		key := "voicemail_option"
		if *afterHours {
			key = "afterhours_greeting_option"
		}

		singleSelectValue, err := settings.GetSingleSelectValue(context.Background(), c.settingsCli, &settings.GetValuesRequest{
			NodeID: orgID,
			Keys: []*settings.ConfigKey{
				{
					Key:    key,
					Subkey: sprucePhoneNumbers[orgID],
				},
			},
		})
		if err != nil {
			return errors.Trace(fmt.Errorf("Unable to get current voicemail location for %s: %s", orgID, err))
		}

		if singleSelectValue.Item.ID == "voicemail_option_default" || singleSelectValue.Item.ID == "afterhours_greeting_option_default" {
			golog.Warningf("Skipping voicemail migration for %s since the voicemail greeting option is the default one", orgID)
			continue
		} else if singleSelectValue.Item.FreeTextResponse == "" {
			return errors.Trace(fmt.Errorf("Expected media location to be set in free text response instead got none for %s", orgID))
		}

		voicemailLocations[orgID] = singleSelectValue.Item.FreeTextResponse
	}

	// now for each of the orgs, go ahead and reupload the voicemail into the generic
	// s3 location and update the settings
	for _, orgID := range orgIDs {

		voicemailLocation := voicemailLocations[orgID]
		if parts := strings.Split(voicemailLocation, "/"); len(parts) == 1 {
			golog.Warningf("Skipping migrating voicemail for %s since it is already in the right format %s", orgID, voicemailLocation)
			continue
		}

		mp3Data, _, err := store.Get(voicemailLocation)
		if err != nil {
			return errors.Trace(fmt.Errorf("unable to read voicemail at location %s for %s: %s", voicemailLocations[orgID], orgID, err))
		}

		mp3Buffer := bytes.NewReader(mp3Data)

		size, err := media.SeekerSize(mp3Buffer)
		if err != nil {
			return errors.Trace(fmt.Errorf("Unable to determine size for media %s for %s: %s", voicemailLocation, orgID, err))
		}

		id, err := media.NewID()
		if err != nil {
			return errors.Trace(err)
		}

		// store the media object at the location it is intended to be stored at
		_, err = store.PutReader(id, mp3Buffer, size, "audio/mpeg", nil)
		if err != nil {
			return errors.Trace(fmt.Errorf("Unable to upload media %s to new location %s for %s: %s ", voicemailLocation, id, orgID, err))
		}

		// now re-set the settings for the org with the new mediaID
		_, err = c.settingsCli.SetValue(context.Background(), &settings.SetValueRequest{
			NodeID: orgID,
			Value: &settings.Value{
				Key: &settings.ConfigKey{
					Key:    "voicemail_option",
					Subkey: sprucePhoneNumbers[orgID],
				},
				Type: settings.ConfigType_SINGLE_SELECT,
				Value: &settings.Value_SingleSelect{
					SingleSelect: &settings.SingleSelectValue{
						Item: &settings.ItemValue{
							ID:               "voicemail_option_custom",
							FreeTextResponse: id,
						},
					},
				},
			},
		})
		if err != nil {
			return errors.Trace(fmt.Errorf("Unable to set setting for %s: %s", orgID, err))
		}
		golog.Infof("Migrated setting for %s", orgID)
	}

	return nil
}

func getOrgIDs(orgIDFileName string) ([]string, error) {
	file, err := os.Open(orgIDFileName)
	if err != nil {
		return nil, errors.Trace(err)
	}

	orgIDs := make([]string, 0)
	r := csv.NewReader(file)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		orgIDs = append(orgIDs, row[0])
	}

	return orgIDs, nil
}
