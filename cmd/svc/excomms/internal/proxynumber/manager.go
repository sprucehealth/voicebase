package proxynumber

import (
	"fmt"
	"sort"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/excomms/internal/models"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
)

type manager struct {
	dal   dal.DAL
	clock clock.Clock
}

// Manager is an interface that can be conformed to to provide functionality to manage proxy phone numbers.
type Manager interface {

	// ReserveNumber returns a proxy number that is reserved for the originatingNumber to call the destinationNumber via the proxyPhoneNumber.
	ReserveNumber(originatingNumber, destinationNumber, provisionedPhoneNumber phone.Number, destinationEntityID, sourceEntityID, organizationID string) (phone.Number, error)

	ActiveReservation(originatingNumber, proxyNumber phone.Number) (*models.ProxyPhoneNumberReservation, error)

	// CallStarted indicates that a call on the proxy phone number has been started.
	CallStarted(originatingNumber, proxyNumber phone.Number) error

	// CallEnded indicates that a call on the proxy phone number has been ended.
	CallEnded(originatingNumber, proxyNumber phone.Number) error
}

func NewManager(dal dal.DAL, clock clock.Clock) Manager {
	return &manager{
		dal:   dal,
		clock: clock,
	}
}

// TODO: Move these values to config such that they are easily changeable.
var (
	// phoneReservationDuration represents the duration of time for which
	// a proxy phone number reservation to dial a particular number lasts.
	phoneReservationDuration = 15 * time.Minute

	// phoneReservationDurationGrace represents the grace period after the expiration
	// where the proxy phone number is not reserved for another phone call.
	phoneReservationDurationGrace = 5 * time.Minute

	// ongoingCallThreshold represents the maximum time for which an ongoing call can be considered
	// to be active. This enables us to release a phone number for use that may be marked as being in an
	// ongoing call when the call may have actually ended.
	ongoingCallThreshold = 5 * time.Hour
)

func (m *manager) ReserveNumber(originatingPhoneNumber, destinationPhoneNumber, provisionedPhoneNumber phone.Number, destinationEntityID, sourceEntityID, organizationID string) (phone.Number, error) {
	var proxyPhoneNumber phone.Number
	if err := m.dal.Transact(func(dl dal.DAL) error {

		// check if an active reservation already exists for the source/destination pair and the same source/destination entity pair, and if
		// so, extend the reservation and return the same number rather than reserving a new number
		ppnr, err := dl.ActiveProxyPhoneNumberReservation(phone.Ptr(originatingPhoneNumber), phone.Ptr(destinationPhoneNumber), nil)
		if err != nil && errors.Cause(err) != dal.ErrProxyPhoneNumberReservationNotFound {
			return errors.Trace(err)
		}
		if ppnr != nil && ppnr.DestinationEntityID == destinationEntityID && ppnr.OwnerEntityID == sourceEntityID {
			expiration := m.clock.Now().Add(phoneReservationDuration)
			// extend the existing reservation rather than creating a new one and return
			if rowsAffected, err := dl.UpdateActiveProxyPhoneNumberReservation(originatingPhoneNumber, phone.Ptr(destinationPhoneNumber), nil, &dal.ProxyPhoneNumberReservationUpdate{
				Expires: &expiration,
			}); err != nil {
				return errors.Trace(err)
			} else if rowsAffected > 1 {
				return errors.Errorf("Expected 1 row to be updated, instead %d rows were updated for proxyPhoneNumber %s", rowsAffected, ppnr.ProxyPhoneNumber)
			}

			proxyPhoneNumber = ppnr.ProxyPhoneNumber
			return nil
		}

		// if no active reservation exists, then lets go ahead and reserve a new number
		ppns, err := dl.AvailableProxyPhoneNumbers(originatingPhoneNumber)
		if err != nil {
			return errors.Trace(err)
		}

		// sort by expires so that phone numbers that were reserved
		// furthest back are reserved first
		sort.Sort(models.ByExpiresProxyPhoneNumbers(ppns))

		for _, ppn := range ppns {
			// select number if it has never been used before or it is beyond the grace reservation window.
			if ppn.Expires == nil || ppn.Expires.Before(m.clock.Now()) {
				proxyPhoneNumber = ppn.PhoneNumber
				break
			}
		}

		if proxyPhoneNumber.IsEmpty() {
			err := fmt.Errorf("Unable to find a free phone number to reserve for %s -> %s by entity %s", originatingPhoneNumber, destinationPhoneNumber, sourceEntityID)
			golog.Errorf(err.Error())
			return errors.Trace(err)
		}

		expiration := m.clock.Now().Add(phoneReservationDuration)

		return errors.Trace(dl.CreateProxyPhoneNumberReservation(&models.ProxyPhoneNumberReservation{
			ProxyPhoneNumber:       proxyPhoneNumber,
			DestinationPhoneNumber: destinationPhoneNumber,
			OriginatingPhoneNumber: originatingPhoneNumber,
			ProvisionedPhoneNumber: provisionedPhoneNumber,
			DestinationEntityID:    destinationEntityID,
			OwnerEntityID:          sourceEntityID,
			OrganizationID:         organizationID,
			Expires:                expiration,
		}))
	}); err != nil {
		return phone.Number(""), err
	}

	return proxyPhoneNumber, nil
}

func (m *manager) ActiveReservation(originatingNumber, proxyNumber phone.Number) (*models.ProxyPhoneNumberReservation, error) {
	// look for an active reservation on the proxy phone number
	ppnr, err := m.dal.ActiveProxyPhoneNumberReservation(phone.Ptr(originatingNumber), nil, phone.Ptr(proxyNumber))
	if errors.Cause(err) == dal.ErrProxyPhoneNumberReservationNotFound {
		return nil, errors.Errorf("No active reservation found for proxy number: %s, originating number:%s", proxyNumber, originatingNumber)
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	return ppnr, nil
}

func (m *manager) CallStarted(originatingNumber, proxyNumber phone.Number) error {
	if rowsUpdated, err := m.dal.UpdateActiveProxyPhoneNumberReservation(originatingNumber, nil, phone.Ptr(proxyNumber), &dal.ProxyPhoneNumberReservationUpdate{
		Expires: ptr.Time(m.clock.Now().Add(ongoingCallThreshold)),
	}); err != nil {
		return err
	} else if rowsUpdated != 1 {
		return errors.Errorf("Expected one row to be updated for active reservation (originatingNumber:%s, proxyNumber:%s) but %d rows were updated", originatingNumber, proxyNumber, rowsUpdated)
	}

	return nil
}

func (m *manager) CallEnded(originatingNumber, proxyNumber phone.Number) error {
	if rowsUpdated, err := m.dal.UpdateActiveProxyPhoneNumberReservation(originatingNumber, nil, phone.Ptr(proxyNumber), &dal.ProxyPhoneNumberReservationUpdate{
		Expires: ptr.Time(m.clock.Now().Add(phoneReservationDurationGrace)),
	}); err != nil {
		return errors.Trace(err)
	} else if rowsUpdated != 1 {
		return errors.Errorf("Expected 1 reservation to be updated for (originatingNumber: %s, proxyNumber: %s) but got %d row updates", originatingNumber, proxyNumber, rowsUpdated)
	}
	return nil
}
