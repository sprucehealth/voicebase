package main

import (
	"flag"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	excommsSettings "github.com/sprucehealth/backend/cmd/svc/excomms/settings"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/media"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/svc/settings"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	flagSettingsAddr      = flag.String("settings_addr", "", "`host:port` for settings service")
	flagVoicemailFileName = flag.String("link_mp3", "", "voicemail file location in mp3 format")
	flagAWSAccessKey      = flag.String("aws_access_key", "", "Access `key` for AWS")
	flagAWSSecretKey      = flag.String("aws_secret_key", "", "Secret `key` for AWS")
	flagBucket            = flag.String("bucket", "", "bucket where voicemail is to be stored")
	flagPrefix            = flag.String("prefix", "voicemail-greetings", "prefix for storage bucket")
	flagEntityID          = flag.String("org_entity_id", "", "entityID of the organization in encoded form")
	flagPhoneNumber       = flag.String("phone_number", "", "phone number for which to turn on custom voicemail greeting")
)

func main() {
	flag.Parse()
	validate()

	awsConfig, err := awsutil.Config("us-east-1", *flagAWSAccessKey, *flagAWSSecretKey, "")
	if err != nil {
		golog.Fatalf(err.Error())
	}
	awsSession := session.New(awsConfig)

	store := storage.NewS3(awsSession, *flagBucket, *flagPrefix)

	settingsConn, err := grpc.Dial(
		*flagSettingsAddr,
		grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with settings service: %s", err.Error())
		return
	}
	defer settingsConn.Close()

	settingsClient := settings.NewSettingsClient(settingsConn)

	// go ahead and upload media file first
	id, err := media.NewID()
	if err != nil {
		golog.Fatalf(err.Error())
	}

	// open the mp3 file for reading
	mp3File, err := os.Open(*flagVoicemailFileName)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	size, err := common.SeekerSize(mp3File)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	mediaLocation, err := store.PutReader(id, mp3File, size, "audio/mpeg", nil)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	pn, err := phone.Format(*flagPhoneNumber, phone.E164)
	if err != nil {
		golog.Fatalf(err.Error())
	}

	_, err = settingsClient.SetValue(context.Background(), &settings.SetValueRequest{
		NodeID: *flagEntityID,
		Value: &settings.Value{
			Key: &settings.ConfigKey{
				Key:    excommsSettings.ConfigKeyVoicemailOption,
				Subkey: pn,
			},
			Type: settings.ConfigType_SINGLE_SELECT,
			Value: &settings.Value_SingleSelect{
				SingleSelect: &settings.SingleSelectValue{
					Item: &settings.ItemValue{
						ID:               excommsSettings.VoicemailOptionCustom,
						FreeTextResponse: mediaLocation,
					},
				},
			},
		},
	})
	if err != nil {
		golog.Fatalf(err.Error())
	}

	golog.Infof("SUCCESS! Custom voicemail configured for entity %s phone number %s", *flagEntityID, pn)
}

func validate() {
	if *flagSettingsAddr == "" {
		golog.Fatalf("setting service address not specified")
	} else if *flagVoicemailFileName == "" {
		golog.Fatalf("voicemail filename not specified")
	} else if *flagBucket == "" {
		golog.Fatalf("bucket not specified")
	} else if *flagPrefix == "" {
		golog.Fatalf("prefix for storage file not specified")
	} else if *flagEntityID == "" {
		golog.Fatalf("entity id for which to turn on custom voicemail not specified")
	} else if *flagPhoneNumber == "" {
		golog.Fatalf("phone number for which to turn on custom voicemail not specified")
	}
}
