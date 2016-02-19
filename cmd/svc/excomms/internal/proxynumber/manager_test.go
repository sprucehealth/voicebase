package proxynumber

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	dalmock "github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal/mock"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/test"
)

func TestReserveNumber_NewReservation(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	originatingPhoneNumber := phone.Number("+12068773590")
	destinationPhoneNumber := phone.Number("+17348465522")
	proxyPhoneNumber := phone.Number("+11234567890")
	destinationEntityID := "d1"
	sourceEntityID := "s1"
	organizationID := "o1"
	mclock := clock.NewManaged(time.Now())

	md.Expect(mock.NewExpectation(md.ActiveProxyPhoneNumberReservation, originatingPhoneNumber, phone.Ptr(destinationPhoneNumber), phone.Ptr(phone.Number(""))).
		WithReturns(nil, dal.ErrProxyPhoneNumberReservationNotFound))
	md.Expect(mock.NewExpectation(md.AvailableProxyPhoneNumbers, originatingPhoneNumber).WithReturns(
		[]*models.ProxyPhoneNumber{
			{
				PhoneNumber: proxyPhoneNumber,
			},
		}, nil))
	md.Expect(mock.NewExpectation(md.CreateProxyPhoneNumberReservation, &models.ProxyPhoneNumberReservation{
		ProxyPhoneNumber:       proxyPhoneNumber,
		DestinationPhoneNumber: destinationPhoneNumber,
		OriginatingPhoneNumber: originatingPhoneNumber,
		DestinationEntityID:    destinationEntityID,
		OwnerEntityID:          sourceEntityID,
		OrganizationID:         organizationID,
		Expires:                mclock.Now().Add(phoneReservationDuration),
	}))

	manager := NewManager(md, mclock)

	pn, err := manager.ReserveNumber(originatingPhoneNumber, destinationPhoneNumber, destinationEntityID, sourceEntityID, organizationID)
	test.OK(t, err)
	test.Equals(t, proxyPhoneNumber, pn)
}

func TestReserveNumber_ExistingReservation(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	originatingPhoneNumber := phone.Number("+12068773590")
	destinationPhoneNumber := phone.Number("+17348465522")
	proxyPhoneNumber := phone.Number("+11234567890")
	destinationEntityID := "d1"
	sourceEntityID := "s1"
	organizationID := "o1"
	mclock := clock.NewManaged(time.Now())

	md.Expect(mock.NewExpectation(md.ActiveProxyPhoneNumberReservation, originatingPhoneNumber, phone.Ptr(destinationPhoneNumber), phone.Ptr(phone.Number(""))).
		WithReturns(&models.ProxyPhoneNumberReservation{
			ProxyPhoneNumber:       proxyPhoneNumber,
			DestinationPhoneNumber: destinationPhoneNumber,
			OriginatingPhoneNumber: originatingPhoneNumber,
			DestinationEntityID:    destinationEntityID,
			OwnerEntityID:          sourceEntityID,
			OrganizationID:         organizationID,
			Expires:                mclock.Now().Add(phoneReservationDuration),
		}, nil))

	md.Expect(mock.NewExpectation(md.UpdateActiveProxyPhoneNumberReservation, originatingPhoneNumber, phone.Ptr(destinationPhoneNumber), phone.Ptr(phone.Number("")), &dal.ProxyPhoneNumberReservationUpdate{
		Expires: ptr.Time(mclock.Now().Add(phoneReservationDuration)),
	}).WithReturns(int64(1), nil))

	manager := NewManager(md, mclock)

	pn, err := manager.ReserveNumber(originatingPhoneNumber, destinationPhoneNumber, destinationEntityID, sourceEntityID, organizationID)
	test.OK(t, err)
	test.Equals(t, proxyPhoneNumber, pn)
}

