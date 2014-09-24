package common

import (
	"reflect"
)

var Forms = map[string]reflect.Type{
	"notify-me":       reflect.TypeOf(NotifyMeForm{}),
	"doctor-interest": reflect.TypeOf(DoctorInterestForm{}),
}

type NotifyMeForm struct {
	Email    string `json:"email"`
	State    string `json:"state"`
	Platform string `json:"platform"`
}

type DoctorInterestForm struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	States  string `json:"states"`
	Comment string `json:"comment"`
}

func (f *NotifyMeForm) TableColumnValues() (string, []string, []interface{}) {
	return "form_notify_me", []string{"email", "state", "platform"}, []interface{}{f.Email, f.State, f.Platform}
}

func (f *DoctorInterestForm) TableColumnValues() (string, []string, []interface{}) {
	return "form_doctor_interest", []string{"name", "email", "states", "comment"}, []interface{}{f.Name, f.Email, f.States, f.Comment}
}
