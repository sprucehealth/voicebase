package cost

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/analytics"
	"github.com/sprucehealth/backend/cmd/svc/restapi/api"
	"github.com/sprucehealth/backend/cmd/svc/restapi/apiservice"
	"github.com/sprucehealth/backend/cmd/svc/restapi/common"
	"github.com/sprucehealth/backend/cmd/svc/restapi/email"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/stripe"
)

const (
	receiptNumberMax  = 99999
	receiptNumDigits  = 5
	defaultTimePeriod = 60
)

var (
	batchSize         int64 = 1
	visibilityTimeout int64 = 60 * 5
	waitTimeSeconds   int64 = 20
)

// Worker represents the data needed to perform the async operations related to cost
type Worker struct {
	dataAPI              api.DataAPI
	launchPromoStartDate *time.Time
	analyticsLogger      analytics.Logger
	dispatcher           *dispatch.Dispatcher
	stripeCli            apiservice.StripeClient
	emailService         email.Service
	supportEmail         string
	queue                *common.SQSQueue
	chargeSuccess        *metrics.Counter
	chargeFailure        *metrics.Counter
	receiptSendSuccess   *metrics.Counter
	receiptSendFailure   *metrics.Counter
	timePeriodInSeconds  int
	cfgStore             cfg.Store
	stopChan             chan bool
}

// NewWorker returns an initialized instance of Worker
func NewWorker(dataAPI api.DataAPI, analyticsLogger analytics.Logger, dispatcher *dispatch.Dispatcher,
	stripeCli apiservice.StripeClient, emailService email.Service,
	queue *common.SQSQueue, metricsRegistry metrics.Registry,
	timePeriodInSeconds int, supportEmail string, cfgStore cfg.Store) *Worker {
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

	return &Worker{
		dataAPI:             dataAPI,
		analyticsLogger:     analyticsLogger,
		dispatcher:          dispatcher,
		stripeCli:           stripeCli,
		emailService:        emailService,
		supportEmail:        supportEmail,
		queue:               queue,
		chargeSuccess:       chargeSuccess,
		chargeFailure:       chargeFailure,
		receiptSendSuccess:  receiptSendSuccess,
		receiptSendFailure:  receiptSendFailure,
		timePeriodInSeconds: timePeriodInSeconds,
		cfgStore:            cfgStore,
		stopChan:            make(chan bool),
	}
}

// Start begins the async operations performed by the worker in a goroutine
func (w *Worker) Start() {
	go func() {
		for {
			select {
			case <-w.stopChan:
				return
			default:
			}

			if err := w.Do(); err != nil {
				golog.Errorf(err.Error())
			}
		}
	}()
}

// Stop stops the routine that the worker is currently running
func (w *Worker) Stop() {
	close(w.stopChan)
}

// Do is the actual operation performed by the worker
func (w *Worker) Do() error {

	// keep going until there are no messages left to consume
	for {
		msgConsumed, err := w.consumeMessage()
		if err != nil {
			golog.Errorf(err.Error())
		}

		if !msgConsumed {
			break
		}
	}

	return nil
}

func (w *Worker) consumeMessage() (bool, error) {
	res, err := w.queue.QueueService.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            &w.queue.QueueURL,
		MaxNumberOfMessages: &batchSize,
		VisibilityTimeout:   &visibilityTimeout,
		WaitTimeSeconds:     &waitTimeSeconds,
	})
	if err != nil {
		return false, err
	}

	allMsgsConsumed := len(res.Messages) > 0

	for _, m := range res.Messages {
		v := &VisitMessage{}
		if err := json.Unmarshal([]byte(*m.Body), v); err != nil {
			return false, err
		}

		if err := w.processMessage(v); err != nil {
			golog.Errorf(err.Error())
			allMsgsConsumed = false
		} else {
			_, err := w.queue.QueueService.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      &w.queue.QueueURL,
				ReceiptHandle: m.ReceiptHandle,
			})
			if err != nil {
				golog.Errorf(err.Error())
				allMsgsConsumed = false
			}
		}
	}

	return allMsgsConsumed, nil
}