func TestReserveNumber_LastReserved(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	originatingPhoneNumber := phone.Number("+12068773590")
	destinationPhoneNumber := phone.Number("+17348465522")
	proxyPhoneNumber1 := phone.Number("+11234567890")
	proxyPhoneNumber2 := phone.Number("+12222222222")
	proxyPhoneNumber3 := phone.Number("+13333333333")
	destinationEntityID := "d1"
	sourceEntityID := "s1"
	organizationID := "o1"
	mclock := clock.NewManaged(time.Now())

	md.Expect(mock.NewExpectation(md.ActiveProxyPhoneNumberReservation, originatingPhoneNumber, phone.Ptr(destinationPhoneNumber), phone.Ptr(phone.Number(""))).
		WithReturns(nil, dal.ErrProxyPhoneNumberReservationNotFound))
	md.Expect(mock.NewExpectation(md.AvailableProxyPhoneNumbers, originatingPhoneNumber).WithReturns(
		[]*models.ProxyPhoneNumber{
			{
				PhoneNumber: proxyPhoneNumber2,
				Expires:     ptr.Time(time.Date(2016, 1, 1, 1, 1, 1, 1, time.UTC)),
			},
			{
				PhoneNumber: proxyPhoneNumber1,
				Expires:     ptr.Time(time.Date(2014, 1, 1, 1, 1, 1, 1, time.UTC)),
			},
			{
				PhoneNumber: proxyPhoneNumber3,
				Expires:     ptr.Time(time.Date(2015, 1, 1, 1, 1, 1, 1, time.UTC)),
			},
		}, nil))
	md.Expect(mock.NewExpectation(md.CreateProxyPhoneNumberReservation, &models.ProxyPhoneNumberReservation{
		ProxyPhoneNumber:       proxyPhoneNumber1,
		DestinationPhoneNumber: destinationPhoneNumber,
		OriginatingPhoneNumber: originatingPhoneNumber,
		DestinationEntityID:    destinationEntityID,
		OwnerEntityID:          sourceEntityID,
		OrganizationID:         organizationID,
		Expires:                mclock.Now().Add(phoneReservationDuration),
	}))

	manager := NewManager(md, mclock)

	pn, err := manager.ReserveNumber(originatingPhoneNumber, destinationPhoneNumber, destinationEntityID, sourceEntityID, organizationID)
	test.OK(t, err)
	test.Equals(t, proxyPhoneNumber1, pn)
}

func TestReserveNumber_BeyondGracePeriod(t *testing.T) {
	md := dalmock.New(t)
	defer md.Finish()

	originatingPhoneNumber := phone.Number("+12068773590")
	destinationPhoneNumber := phone.Number("+17348465522")
	proxyPhoneNumber1 := phone.Number("+11234567890")
	destinationEntityID := "d1"
	sourceEntityID := "s1"
	organizationID := "o1"
	mclock := clock.NewManaged(time.Now())

	md.Expect(mock.NewExpectation(md.ActiveProxyPhoneNumberReservation, originatingPhoneNumber, phone.Ptr(destinationPhoneNumber), phone.Ptr(phone.Number(""))).
		WithReturns(nil, dal.ErrProxyPhoneNumberReservationNotFound))
	md.Expect(mock.NewExpectation(md.AvailableProxyPhoneNumbers, originatingPhoneNumber).WithReturns(
		[]*models.ProxyPhoneNumber{
			{
				PhoneNumber: proxyPhoneNumber1,
				Expires:     ptr.Time(mclock.Now().Add(-time.Hour)),
			},
		}, nil))
	md.Expect(mock.NewExpectation(md.CreateProxyPhoneNumberReservation, &models.ProxyPhoneNumberReservation{
		ProxyPhoneNumber:       proxyPhoneNumber1,
		DestinationPhoneNumber: destinationPhoneNumber,
		OriginatingPhoneNumber: originatingPhoneNumber,
		DestinationEntityID:    destinationEntityID,
		OwnerEntityID:          sourceEntityID,
		OrganizationID:         organizationID,
		Expires:                mclock.Now().Add(phoneReservationDuration),
	}))

	manager := NewManager(md, mclock)

	pn, err := manager.ReserveNumber(originatingPhoneNumber, destinationPhoneNumber, destinationEntityID, sourceEntityID, organizationID)
	test.OK(t, err)
	test.Equals(t, proxyPhoneNumber1, pn)
}
