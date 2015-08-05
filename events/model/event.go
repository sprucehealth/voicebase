package model

import (
	"time"

	"github.com/sprucehealth/backend/analytics"
)

type ClientEvent struct {
	Event            string
	Timestamp        time.Time
	Error            *string
	SessionID        string
	DeviceID         string
	AccountID        *int64
	PatientID        *int64
	DoctorID         *int64
	CaseID           *int64
	VisitID          *int64
	ScreenID         *string
	QuestionID       *string
	TimeSpent        *float64
	AppType          *string
	AppEnv           *string
	AppVersion       *string
	AppBuild         *string
	Platform         *string
	PlatformVersion  *string
	DeviceType       *string
	DeviceModel      *string
	ScreenWidth      *int
	ScreenHeight     *int
	ScreenResolution *string
	ExtraJSON        *string
}

func TransformClientEvent(cev *analytics.ClientEvent) *ClientEvent {
	timestamp := time.Time(cev.Timestamp)
	return &ClientEvent{
		Event:            cev.Event,
		Timestamp:        timestamp,
		Error:            stringPtr(cev.Error),
		SessionID:        cev.SessionID,
		DeviceID:         cev.DeviceID,
		AccountID:        int64Ptr(cev.AccountID),
		PatientID:        int64Ptr(cev.PatientID),
		DoctorID:         int64Ptr(cev.DoctorID),
		CaseID:           int64Ptr(cev.CaseID),
		VisitID:          int64Ptr(cev.VisitID),
		ScreenID:         stringPtr(cev.ScreenID),
		QuestionID:       stringPtr(cev.QuestionID),
		TimeSpent:        cev.TimeSpent,
		AppType:          stringPtr(cev.AppType),
		AppEnv:           stringPtr(cev.AppVersion),
		AppVersion:       stringPtr(cev.AppVersion),
		AppBuild:         stringPtr(cev.AppBuild),
		Platform:         stringPtr(cev.PlatformVersion),
		PlatformVersion:  stringPtr(cev.PlatformVersion),
		DeviceType:       stringPtr(cev.DeviceType),
		DeviceModel:      stringPtr(cev.DeviceModel),
		ScreenWidth:      intPtr(cev.ScreenWidth),
		ScreenHeight:     intPtr(cev.ScreenHeight),
		ScreenResolution: stringPtr(cev.ScreenResolution),
		ExtraJSON:        stringPtr(cev.ExtraJSON),
	}
}

type ServerEvent struct {
	Event           string
	Timestamp       time.Time
	SessionID       *string
	AccountID       *int64
	PatientID       *int64
	DoctorID        *int64
	VisitID         *int64
	CaseID          *int64
	TreatmentPlanID *int64
	Role            *string
	ExtraJSON       *string
}

func TransformServerEvent(sev *analytics.ServerEvent) *ServerEvent {
	timestamp := time.Time(sev.Timestamp)
	return &ServerEvent{
		Event:           sev.Event,
		Timestamp:       timestamp,
		SessionID:       stringPtr(sev.SessionID),
		AccountID:       int64Ptr(sev.AccountID),
		PatientID:       int64Ptr(sev.PatientID),
		DoctorID:        int64Ptr(sev.DoctorID),
		VisitID:         int64Ptr(sev.VisitID),
		CaseID:          int64Ptr(sev.CaseID),
		TreatmentPlanID: int64Ptr(sev.TreatmentPlanID),
		Role:            stringPtr(sev.Role),
		ExtraJSON:       stringPtr(sev.ExtraJSON),
	}
}

func FromServerEventModel(sevm *ServerEvent) *analytics.ServerEvent {
	timestamp := analytics.Time(sevm.Timestamp)
	return &analytics.ServerEvent{
		Event:           sevm.Event,
		Timestamp:       timestamp,
		SessionID:       stringFromPtr(sevm.SessionID),
		AccountID:       int64FromPtr(sevm.AccountID),
		PatientID:       int64FromPtr(sevm.PatientID),
		VisitID:         int64FromPtr(sevm.VisitID),
		CaseID:          int64FromPtr(sevm.CaseID),
		TreatmentPlanID: int64FromPtr(sevm.TreatmentPlanID),
		Role:            stringFromPtr(sevm.Role),
		ExtraJSON:       stringFromPtr(sevm.ExtraJSON),
	}
}

type WebRequestEvent struct {
	Service      string
	Path         string
	Timestamp    time.Time
	RequestID    uint64
	StatusCode   int
	Method       string
	URL          string
	RemoteAddr   *string
	ContentType  *string
	UserAgent    *string
	Referrer     *string
	ResponseTime int
	Server       string
	AccountID    *int64
	DeviceID     *string
}

func TransformWebRequestEvent(wev *analytics.WebRequestEvent) *WebRequestEvent {
	timestamp := time.Time(wev.Timestamp)
	return &WebRequestEvent{
		Service:      wev.Service,
		Path:         wev.Path,
		Timestamp:    timestamp,
		RequestID:    wev.RequestID,
		StatusCode:   wev.StatusCode,
		Method:       wev.Method,
		URL:          wev.URL,
		RemoteAddr:   stringPtr(wev.RemoteAddr),
		ContentType:  stringPtr(wev.ContentType),
		UserAgent:    stringPtr(wev.UserAgent),
		Referrer:     stringPtr(wev.Referrer),
		ResponseTime: wev.ResponseTime,
		Server:       wev.Server,
		AccountID:    int64Ptr(wev.AccountID),
		DeviceID:     stringPtr(wev.DeviceID),
	}
}

func int64Ptr(i int64) *int64 {
	if i == 0 {
		return nil
	}
	return &i
}

func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

func float64Ptr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func int64FromPtr(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func intFromPtr(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func float64FromPtr(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func stringFromPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
