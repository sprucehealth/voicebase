package events

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/events/model"
	"github.com/sprucehealth/backend/events/query"
	"github.com/sprucehealth/backend/libs/dbutil"
	"github.com/sprucehealth/backend/libs/errors"
)

const (
	ClientEventDimensions = 24
)

type Client interface {
	InsertServerEvent(sev *model.ServerEvent) error
	InsertWebRequestEvent(wev *model.WebRequestEvent) error
	InsertClientEvent(cevs []*model.ClientEvent) error
	ServerEvents(q *query.ServerEventQuery) ([]*model.ServerEvent, error)
}

type NullClient struct{}

func (NullClient) InsertServerEvent(sev *model.ServerEvent) error {
	return nil
}

func (NullClient) InsertWebRequestEvent(wev *model.WebRequestEvent) error {
	return nil
}

func (NullClient) InsertClientEvent(cevs []*model.ClientEvent) error {
	return nil
}

func (NullClient) ServerEvents(q *query.ServerEventQuery) ([]*model.ServerEvent, error) {
	return nil, nil
}

type client struct {
	db *sql.DB
}

func NewClient(config *config.DB) (Client, error) {
	s := &client{}

	var err error
	s.db, err = config.ConnectPostgres()
	if err != nil {
		return nil, errors.Trace(err)
	}

	return s, nil
}

func (c *client) InsertServerEvent(sev *model.ServerEvent) error {
	_, err := c.db.Exec(
		`INSERT INTO server_event
			(name, timestamp, session_id, account_id, patient_id, doctor_id, visit_id, case_id, treatment_plan_id, role, extra_json)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		sev.Event, sev.Timestamp, sev.SessionID, sev.AccountID, sev.PatientID,
		sev.DoctorID, sev.VisitID, sev.CaseID, sev.TreatmentPlanID, sev.Role, sev.ExtraJSON)
	if err != nil {
		return errors.Trace(fmt.Errorf("Failed to log ServerEvent %+v: %s", sev, err.Error()))
	}
	return nil
}

func (c *client) InsertWebRequestEvent(wev *model.WebRequestEvent) error {
	_, err := c.db.Exec(
		`INSERT INTO web_request_event
			(service, path, timestamp, request_id, status_code, method, url, remote_addr, content_type, user_agent, referrer, 
			response_time, server, account_id, device_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		wev.Service, wev.Path, wev.Timestamp, wev.RequestID, wev.StatusCode, wev.Method, wev.URL, wev.RemoteAddr,
		wev.ContentType, wev.UserAgent, wev.Referrer, wev.ResponseTime, wev.Server, wev.AccountID, wev.DeviceID)
	if err != nil {
		return errors.Trace(fmt.Errorf("Failed to log WebRequestEvent %+v: %s", wev, err.Error()))
	}
	return nil
}

func (c *client) InsertClientEvent(cevs []*model.ClientEvent) error {
	values := make([]interface{}, 0, len(cevs)*ClientEventDimensions)
	params := make([]string, len(cevs))
	for i, cev := range cevs {
		params[i] = `(` + dbutil.PostgresArgs((i*ClientEventDimensions)+1, ClientEventDimensions) + `)`
		values = append(values, cev.Event, cev.Timestamp, cev.Error, cev.SessionID, cev.DeviceID, cev.AccountID, cev.PatientID, cev.DoctorID, cev.VisitID, cev.CaseID,
			cev.ScreenID, cev.QuestionID, cev.TimeSpent, cev.AppType, cev.AppEnv, cev.AppBuild, cev.Platform, cev.PlatformVersion, cev.DeviceType,
			cev.DeviceModel, cev.ScreenWidth, cev.ScreenHeight, cev.ScreenResolution, cev.ExtraJSON)
	}
	_, err := c.db.Exec(
		`INSERT INTO client_event
			(name, timestamp, error, session_id, device_id, account_id, patient_id, doctor_id, visit_id, case_id, screen_id, question_id,
			time_spent, app_type, app_env, app_build, platform, platform_version, device_type, device_model, screen_width, screen_height, 
			screen_resolution, extra_json)
			VALUES `+strings.Join(params, `,`), values...)
	if err != nil {
		eventMsgs := make([]string, len(cevs))
		for i, cev := range cevs {
			eventMsgs[i] = fmt.Sprintf("%+v", cev)
		}
		return errors.Trace(fmt.Errorf("Failed to log ClientEvents %v: %s", eventMsgs, err.Error()))
	}
	return nil
}

func (c *client) ServerEvents(q *query.ServerEventQuery) ([]*model.ServerEvent, error) {
	s, v := q.SQL()
	rows, err := c.db.Query(s, v...)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	var events []*model.ServerEvent
	for rows.Next() {
		ev := &model.ServerEvent{}
		if err := rows.Scan(&ev.Event, &ev.Timestamp, &ev.SessionID, &ev.AccountID, &ev.PatientID, &ev.DoctorID,
			&ev.VisitID, &ev.CaseID, &ev.TreatmentPlanID, &ev.Role, &ev.ExtraJSON); err != nil {
			return nil, errors.Trace(err)
		}
		events = append(events, ev)
	}
	return events, errors.Trace(rows.Err())
}
