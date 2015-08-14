package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
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

func (d mockedDataAPI_handlerFTPMembership) CreateFTPMemberships(memberships []*common.FTPMembership) error {
	return nil
}

func (d mockedDataAPI_handlerFTPMembership) DeleteFTPMembership(ftpID, doctorID, pathwayID int64) (int64, error) {
	return 0, nil
}

func (d mockedDataAPI_handlerFTPMembership) PathwayForTag(tag string, opts api.PathwayOption) (*common.Pathway, error) {
	return &common.Pathway{ID: 1}, nil
}

func (d mockedDataAPI_handlerFTPMembership) Pathway(id int64, opts api.PathwayOption) (*common.Pathway, error) {
	return &common.Pathway{ID: id}, nil
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
			ID:        encoding.NewObjectID(1),
			FirstName: "DFN1",
			LastName:  "DLN1",
		},
		&common.Doctor{
			ID:        encoding.NewObjectID(2),
			FirstName: "DFN2",
			LastName:  "DLN2",
		},
	}
	ftpMembershipHandler := newFTPMembershipHandler(mockedDataAPI_handlerFTPMembership{ftp: ftp, memberships: memberships, doctors: doctors})
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
	m.Handle(`/admin/api/treatment_plan/favorite/{id:[0-9]+}/membership`, ftpMembershipHandler)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, resp)
	m.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestHandlerFTPMembershipPOST(t *testing.T) {
	r, err := http.NewRequest("POST", "/admin/api/treatment_plan/favorite/1/membership", strings.NewReader(`{"requests":[{"doctor_id":"2","pathway_tag":"foo"},{"doctor_id":"1","pathway_tag":"foo"}]}`))
	test.OK(t, err)
	ftpMembershipHandler := newFTPMembershipHandler(mockedDataAPI_handlerFTPMembership{})
	m := mux.NewRouter()
	m.Handle(`/admin/api/treatment_plan/favorite/{id:[0-9]+}/membership`, ftpMembershipHandler)
	responseWriter := httptest.NewRecorder()
	m.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, "true\n", responseWriter.Body.String())
	test.Equals(t, http.StatusOK, responseWriter.Code)
}

func TestHandlerFTPMembershipDELETE(t *testing.T) {
	r, err := http.NewRequest("DELETE", "/admin/api/treatment_plan/favorite/1/membership", strings.NewReader(`{"doctor_id":"1","pathway_tag":"foo"}`))
	test.OK(t, err)
	ftpMembershipHandler := newFTPMembershipHandler(mockedDataAPI_handlerFTPMembership{})
	m := mux.NewRouter()
	m.Handle(`/admin/api/treatment_plan/favorite/{id:[0-9]+}/membership`, ftpMembershipHandler)
	responseWriter := httptest.NewRecorder()
	m.ServeHTTP(context.Background(), responseWriter, r)
	test.Equals(t, "true\n", responseWriter.Body.String())
	test.Equals(t, http.StatusOK, responseWriter.Code)
}
