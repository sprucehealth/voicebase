package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
)

type mockedDataAPI_handlerFTPMembership struct {
	api.DataAPI
	doctors      []*common.Doctor
	memberships  []*common.FTPMembership
	ftp          *common.FavoriteTreatmentPlan
	CreateCalled bool
	DeleteCalled bool
}

func (d mockedDataAPI_handlerFTPMembership) FavoriteTreatmentPlan(id int64) (*common.FavoriteTreatmentPlan, error) {
	return d.ftp, nil
}

func (d mockedDataAPI_handlerFTPMembership) FTPMemberships(ftpID int64) ([]*common.FTPMembership, error) {
	return d.memberships, nil
}

func (d mockedDataAPI_handlerFTPMembership) Doctors(id []int64) ([]*common.Doctor, error) {
	return d.doctors, nil
}

func (d mockedDataAPI_handlerFTPMembership) CreateFTPMembership(ftpID, doctorID, pathwayID int64) (int64, error) {
	return 0, nil
}

func (d mockedDataAPI_handlerFTPMembership) DeleteFTPMembership(ftpID, doctorID, pathwayID int64) (int64, error) {
	return 0, nil
}

func (d mockedDataAPI_handlerFTPMembership) PathwayForTag(tag string, opts api.PathwayOption) (*common.Pathway, error) {
	return &common.Pathway{ID: 1}, nil
}

func TestHandlerFTPMembershipGETSuccess(t *testing.T) {
	r, err := http.NewRequest("GET", "/admin/api/treatment_plan/favorite/1/membership", nil)
	test.OK(t, err)
	ftp := &common.FavoriteTreatmentPlan{
		Name: "Foo",
	}
	memberships := []*common.FTPMembership{
		&common.FTPMembership{
			DoctorID:          1,
			ClinicalPathwayID: 1,
		},
		&common.FTPMembership{
			DoctorID:          1,
			ClinicalPathwayID: 2,
		},
		&common.FTPMembership{
			DoctorID:          2,
			ClinicalPathwayID: 1,
		},
	}
	doctors := []*common.Doctor{
		&common.Doctor{
			DoctorID:  encoding.ObjectID{Int64Value: 1},
			FirstName: "DFN1",
			LastName:  "DLN1",
		},
		&common.Doctor{
			DoctorID:  encoding.ObjectID{Int64Value: 2},
			FirstName: "DFN2",
			LastName:  "DLN2",
		},
	}
	ftpMembershipHandler := NewFTPMembershipHandler(mockedDataAPI_handlerFTPMembership{DataAPI: &api.DataService{}, ftp: ftp, memberships: memberships, doctors: doctors})
	resp := ftpMembershipGETResponse{
		Name: "Foo",
		Memberships: []*responses.FavoriteTreatmentPlanMembership{
			&responses.FavoriteTreatmentPlanMembership{
				DoctorID:  1,
				FirstName: "DFN1",
				LastName:  "DLN1",
				PathwayID: 1,
			},
			&responses.FavoriteTreatmentPlanMembership{
				DoctorID:  1,
				FirstName: "DFN1",
				LastName:  "DLN1",
				PathwayID: 2,
			},
			&responses.FavoriteTreatmentPlanMembership{
				DoctorID:  2,
				FirstName: "DFN2",
				LastName:  "DLN2",
				PathwayID: 1,
			},
		},
	}
	m := mux.NewRouter()
	m.HandleFunc(`/admin/api/treatment_plan/favorite/{id:[0-9]+}/membership`, ftpMembershipHandler.ServeHTTP)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, resp)
	m.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestHandlerFTPMembershipPOST(t *testing.T) {
	r, err := http.NewRequest("POST", "/admin/api/treatment_plan/favorite/1/membership", strings.NewReader(`{"doctor_id":"1","pathway_tag":"foo"}`))
	test.OK(t, err)
	ftpMembershipHandler := NewFTPMembershipHandler(mockedDataAPI_handlerFTPMembership{DataAPI: &api.DataService{}})
	m := mux.NewRouter()
	m.HandleFunc(`/admin/api/treatment_plan/favorite/{id:[0-9]+}/membership`, ftpMembershipHandler.ServeHTTP)
	responseWriter := httptest.NewRecorder()
	m.ServeHTTP(responseWriter, r)
	test.Equals(t, "", string(responseWriter.Body.Bytes()))
}

func TestHandlerFTPMembershipDELETE(t *testing.T) {
	r, err := http.NewRequest("DELETE", "/admin/api/treatment_plan/favorite/1/membership", strings.NewReader(`{"doctor_id":"1","pathway_tag":"foo"}`))
	test.OK(t, err)
	ftpMembershipHandler := NewFTPMembershipHandler(mockedDataAPI_handlerFTPMembership{DataAPI: &api.DataService{}})
	m := mux.NewRouter()
	m.HandleFunc(`/admin/api/treatment_plan/favorite/{id:[0-9]+}/membership`, ftpMembershipHandler.ServeHTTP)
	responseWriter := httptest.NewRecorder()
	m.ServeHTTP(responseWriter, r)
	test.Equals(t, "", string(responseWriter.Body.Bytes()))
}