func (w *Worker) processMessage(m *VisitMessage) error {
	patient, err := w.dataAPI.GetPatientFromPatientVisitID(m.PatientVisitID)
	if err != nil {
		return errors.Trace(err)
	} else if patient.Training {
		return nil
	}

	patientCase, err := w.dataAPI.GetPatientCaseFromID(m.PatientCaseID)
	if err != nil {
		return errors.Trace(err)
	} else if patient.Training {
		return nil
	}

	// get the cost of the visit
	costBreakdown, err := totalCostForItems([]string{m.SKUType}, m.AccountID, true, w.dataAPI, w.analyticsLogger, w.cfgStore)
	if err != nil {
		return errors.Trace(err)
	}

	pReceipt, err := w.retrieveOrCreatePatientReceipt(m.PatientID,
		m.PatientVisitID,
		costBreakdown.ItemCosts[0].ID,
		m.SKUType,
		costBreakdown)
	if err != nil {
		return errors.Trace(err)
	}

	currentStatus := pReceipt.Status
	nextStatus := common.PRCharged
	patientReceiptUpdate := &api.PatientReceiptUpdate{Status: &nextStatus}

	if costBreakdown.TotalCost.Amount > 0 && currentStatus == common.PRChargePending {
		// check if the charge already exists for the customer
		var charge *stripe.Charge
		charges, err := w.stripeCli.ListAllCustomerCharges(patient.PaymentCustomerID)
		if err != nil {
			return errors.Trace(err)
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
			card, err = w.dataAPI.GetCardFromThirdPartyID(charge.Card.ID)
			if err != nil && !api.IsErrNotFound(err) {
				return errors.Trace(err)
			}
		} else if m.CardID != 0 {
			card, err = w.dataAPI.GetCardFromID(m.CardID)
			if err != nil {
				return errors.Trace(err)
			}
		} else {
			// get the default card of the patient from the visit that we are going to charge
			card, err = w.dataAPI.GetDefaultCardForPatient(m.PatientID)
			if api.IsErrNotFound(err) {
				return errors.Trace(errors.New("No default card for patient"))
			} else if err != nil {
				return errors.Trace(err)
			}
		}

		// only create a charge if one doesn't already exist for the customer
		if charge == nil {

			// lets get the state that the patient is located in
			_, state, err := w.dataAPI.PatientLocation(m.PatientID)
			if err != nil {
				return errors.Trace(err)
			}

			var requestedDoctorID string
			var doctorLongDisplayName string
			if patientCase.RequestedDoctorID != nil {
				doctor, err := w.dataAPI.GetDoctorFromID(*patientCase.RequestedDoctorID)
				if err != nil {
					return errors.Trace(err)
				}
				doctorLongDisplayName = doctor.LongDisplayName
				requestedDoctorID = strconv.FormatInt(*patientCase.RequestedDoctorID, 10)
			}
			charge, err = w.stripeCli.CreateChargeForCustomer(&stripe.CreateChargeRequest{
				Amount:       costBreakdown.TotalCost.Amount,
				CurrencyCode: costBreakdown.TotalCost.Currency,
				CustomerID:   patient.PaymentCustomerID,
				Description:  fmt.Sprintf("Spruce Visit for %s %s", patient.FirstName, patient.LastName),
				CardToken:    card.ThirdPartyID,
				ReceiptEmail: patient.Email,
				Metadata: map[string]string{
					"receipt_ref_num":     pReceipt.ReferenceNumber,
					"visit_id":            strconv.FormatInt(m.PatientVisitID, 10),
					"state":               state,
					"sku":                 m.SKUType,
					"practice_extension":  strconv.FormatBool(patientCase.PracticeExtension),
					"requested_doctor_id": requestedDoctorID,
					"doctor_name":         doctorLongDisplayName,
				},
			})
			if err != nil {
				w.chargeFailure.Inc(1)
				return errors.Trace(err)
			}
			w.chargeSuccess.Inc(1)
		}

		patientReceiptUpdate.StripeChargeID = &charge.ID
	}

	if currentStatus == common.PRChargePending {
		// update receipt to indicate that any payment was successfully charged to the customer
		if err := w.dataAPI.UpdatePatientReceipt(pReceipt.ID, patientReceiptUpdate); err != nil {
			return errors.Trace(err)
		}
	}

	// update the patient visit to indicate that it was successfully charged
	pvStatus := common.PVStatusCharged
	if _, err := w.dataAPI.UpdatePatientVisit(m.PatientVisitID, &api.PatientVisitUpdate{Status: &pvStatus}); err != nil {
		return errors.Trace(err)
	}

	// first publish the charged event before sending the email so that we are not waiting too long
	// before routing the case (say, in the event that email service is down)
	w.publishVisitChargedEvent(m)

	return nil
}

func (w *Worker) retrieveOrCreatePatientReceipt(patientID common.PatientID, patientVisitID, itemCostID int64,
	skuType string, costBreakdown *common.CostBreakdown) (*common.PatientReceipt, error) {
	// check if a receipt exists in the databse
	var pReceipt *common.PatientReceipt
	var err error
	pReceipt, err = w.dataAPI.GetPatientReceipt(patientID, patientVisitID, skuType, false)
	if err == nil {
		return pReceipt, nil
	} else if !api.IsErrNotFound(err) {
		return nil, err
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
		SKUType:         skuType,
		ItemID:          patientVisitID,
		PatientID:       patientID,
		Status:          common.PRChargePending,
		CostBreakdown:   costBreakdown,
		ItemCostID:      itemCostID,
	}

	if err := w.dataAPI.CreatePatientReceipt(pReceipt); err != nil {
		return nil, err
	}

	return pReceipt, nil
}

func (w *Worker) publishVisitChargedEvent(m *VisitMessage) error {
	if err := w.dispatcher.Publish(&VisitChargedEvent{
		PatientID:     m.PatientID,
		AccountID:     m.AccountID,
		VisitID:       m.PatientVisitID,
		PatientCaseID: m.PatientCaseID,
		IsFollowup:    m.IsFollowup,
	}); err != nil {
		return err
	}
	return nil
}