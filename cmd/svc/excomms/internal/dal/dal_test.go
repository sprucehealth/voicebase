package dal

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testsql"
)

const schemaGlob = "schema/*.sql"

func TestMedia(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	clk := clock.NewManaged(time.Unix(1e9, 0))

	dal := New(dt.DB, clk)

	med := &models.Media{
		ID:   "media1",
		Type: "image/jpeg",
		Name: ptr.String("boo"),
	}
	test.OK(t, dal.StoreMedia([]*models.Media{med}))
	ms, err := dal.LookupMedia([]string{"media1"})
	test.OK(t, err)
	test.Equals(t, 1, len(ms))
	test.Equals(t, med, ms["media1"])
}

func TestProxyPhoneNumberReservation(t *testing.T) {
	dt := testsql.Setup(t, schemaGlob)
	defer dt.Cleanup(t)

	clk := clock.NewManaged(time.Unix(1e9, 0))
	dal := New(dt.DB, clk)

	proxyPN := phone.Number("+14155550001")
	origPN := phone.Number("+14155550002")
	destPN := phone.Number("+14155550003")
	provPN := phone.Number("+14155550003")
	expRes := &models.ProxyPhoneNumberReservation{
		ProxyPhoneNumber:       proxyPN,
		OriginatingPhoneNumber: origPN,
		DestinationPhoneNumber: destPN,
		ProvisionedPhoneNumber: provPN,
		DestinationEntityID:    "dest",
		OwnerEntityID:          "owner",
		OrganizationID:         "org",
		Expires:                clk.Now().Add(time.Hour).Truncate(time.Second),
	}
	test.OK(t, dal.CreateProxyPhoneNumberReservation(expRes))

	res, err := dal.ActiveProxyPhoneNumberReservation(&origPN, nil, nil)
	test.OK(t, err)
	expRes.Created = res.Created
	test.Equals(t, expRes, res)

	res, err = dal.ActiveProxyPhoneNumberReservation(nil, &destPN, nil)
	test.OK(t, err)
	test.Equals(t, expRes, res)

	res, err = dal.ActiveProxyPhoneNumberReservation(nil, nil, &proxyPN)
	test.OK(t, err)
	test.Equals(t, expRes, res)

	res, err = dal.ActiveProxyPhoneNumberReservation(&origPN, &destPN, &proxyPN)
	test.OK(t, err)
	test.Equals(t, expRes, res)

	newExpires := clk.Now().Add(24 * time.Hour).Truncate(time.Second)
	n, err := dal.UpdateActiveProxyPhoneNumberReservation(origPN, &destPN, nil, &ProxyPhoneNumberReservationUpdate{Expires: &newExpires})
	test.OK(t, err)
	test.Equals(t, int64(1), n)
	res, err = dal.ActiveProxyPhoneNumberReservation(&origPN, &destPN, &proxyPN)
	test.OK(t, err)
	test.Equals(t, newExpires, res.Expires)
}
