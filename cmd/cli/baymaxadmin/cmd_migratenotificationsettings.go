package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"flag"
	"io"
	"os"
	"strings"
	"time"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/notification"
	"github.com/sprucehealth/backend/svc/settings"
	"github.com/sprucehealth/backend/svc/threading"
)

type migrateNotificationSettingCmd struct {
	cnf          *config
	settingsCli  settings.SettingsClient
	threadingCli threading.ThreadsClient
	directoryDB  *sql.DB
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

	directoryDB, err := cnf.db("directory")
	if err != nil {
		return nil, err
	}
	return &migrateNotificationSettingCmd{
		cnf:          cnf,
		threadingCli: threadingCli,
		settingsCli:  settingsCli,
		directoryDB:  directoryDB,
	}, nil
}

func (c *migrateNotificationSettingCmd) run(args []string) error {
	ctx := context.Background()
	fs := flag.NewFlagSet("migratenotificationsettings", flag.ExitOnError)
	args = fs.Args()

	entityIDs, err := internalEntityIDs(c.directoryDB)
	if err != nil {
		return errors.Trace(err)
	}

	golog.Infof("Entites: %d - %d Minutes Estimated", len(entityIDs), len(entityIDs)/60)
	for i, entID := range entityIDs {
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
						golog.Errorf("Entity %s has no Patient saved query. Ignoring", entID)
						continue
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
						golog.Errorf("Entity %s has no Team saved query. Ignoring", entID)
						continue
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
		if i%25 == 0 {
			golog.Infof("%d completed", i)
		}
		time.Sleep(time.Second)
	}
	golog.Infof("Completed: %d", len(entityIDs))

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
