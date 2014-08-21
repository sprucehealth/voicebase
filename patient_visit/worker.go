package patient_visit

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

const (
	batchSize               = 1
	visibilityTimeout       = 60 * 5
	waitTimeSeconds         = 20
	timeBetweenEmailRetries = 10
	receiptNumberMax        = 99999
	receiptNumDigits        = 5
	defaultTimePeriod       = 60
)

type worker struct {
	dataAPI             api.DataAPI
	stripeCli           apiservice.StripeClient
	emailService        email.Service
	supportEmail        string
	queue               *common.SQSQueue
	chargeSuccess       metrics.Counter
	chargeFailure       metrics.Counter
	receiptSendSuccess  metrics.Counter
	receiptSendFailure  metrics.Counter
	timePeriodInSeconds int
}

func StartWorker(dataAPI api.DataAPI, stripeCli apiservice.StripeClient, emailService email.Service, queue *common.SQSQueue, metricsRegistry metrics.Registry, timePeriodInSeconds int, supportEmail string) {
	if timePeriodInSeconds == 0 {
		timePeriodInSeconds = defaultTimePeriod
	}

	chargeSuccess := metrics.NewCounter()
	chargeFailure := metrics.NewCounter()
	receiptSendSuccess := metrics.NewCounter()
	receiptSendFailure := metrics.NewCounter()

	metricsRegistry.Add("case_charge/success", chargeSuccess)
	metricsRegistry.Add("case_charge/failure", chargeFailure)
	metricsRegistry.Add("receipt_send/success", receiptSendSuccess)
	metricsRegistry.Add("receipt_send/failure", receiptSendFailure)

	(&worker{
		dataAPI:             dataAPI,
		stripeCli:           stripeCli,
		emailService:        emailService,
		supportEmail:        supportEmail,
		queue:               queue,
		chargeSuccess:       chargeSuccess,
		chargeFailure:       chargeFailure,
		receiptSendSuccess:  receiptSendSuccess,
		receiptSendFailure:  receiptSendFailure,
		timePeriodInSeconds: timePeriodInSeconds,
	}).start()
}

func (w *worker) start() {
	go func() {
		for {
			if msgConsumed, err := w.consumeMessage(); err != nil {
				golog.Errorf(err.Error())
			} else if !msgConsumed {
				time.Sleep(time.Duration(w.timePeriodInSeconds) * time.Second)
			}
		}
	}()
}

func (w *worker) consumeMessage() (bool, error) {
	msgs, err := w.queue.QueueService.ReceiveMessage(w.queue.QueueUrl, nil, batchSize, visibilityTimeout, waitTimeSeconds)
	if err != nil {
		return false, err
	}

	allMsgsConsumed := len(msgs) > 0

	for _, m := range msgs {
		v := &visitMessage{}
		if err := json.Unmarshal([]byte(m.Body), v); err != nil {
			return false, err
		}

		if err := w.processMessage(v); err != nil {
			golog.Errorf(err.Error())
			allMsgsConsumed = false
		} else {
			if err := w.queue.QueueService.DeleteMessage(w.queue.QueueUrl, m.ReceiptHandle); err != nil {
				golog.Errorf(err.Error())
				allMsgsConsumed = false
			}
		}
	}

	return allMsgsConsumed, nil
}

