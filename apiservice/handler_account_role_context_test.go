package apiservice

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type accoutRoleContextHandlerAccountRoleSubHandler struct {
	t                                      *testing.T
	accoutRoleContextHandlerAccountRoleDAL *accoutRoleContextHandlerAccountRoleDAL
	assert                                 func(*testing.T, context.Context, *accoutRoleContextHandlerAccountRoleDAL)
}

func (h *accoutRoleContextHandlerAccountRoleSubHandler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h.assert(h.t, ctx, h.accoutRoleContextHandlerAccountRoleDAL)
}

type accoutRoleContextHandlerAccountRoleDAL struct {
	getPatientFromAccountIDCallCount int
	getPatientFromAccountIDParam     int64
	getPatientFromAccountIDErr       error
	getPatientFromAccountID          *common.Patient
	getDoctorFromAccountIDCallCount  int
	getDoctorFromAccountIDParam      int64
	getDoctorFromAccountIDErr        error
	getDoctorFromAccountID           *common.Doctor
}

func (s *accoutRoleContextHandlerAccountRoleDAL) GetPatientFromAccountID(accountID int64) (*common.Patient, error) {
	s.getPatientFromAccountIDParam = accountID
	return s.getPatientFromAccountID, s.getPatientFromAccountIDErr
}

func (s *accoutRoleContextHandlerAccountRoleDAL) GetDoctorFromAccountID(accountID int64) (*common.Doctor, error) {
	s.getDoctorFromAccountIDParam = accountID
	return s.getDoctorFromAccountID, s.getDoctorFromAccountIDErr
}

type accountRoleContextHandlerTestData struct {
	requestMethod                          string
	accountRole                            string
	accoutRoleContextHandlerAccountRoleDAL *accoutRoleContextHandlerAccountRoleDAL
	methods                                []string
	assert                                 func(*testing.T, context.Context, *accoutRoleContextHandlerAccountRoleDAL)
	code                                   int
}

func TestAccountRoleContextHandler(t *testing.T) {
	var accountID int64 = 100
	var nild *common.Doctor
	var nilp *common.Patient
	patient := &common.Patient{ID: common.PatientID{ObjectID: encoding.NewObjectID(1)}}
	doctor := &common.Doctor{ID: encoding.NewObjectID(2)}
	testData := []accountRoleContextHandlerTestData{
		{
			requestMethod:                          httputil.Get,
			accoutRoleContextHandlerAccountRoleDAL: &accoutRoleContextHandlerAccountRoleDAL{},
			assert: func(t *testing.T, ctxt context.Context, dal *accoutRoleContextHandlerAccountRoleDAL) {
				test.Equals(t, 0, dal.getPatientFromAccountIDCallCount)
				test.Equals(t, 0, dal.getDoctorFromAccountIDCallCount)
				d, ok := CtxDoctor(ctxt)
				test.Equals(t, false, ok)
				test.Equals(t, nild, d)
				p, ok := CtxPatient(ctxt)
				test.Equals(t, false, ok)
				test.Equals(t, nilp, p)
				c, ok := CtxCC(ctxt)
				test.Equals(t, false, ok)
				test.Equals(t, nild, c)
			},
			methods: []string{httputil.Put},
			code:    http.StatusOK,
		},
		{
			requestMethod: httputil.Get,
			accountRole:   api.RolePatient,
			accoutRoleContextHandlerAccountRoleDAL: &accoutRoleContextHandlerAccountRoleDAL{
				getPatientFromAccountIDErr: errors.New("Foo"),
			},
			code: http.StatusInternalServerError,
		},
		{
			requestMethod: httputil.Get,
			accountRole:   api.RolePatient,
			accoutRoleContextHandlerAccountRoleDAL: &accoutRoleContextHandlerAccountRoleDAL{
				getPatientFromAccountID: patient,
			},
			assert: func(t *testing.T, ctxt context.Context, dal *accoutRoleContextHandlerAccountRoleDAL) {
				test.Equals(t, 0, dal.getPatientFromAccountIDCallCount)
				test.Equals(t, 0, dal.getDoctorFromAccountIDCallCount)
				d, _ := CtxDoctor(ctxt)
				test.Equals(t, nild, d)
				p, _ := CtxPatient(ctxt)
				test.Equals(t, patient, p)
				c, _ := CtxCC(ctxt)
				test.Equals(t, nild, c)
			},
			code: http.StatusOK,
		},
		{
			requestMethod: httputil.Get,
			accountRole:   api.RoleDoctor,
			accoutRoleContextHandlerAccountRoleDAL: &accoutRoleContextHandlerAccountRoleDAL{
				getDoctorFromAccountIDErr: errors.New("Foo"),
			},
			code: http.StatusInternalServerError,
		},
		{
			requestMethod: httputil.Get,
			accountRole:   api.RoleDoctor,
			accoutRoleContextHandlerAccountRoleDAL: &accoutRoleContextHandlerAccountRoleDAL{
				getDoctorFromAccountID: doctor,
			},
			assert: func(t *testing.T, ctxt context.Context, dal *accoutRoleContextHandlerAccountRoleDAL) {
				test.Equals(t, 0, dal.getPatientFromAccountIDCallCount)
				test.Equals(t, 0, dal.getDoctorFromAccountIDCallCount)
				d, _ := CtxDoctor(ctxt)
				test.Equals(t, doctor, d)
				p, _ := CtxPatient(ctxt)
				test.Equals(t, nilp, p)
				c, _ := CtxCC(ctxt)
				test.Equals(t, nild, c)
			},
			code: http.StatusOK,
		},
		{
			requestMethod: httputil.Get,
			accountRole:   api.RoleCC,
			accoutRoleContextHandlerAccountRoleDAL: &accoutRoleContextHandlerAccountRoleDAL{
				getDoctorFromAccountIDErr: errors.New("Foo"),
			},
			code: http.StatusInternalServerError,
		},
		{
			requestMethod: httputil.Get,
			accountRole:   api.RoleCC,
			accoutRoleContextHandlerAccountRoleDAL: &accoutRoleContextHandlerAccountRoleDAL{
				getDoctorFromAccountID: doctor,
			},
			assert: func(t *testing.T, ctxt context.Context, dal *accoutRoleContextHandlerAccountRoleDAL) {
				test.Equals(t, 0, dal.getPatientFromAccountIDCallCount)
				test.Equals(t, 0, dal.getDoctorFromAccountIDCallCount)
				d, _ := CtxDoctor(ctxt)
				test.Equals(t, nild, d)
				p, _ := CtxPatient(ctxt)
				test.Equals(t, nilp, p)
				c, _ := CtxCC(ctxt)
				test.Equals(t, doctor, c)
			},
			code: http.StatusOK,
		},
		{
			requestMethod:                          httputil.Get,
			accountRole:                            "Unknown role",
			accoutRoleContextHandlerAccountRoleDAL: &accoutRoleContextHandlerAccountRoleDAL{},
			code: http.StatusInternalServerError,
		},
	}
	for _, td := range testData {
		w := httptest.NewRecorder()
		r, err := http.NewRequest(td.requestMethod, "", nil)
		test.OK(t, err)
		NewAccountRoleContextHandler(
			&accoutRoleContextHandlerAccountRoleSubHandler{t, td.accoutRoleContextHandlerAccountRoleDAL, td.assert},
			td.accoutRoleContextHandlerAccountRoleDAL,
			td.methods...).ServeHTTP(CtxWithAccount(context.Background(), &common.Account{ID: accountID, Role: td.accountRole}), w, r)
		test.Equals(t, td.code, w.Code)
	}
}
