package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
)

type setGreetingCmd struct {
	cnf         *config
	settingsCli settings.SettingsClient
}

func newSetGreetingCmd(cnf *config) (command, error) {
	settingsCli, err := cnf.settingsClient()
	if err != nil {
		return nil, err
	}

	return &setGreetingCmd{
		cnf:         cnf,
		settingsCli: settingsCli,
	}, nil
}

func (c *setGreetingCmd) run(args []string) error {
	fs := flag.NewFlagSet("setgreeting", flag.ExitOnError)
	orgEntityID := fs.String("org_entity_id", "", "EntityID of the organization")
	key := fs.String("key", "", "Setting key")
	phoneNumber := fs.String("phone_number", "", "Phone number for which to configure greeting")
	bucket := fs.String("s3_bucket", "", "S3 bucket for where the greeting should be stored")
	prefix := fs.String("s3_prefix", "", "prefix for the file on s3")
	fileName := fs.String("file_name", "", "name of file containing the greeting")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	if *orgEntityID == "" {
		*orgEntityID = prompt(scn, "OrgEntityID: ")
	}
	if *orgEntityID == "" {
		return errors.New("EntityID for org required")
	}

	if *key == "" {
		*key = prompt(scn, "Key: ")
	}
	if *key == "" {
		return errors.New("Setting key required")
	}

	if *phoneNumber == "" {
		*phoneNumber = prompt(scn, "Phone number: ")
	}
	if *phoneNumber == "" {
		return errors.New("Phone number required")
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

	if *fileName == "" {
		*fileName = prompt(scn, "Filename for mp3: ")
	}
	if *fileName == "" {
		return errors.New("Filename of file containing mp3 required")
	}

	awsConfig, err := awsutil.Config("us-east-1", "", "", "")
	if err != nil {
		return errors.Trace(err)
	}

	awsSession := session.New(awsConfig)

	store := storage.NewS3(awsSession, *bucket, *prefix)

	id, err := media.NewID()
	if err != nil {
		return errors.Trace(err)
	}

	mp3File, err := os.Open(*fileName)
	if err != nil {
		return errors.Trace(err)
	}

	size, err := media.SeekerSize(mp3File)
	if err != nil {
		return errors.Trace(err)
	}

	mediaLocation, err := store.PutReader(id, mp3File, size, "audio/mpeg", nil)
	if err != nil {
		return errors.Trace(err)
	}

	pn, err := phone.Format(*phoneNumber, phone.E164)
	if err != nil {
		return errors.Trace(err)
	}

	var selectionID string
	switch *key {
	case excommsSettings.ConfigKeyVoicemailOption:
		selectionID = excommsSettings.VoicemailOptionCustom
	case excommsSettings.ConfigKeyAfterHoursGreetingOption:
		selectionID = excommsSettings.AfterHoursGreetingOptionCustom
	default:
		return errors.Trace(fmt.Errorf("Unknown key for setting greeting: %s", *key))
	}

	_, err = c.settingsCli.SetValue(context.Background(), &settings.SetValueRequest{
		NodeID: *orgEntityID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    *key,
				Subkey: pn,
			},
			Type: settings.ConfigType_SINGLE_SELECT,
			Value: &settings.Value_SingleSelect{
				SingleSelect: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID:               selectionID,
						FreeTextResponse: mediaLocation,
					},
				},
			},
		},
	})
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