func (w *worker) processMessage(m *visitMessage) error {
	patient, err := w.dataAPI.GetPatientFromPatientVisitId(m.PatientVisitID)
	if err != nil {
		return err
	}

	// get the cost of the visit
	itemCost, err := w.dataAPI.GetItemCost(m.ItemCostID)
	if err != nil {
		return err
	}

	costBreakdown := &common.CostBreakdown{LineItems: itemCost.LineItems}
	costBreakdown.CalculateTotal()

	pReceipt, err := w.retrieveOrCreatePatientReceipt(m.PatientID, m.PatientVisitID, m.ItemType, costBreakdown)
	if err != nil {
		return err
	}

	currentStatus := pReceipt.Status
	nextStatus := common.PREmailPending
	patientReceiptUpdate := &api.PatientReceiptUpdate{Status: &nextStatus}

	if costBreakdown.TotalCost.Amount > 0 && currentStatus == common.PRChargePending {
		// check if the charge already exists for the customer
		var charge *stripe.Charge
		charges, err := w.stripeCli.ListAllCustomerCharges(patient.PaymentCustomerId)
		if err != nil {
			return err
		}
		for _, cItem := range charges {
			if refNum, ok := cItem.Metadata["receipt_ref_num"]; ok && refNum == pReceipt.ReferenceNumber {
				charge = cItem
				break
			}
		}

		// if a charge exists, get the card used for the charge, else get the default card for the customer
		var card *common.Card
		if charge != nil {
			card, err = w.dataAPI.GetCardFromThirdPartyId(charge.Card.ID)
			if err != nil && err != api.NoRowsError {
				return err
			}
		} else {
			// get the default card of the patient from the visit that we are going to charge
			card, err = w.dataAPI.GetDefaultCardForPatient(m.PatientID)
			if err != nil {
				return err
			}
		}

		// only create a charge if one doesn't already exist for the customer
		if charge == nil {
			charge, err = w.stripeCli.CreateChargeForCustomer(&stripe.CreateChargeRequest{
				Amount:       int(costBreakdown.TotalCost.AmountInSmallestUnit()),
				CurrencyCode: costBreakdown.TotalCost.Currency,
				CustomerID:   patient.PaymentCustomerId,
				CardToken:    card.ThirdPartyId,
				Metadata: map[string]string{
					"receipt_ref_num": pReceipt.ReferenceNumber,
				},
			})
			if err != nil {
				w.chargeFailure.Inc(1)
				return err
			}
			w.chargeSuccess.Inc(1)
			defaultCardId := card.Id.Int64()
			patientReceiptUpdate.CreditCardID = &defaultCardId
		}

		patientReceiptUpdate.StripeChargeID = &charge.ID
	}

	if currentStatus == common.PRChargePending {
		// update receipt to indicate that any payment was successfully charged to the customer
		if err := w.dataAPI.UpdatePatientReceipt(pReceipt.ID, patientReceiptUpdate); err != nil {
			return err
		}
		currentStatus = common.PREmailPending
	}

	// update the patient visit to indicate that it was successfully charged
	pvStatus := common.PVStatusCharged
	if err := w.dataAPI.UpdatePatientVisit(m.PatientVisitID, &api.PatientVisitUpdate{Status: &pvStatus}); err != nil {
		return err
	}

	// first publish the charged event before sending the email so that we are not waiting too long
	// before routing the case (say, in the event that email service is down)
	w.publishVisitChargedEvent(m)

	// attempt to send the email a few times, but if we consistently fail then give up and move on
	for i := 0; i < 3; i++ {
		// send the email for the patient receipt
		if currentStatus == common.PREmailPending {
			if err := w.sendReceipt(patient, pReceipt); err != nil {
				w.receiptSendFailure.Inc(1)
				golog.Errorf("Unable to send receipt over email: %s", err)
			} else {
				w.receiptSendSuccess.Inc(1)
				// update the receipt status to indicate that email was sent
				status := common.PREmailSent
				if err := w.dataAPI.UpdatePatientReceipt(pReceipt.ID, &api.PatientReceiptUpdate{Status: &status}); err != nil {
					return err
				}
				break
			}
		} else {
			break
		}
		time.Sleep(timeBetweenEmailRetries * time.Second)
	}

	return nil
}

func (w *worker) retrieveOrCreatePatientReceipt(patientID, patientVisitID int64,
	itemType string, costBreakdown *common.CostBreakdown) (*common.PatientReceipt, error) {
	// check if a receipt exists in the databse
	var pReceipt *common.PatientReceipt
	var err error
	pReceipt, err = w.dataAPI.GetPatientReceipt(patientID, patientVisitID, itemType, false)
	if err != api.NoRowsError && err != nil {
		return nil, err
	} else if err != api.NoRowsError {
		return pReceipt, nil
	}

	// generate a random receipt number
	refNum, err := common.GenerateRandomNumber(receiptNumberMax, receiptNumDigits)
	if err != nil {
		return nil, err
	}

	// append the itemID to ensure that the number is unique
	refNum += strconv.FormatInt(patientVisitID, 10)

	pReceipt = &common.PatientReceipt{
		ReferenceNumber: refNum,
		ItemType:        itemType,
		ItemID:          patientVisitID,
		PatientID:       patientID,
		Status:          common.PRChargePending,
		CostBreakdown:   costBreakdown,
	}

	if err := w.dataAPI.CreatePatientReceipt(pReceipt); err != nil {
		return nil, err
	}

	return pReceipt, nil
}

func (w *worker) sendReceipt(patient *common.Patient, pReceipt *common.PatientReceipt) error {
	// nothing to do if we don't have an email service running
	if w.emailService == nil {
		return nil
	}

	var orderDetails string
	for _, lItem := range pReceipt.CostBreakdown.LineItems {
		orderDetails += fmt.Sprintf(`- %s: $%.2f`, lItem.Description, lItem.Cost.Amount)
	}

	em := &email.Email{
		From:    w.supportEmail,
		To:      patient.Email,
		Subject: "Spruce Visit Receipt",
		BodyText: fmt.Sprintf(`Hello %s,

Here is a receipt of your recent Spruce Visit for your records. If you have any questions or concerns, please don't hesitate to email us at %s.

Receipt #: %s
Transaction Date: %s
Order Details:
%s
---
Total: $%.2f

Thank you,
The Spruce Team
-
Need help? Contact %s`, patient.FirstName, w.supportEmail, pReceipt.ReferenceNumber, pReceipt.CreationTimestamp.Format("January 2 2006"), orderDetails, pReceipt.CostBreakdown.TotalCost.Amount, w.supportEmail),
	}

	return w.emailService.SendEmail(em)
}

func (w *worker) publishVisitChargedEvent(m *visitMessage) error {
	if err := dispatch.Default.Publish(&VisitChargedEvent{
		PatientID:     m.PatientID,
		VisitID:       m.PatientVisitID,
		PatientCaseID: m.PatientCaseID,
	}); err != nil {
		return err
	}
	return nil
}
