package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tcolgate/mp3"

	"context"

	"github.com/aws/aws-sdk-go/aws/session"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/settings"
)

type enablePackageCmd struct {
	cnf          *config
	directoryCli directory.DirectoryClient
	settingsCli  settings.SettingsClient
}

func newEnablePackageCmd(cnf *config) (command, error) {
	settingsCli, err := cnf.settingsClient()
	if err != nil {
		return nil, err
	}
	directoryCli, err := cnf.directoryClient()
	if err != nil {
		return nil, err
	}
	return &enablePackageCmd{
		cnf:          cnf,
		directoryCli: directoryCli,
		settingsCli:  settingsCli,
	}, nil
}

func (c *enablePackageCmd) run(args []string) error {
	fs := flag.NewFlagSet("enablepackage", flag.ExitOnError)
	orgEntityID := fs.String("org_entity_id", "", "EntityID of the organization")
	answeringService := fs.Bool("answering_service", false, "Enable answering service?")
	digitalCare := fs.Bool("digital_care", false, "Enable digital care?")
	phoneNumber := fs.String("phone_number", "", "Phone number for which to configure greeting")
	bucket := fs.String("s3_bucket", "", "S3 bucket for where the greeting should be stored")
	fileName := fs.String("file_name", "", "name of file containing the greeting")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)

	ctx := context.Background()

	if !*answeringService && !*digitalCare {
		return errors.New("Specify enabling of digital care of answering service")
	}

	if *orgEntityID == "" {
		*orgEntityID = prompt(scn, "OrgEntityID: ")
	}
	if *orgEntityID == "" {
		return errors.New("EntityID for org required")
	}

	ent, err := lookupAndDisplayEntity(ctx, c.directoryCli, *orgEntityID, nil)
	if err != nil {
		return fmt.Errorf("Failed to lookup entity: %s", err)
	}
	if ent.Type != directory.EntityType_ORGANIZATION {
		return errors.New("Entity is not an organization")
	}

	if *answeringService {
		if err := c.enableAfterHours(scn, orgEntityID, phoneNumber, fileName, bucket); err != nil {
			return errors.Trace(err)
		}
	}

	if *digitalCare {
		if err := c.enableAfterHours(scn, orgEntityID, phoneNumber, fileName, bucket); err != nil {
			return errors.Trace(err)
		}

		if err := c.enableDigitalCare(scn, orgEntityID, fileName, bucket); err != nil {
			return errors.Trace(err)
		}

	}

	return nil
}

func (c *enablePackageCmd) enableDigitalCare(
	scn *bufio.Scanner,
	orgEntityID, fileName, bucket *string) error {

	// turn on video calling
	if err := c.turnOnSetting(*orgEntityID, "video_calling_enabled", ""); err != nil {
		return errors.Trace(err)
	}

	// turn on spruce visit_attachments_enabled
	if err := c.turnOnSetting(*orgEntityID, "visit_attachments_enabled", ""); err != nil {
		return errors.Trace(err)
	}

	// turn on care plans
	if err := c.turnOnSetting(*orgEntityID, "care_plans_enabled", ""); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *enablePackageCmd) enableAfterHours(
	scn *bufio.Scanner,
	orgEntityID, phoneNumber, fileName, bucket *string) error {
	if *phoneNumber == "" {
		*phoneNumber = prompt(scn, "Phone number: ")
	}
	if *phoneNumber == "" {
		return errors.New("Phone number required")
	}

	if *fileName == "" {
		*fileName = prompt(scn, "Filename for mp3: ")
	}
	if *fileName != "" {
		if !strings.HasSuffix(*fileName, ".mp3") {
			return errors.New("File must be an mp3 and end with the extension .mp3")
		}

		if err := c.uploadCustomVoicemail(orgEntityID, phoneNumber, fileName, bucket); err != nil {
			return errors.Trace(err)
		}
	}

	// turn on voicemail transcription
	if err := c.turnOnSetting(*orgEntityID, excommsSettings.ConfigKeyTranscribeVoicemail, ""); err != nil {
		return errors.Trace(err)
	}

	// turn on answering service
	if err := c.turnOnSetting(*orgEntityID, excommsSettings.ConfigKeyAfterHoursVociemailEnabled, *phoneNumber); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *enablePackageCmd) uploadCustomVoicemail(orgEntityID, phoneNumber, fileName, bucket *string) error {
	awsConfig, err := awsutil.Config("us-east-1", "", "", "")
	if err != nil {
		return errors.Trace(err)
	}

	awsSession := session.New(awsConfig)

	store := storage.NewS3(awsSession, *bucket, "media")

	id, err := media.NewID()
	if err != nil {
		return errors.Trace(err)
	}

	mp3File, err := os.Open(*fileName)
	if err != nil {
		return errors.Trace(err)
	}

	// Make sure we can decode at least one frame
	dec := mp3.NewDecoder(mp3File)
	var frame mp3.Frame
	if err := dec.Decode(&frame); err != nil {
		return fmt.Errorf("Failed to decode MP3 frame: %s", err)
	}

	size, err := media.SeekerSize(mp3File)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = store.PutReader(id, mp3File, size, "audio/mpeg", nil)
	if err != nil {
		return errors.Trace(err)
	}

	pn, err := phone.Format(*phoneNumber, phone.E164)
	if err != nil {
		return errors.Trace(err)
	}

	selectionID := excommsSettings.VoicemailOptionCustom

	_, err = c.settingsCli.SetValue(context.Background(), &settings.SetValueRequest{
		NodeID: *orgEntityID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    excommsSettings.ConfigKeyVoicemailOption,
				Subkey: pn,
			},
			Type: settings.ConfigType_SINGLE_SELECT,
			Value: &settings.Value_SingleSelect{
				SingleSelect: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID:               selectionID,
						FreeTextResponse: id,
					},
				},
			},
		},
	})
	if err != nil {
		return errors.Trace(err)
	}

	golog.Infof("Uploaded custom voicemail for %s.%s", *orgEntityID, *phoneNumber)
	return nil
}

func (c *enablePackageCmd) turnOnSetting(nodeID, key, subkey string) error {
	_, err := c.settingsCli.SetValue(context.Background(), &settings.SetValueRequest{
		NodeID: nodeID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    key,
				Subkey: subkey,
			},
			Type: settings.ConfigType_BOOLEAN,
			Value: &settings.Value_Boolean{
				Boolean: &settings.BooleanValue{
					Value: true,
				},
			},
		},
	})
	if err != nil {
		return errors.Trace(err)
	}
	golog.Infof("Turned on %s for %s.%s", key, nodeID, subkey)
	return nil
}
