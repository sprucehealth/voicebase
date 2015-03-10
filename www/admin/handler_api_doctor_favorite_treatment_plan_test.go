package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
)

type mockedDataAPI_handlerDoctorFTP struct {
	api.DataAPI
	doctors     []*common.Doctor
	memberships []*common.FTPMembership
	ftp         *common.FavoriteTreatmentPlan
}

func (d mockedDataAPI_handlerDoctorFTP) FavoriteTreatmentPlan(id int64) (*common.FavoriteTreatmentPlan, error) {
	return d.ftp, nil
}

func (d mockedDataAPI_handlerDoctorFTP) FTPMembershipsForDoctor(ftpID int64) ([]*common.FTPMembership, error) {
	return d.memberships, nil
}

func (d mockedDataAPI_handlerDoctorFTP) Pathway(id int64, opts api.PathwayOption) (*common.Pathway, error) {
	return &common.Pathway{ID: 1, Name: "Pathway"}, nil
}

func TestHandlerDoctorFTPGETSuccess(t *testing.T) {
	r, err := http.NewRequest("GET", "/admin/api/doctors/1/treatment_plan/favorite", nil)
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
	dataAPI := mockedDataAPI_handlerDoctorFTP{DataAPI: &api.DataService{}, ftp: ftp, memberships: memberships, doctors: doctors}
	tresp, err := responses.TransformFTPToResponse(dataAPI, nil, 1, ftp, "")
	test.OK(t, err)
	doctorFTPHandler := NewDoctorFTPHandler(dataAPI, nil)
	resp := doctorFTPGETResponse{
		FavoriteTreatmentPlans: map[string][]*responses.FavoriteTreatmentPlan{
			"Pathway": []*responses.FavoriteTreatmentPlan{tresp, tresp},
		},
	}
	m := mux.NewRouter()
	m.HandleFunc(`/admin/api/doctors/{id:[0-9]+}/treatment_plan/favorite`, doctorFTPHandler.ServeHTTP)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, resp)
	m.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}
