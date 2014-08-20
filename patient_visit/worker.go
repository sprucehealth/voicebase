package patient_visit

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"strconv"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/stripe"
	"github.com/sprucehealth/backend/third_party/github.com/samuel/go-metrics/metrics"
)

const (
	batchSize         = 1
	visibilityTimeout = 60 * 5
	waitTimeSeconds   = 60 * 10
	receiptNumberMax  = 5
)

type worker struct {
	dataAPI         api.DataAPI
	stripeCli       apiservice.StripeClient
	queue           *common.SQSQueue
	metricsRegistry metrics.Registry
}

func StartWorker(dataAPI api.DataAPI, stripeCli apiservice.StripeClient, queue *common.SQSQueue, metricsRegistry metrics.Registry) {
	(&worker{
		dataAPI:   dataAPI,
		stripeCli: stripeCli,
		queue:     queue,
	}).start()
}

func (w *worker) start() {
	go func() {
		for {
			if err := w.consumeMessage(); err != nil {
				golog.Errorf(err.Error())
			}
		}
	}()
}

func (w *worker) consumeMessage() error {
	msgs, err := w.queue.QueueService.ReceiveMessage(w.queue.QueueUrl, nil, batchSize, visibilityTimeout, waitTimeSeconds)
	if err != nil {
		return err
	}

	for _, m := range msgs {
		v := &visitMessage{}
		if err := json.Unmarshal([]byte(m.Body), v); err != nil {
			return err
		}

		if err := w.processMessage(v); err != nil {
			golog.Errorf(err.Error())
		} else {
			if err := w.queue.QueueService.DeleteMessage(w.queue.QueueUrl, m.ReceiptHandle); err != nil {
				golog.Errorf(err.Error())
			}
		}
	}

	return nil
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

	nextStatus := common.PREmailPending
	patientReceiptUpdate := &api.PatientReceiptUpdate{Status: &nextStatus}

	if costBreakdown.TotalCost.Amount > 0 {
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

		// get the default card of the patient from the visit that we are going to charge
		defaultCard, err := w.dataAPI.GetDefaultCardForPatient(m.PatientID)
		if err != nil {
			return err
		}

		// only create a charge if one doesn't already exist for the customer
		if charge == nil {
			// if no charge exists, run the charge on stripe
			// TODO Fix conversion problem (probably have all amounts in cents)
			// TODO Fix currency problem so that conversion is not required
			charge, err = w.stripeCli.CreateChargeForCustomer(&stripe.CreateChargeRequest{
				Amount:     int(costBreakdown.TotalCost.Amount * 100),
				Currency:   stripe.USD,
				CustomerID: patient.PaymentCustomerId,
				CardToken:  defaultCard.ThirdPartyId,
				Metadata: map[string]string{
					"receipt_ref_num": pReceipt.ReferenceNumber,
				},
			})
			if err != nil {
				// TODO: if the charge fails, emit a metric that we alarm on because this means that the visit cannot be routed.
				return err
			}
		}

		defaultCardId := defaultCard.Id.Int64()
		patientReceiptUpdate.CreditCardID = &defaultCardId
		patientReceiptUpdate.StripeChargeID = &charge.ID
	}

	// update receipt to indicate that any payment was successfully charged to the customer
	if err := w.dataAPI.UpdatePatientReceipt(pReceipt.ID, patientReceiptUpdate); err != nil {
		return err
	}

	// send the email for the patient receipt
	if err := w.sendReceipt(pReceipt); err != nil {
		return err
	}

	// update the receipt status to indicate that email was sent
	status := common.PREmailSent
	if err := w.dataAPI.UpdatePatientReceipt(pReceipt.ID, &api.PatientReceiptUpdate{Status: &status}); err != nil {
		return err
	}

	// update the status of the case to indicate that we successfully charged for it
	opStatus := common.PCOpStatusCharged
	if err := w.dataAPI.UpdatePatientCase(m.PatientCaseID, &api.PatientCaseUpdate{OperationalStatus: &opStatus}); err != nil {
		return err
	}

	return w.publishVisitChargedEvent(m)
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
	bigRefNum, err := rand.Int(rand.Reader, big.NewInt(receiptNumberMax))
	if err != nil {
		return nil, err
	}
	refNum := bigRefNum.String()
	for len(refNum) < receiptNumberMax {
		refNum = "0" + refNum
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

func (w *worker) sendReceipt(pReceipt *common.PatientReceipt) error {

	// nothing to do if the email has already been sent
	if pReceipt.Status == common.PREmailSent {
		return nil
	}

	// TODO: send email
	return nil
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
