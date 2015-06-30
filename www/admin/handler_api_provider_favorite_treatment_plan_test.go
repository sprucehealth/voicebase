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

type mockedDataAPI_handlerProviderFTP struct {
	api.DataAPI
	doctors     []*common.Doctor
	memberships []*common.FTPMembership
	ftp         *common.FavoriteTreatmentPlan
}

func (d mockedDataAPI_handlerProviderFTP) FavoriteTreatmentPlan(id int64) (*common.FavoriteTreatmentPlan, error) {
	return d.ftp, nil
}

func (d mockedDataAPI_handlerProviderFTP) FTPMembershipsForDoctor(ftpID int64) ([]*common.FTPMembership, error) {
	return d.memberships, nil
}

func (d mockedDataAPI_handlerProviderFTP) Pathway(id int64, opts api.PathwayOption) (*common.Pathway, error) {
	return &common.Pathway{ID: 1, Name: "Pathway"}, nil
}

func TestHandlerProviderFTPGETSuccess(t *testing.T) {
	r, err := http.NewRequest("GET", "/admin/api/providers/1/treatment_plan/favorite", nil)
	test.OK(t, err)
	ftp := &common.FavoriteTreatmentPlan{
		Name: "Foo",
	}
	memberships := []*common.FTPMembership{
		{
			DoctorID:          1,
			ClinicalPathwayID: 1,
		},
		{
			DoctorID:          1,
			ClinicalPathwayID: 2,
		},
	}
	doctors := []*common.Doctor{
		{
			ID:        encoding.ObjectID{Int64Value: 1},
			FirstName: "DFN1",
			LastName:  "DLN1",
		},
		{
			ID:        encoding.ObjectID{Int64Value: 2},
			FirstName: "DFN2",
			LastName:  "DLN2",
		},
	}
	dataAPI := mockedDataAPI_handlerProviderFTP{DataAPI: &api.DataService{}, ftp: ftp, memberships: memberships, doctors: doctors}
	tresp, err := responses.TransformFTPToResponse(dataAPI, nil, 1, ftp, "")
	test.OK(t, err)
	providerFTPHandler := NewProviderFTPHandler(dataAPI, nil)
	resp := providerFTPGETResponse{
		FavoriteTreatmentPlans: map[string][]*responses.FavoriteTreatmentPlan{
			"Pathway": []*responses.FavoriteTreatmentPlan{tresp, tresp},
		},
	}
	m := mux.NewRouter()
	m.HandleFunc(`/admin/api/providers/{id:[0-9]+}/treatment_plan/favorite`, providerFTPHandler.ServeHTTP)
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, resp)
	m.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}
