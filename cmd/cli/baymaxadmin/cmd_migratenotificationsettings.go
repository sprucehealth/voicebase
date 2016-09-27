package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"io"
	"os"
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

type migrateNotificationSettingCmd struct {
	cnf          *config
	settingsCli  settings.SettingsClient
	threadingCli threading.ThreadsClient
}

func newMigrateNotificationSettingCmd(cnf *config) (command, error) {
	settingsCli, err := cnf.settingsClient()
	if err != nil {
		return nil, err
	}

	threadingCli, err := cnf.threadingClient()
	if err != nil {
		return nil, err
	}

	return &migrateNotificationSettingCmd{
		cnf:          cnf,
		threadingCli: threadingCli,
		settingsCli:  settingsCli,
	}, nil
}

func (c *migrateNotificationSettingCmd) run(args []string) error {
	ctx := context.Background()
	fs := flag.NewFlagSet("migratenotificationsettings", flag.ExitOnError)
	entityIDsFile := fs.String("entity_ids_filename", "", "file containing orgIDs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	args = fs.Args()

	scn := bufio.NewScanner(os.Stdin)
	if *entityIDsFile == "" {
		*entityIDsFile = prompt(scn, "Name of file containing entity ids: ")
	}
	if *entityIDsFile == "" {
		return errors.New("Filename required")
	}

	entityIDs, err := getEntityIDs(*entityIDsFile)
	if err != nil {
		return errors.Trace(err)
	}

	for _, entID := range entityIDs {
		getResp, err := c.settingsCli.GetValues(ctx, &settings.GetValuesRequest{
			Keys: []*settings.ConfigKey{
				{
					Key: notification.PatientNotificationPreferencesSettingsKey,
				},
				{
					Key: notification.TeamNotificationPreferencesSettingsKey,
				},
			},
			NodeID: entID,
		})
		if err != nil {
			return errors.Trace(err)
		}
		for _, v := range getResp.Values {
			switch v.Key.Key {
			case notification.PatientNotificationPreferencesSettingsKey, notification.TeamNotificationPreferencesSettingsKey:
				// If it's something we care about then get the saved queries for the entity
				sqsResp, err := c.threadingCli.SavedQueries(ctx, &threading.SavedQueriesRequest{
					EntityID: entID,
				})
				if err != nil {
					return errors.Trace(err)
				}
				settingValue := v.GetSingleSelect().Item.ID
				switch v.Key.Key {
				case notification.PatientNotificationPreferencesSettingsKey:
					patientSQ := savedQueryFromList(ctx, sqsResp.SavedQueries, "Patient")
					if patientSQ == nil {
						return errors.Trace(err)
					}
					switch settingValue {
					case notification.ThreadActivityNotificationPreferenceAllMessages:
						if _, err := c.threadingCli.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
							SavedQueryID:         patientSQ.ID,
							NotificationsEnabled: threading.NOTIFICATIONS_ENABLED_UPDATE_TRUE,
						}); err != nil {
							return errors.Trace(err)
						}
					case notification.ThreadActivityNotificationPreferenceReferencedOnly, notification.ThreadActivityNotificationPreferenceOff:
						if _, err := c.threadingCli.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
							SavedQueryID:         patientSQ.ID,
							NotificationsEnabled: threading.NOTIFICATIONS_ENABLED_UPDATE_FALSE,
						}); err != nil {
							return errors.Trace(err)
						}
					}
				case notification.TeamNotificationPreferencesSettingsKey:
					teamSQ := savedQueryFromList(ctx, sqsResp.SavedQueries, "Team")
					if teamSQ == nil {
						return errors.Trace(err)
					}
					switch settingValue {
					case notification.ThreadActivityNotificationPreferenceAllMessages:
						if _, err := c.threadingCli.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
							SavedQueryID:         teamSQ.ID,
							NotificationsEnabled: threading.NOTIFICATIONS_ENABLED_UPDATE_TRUE,
						}); err != nil {
							return errors.Trace(err)
						}
					case notification.ThreadActivityNotificationPreferenceReferencedOnly, notification.ThreadActivityNotificationPreferenceOff:
						if _, err := c.threadingCli.UpdateSavedQuery(ctx, &threading.UpdateSavedQueryRequest{
							SavedQueryID:         teamSQ.ID,
							NotificationsEnabled: threading.NOTIFICATIONS_ENABLED_UPDATE_FALSE,
						}); err != nil {
							return errors.Trace(err)
						}
					}
				}
			}
		}
	}

	return nil
}

func savedQueryFromList(ctx context.Context, savedQueries []*threading.SavedQuery, title string) *threading.SavedQuery {
	for _, sq := range savedQueries {
		if strings.EqualFold(sq.Title, title) {
			return sq
		}
	}
	return nil
}

func getEntityIDs(orgIDFileName string) ([]string, error) {
	file, err := os.Open(orgIDFileName)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var entityIDs []string
	r := csv.NewReader(file)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		entityIDs = append(entityIDs, row[0])
	}

	return entityIDs, nil
}
