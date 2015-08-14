package medrecord

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sqs"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/mandrill"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/media"
)

const emailType = "medical-record-ready"

var (
	batchSize         int64 = 1
	visibilityTimeout int64 = 60 * 5
	waitTimeSeconds   int64 = 20
)

type Worker struct {
	dataAPI       api.DataAPI
	renderer      *Renderer
	queue         *common.SQSQueue
	emailService  email.Service
	supportEmail  string
	store         storage.Store
	webDomain     string
	stopChan      chan bool
	statSucceeded *metrics.Counter
	statFailed    *metrics.Counter
}

func NewWorker(
	dataAPI api.DataAPI,
	diagnosisSvc diagnosis.API,
	queue *common.SQSQueue,
	emailService email.Service,
	supportEmail, apiDomain, webDomain string,
	signer *sig.Signer,
	store storage.Store,
	mediaStore *media.Store,
	expirationDuration time.Duration,
	metricsRegistry metrics.Registry,
) *Worker {
	w := &Worker{
		renderer: &Renderer{
			DataAPI:            dataAPI,
			DiagnosisSvc:       diagnosisSvc,
			MediaStore:         mediaStore,
			APIDomain:          apiDomain,
			WebDomain:          webDomain,
			Signer:             signer,
			ExpirationDuration: expirationDuration,
		},
		dataAPI:       dataAPI,
		queue:         queue,
		emailService:  emailService,
		supportEmail:  supportEmail,
		store:         store,
		webDomain:     webDomain,
		stopChan:      make(chan bool),
		statSucceeded: metrics.NewCounter(),
		statFailed:    metrics.NewCounter(),
	}
	if metricsRegistry != nil {
		metricsRegistry.Add("succeeded", w.statSucceeded)
		metricsRegistry.Add("failed", w.statFailed)
	}
	return w
}

func (w *Worker) Stop() {
	close(w.stopChan)
}

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
				time.Sleep(time.Minute)
			}
		}
	}()
}

func (w *Worker) Do() error {
	res, err := w.queue.QueueService.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueURL:            &w.queue.QueueURL,
		MaxNumberOfMessages: &batchSize,
		VisibilityTimeout:   &visibilityTimeout,
		WaitTimeSeconds:     &waitTimeSeconds,
	})
	if err != nil {
		return err
	}

	for _, m := range res.Messages {
		msg := &queueMessage{}
		if err := json.Unmarshal([]byte(*m.Body), msg); err != nil {
			golog.Errorf(err.Error())
			continue
		}
		if err := w.processMessage(msg); err != nil {
			w.statFailed.Inc(1)
			golog.Errorf(err.Error())
		} else {
			_, err := w.queue.QueueService.DeleteMessage(&sqs.DeleteMessageInput{
				QueueURL:      &w.queue.QueueURL,
				ReceiptHandle: m.ReceiptHandle,
			})
			if err != nil {
				golog.Errorf(err.Error())
				w.statFailed.Inc(1)
			} else {
				w.statSucceeded.Inc(1)
			}
		}
	}

	return nil
}

func (w *Worker) processMessage(msg *queueMessage) error {
	mr, err := w.dataAPI.MedicalRecord(msg.MedicalRecordID)
	if api.IsErrNotFound(err) {
		golog.Errorf("Medical record not found for ID %d", msg.MedicalRecordID)
		// Don't return an error so the message is removed from the queue since this
		// is unrecoverable.
		return nil
	} else if err != nil {
		return err
	}

	if mr.Status != common.MRPending {
		golog.Warningf("Medical record %d not pending. Status = %+v", mr.ID, mr.Status)
		return nil
	}

	patient, err := w.dataAPI.GetPatientFromID(mr.PatientID)
	if api.IsErrNotFound(err) {
		golog.Errorf("Patient %s does not exist for medical record %d", mr.PatientID, mr.ID)
		return nil
	} else if err != nil {
		return err
	}

	recordFile, err := w.renderer.Render(patient, 0)
	if err != nil {
		return fmt.Errorf("Failed to render medical record: %s", err)
	}

	// TODO: caching headers
	url, err := w.store.Put(fmt.Sprintf("%d.html", mr.ID), recordFile, "text/html", nil)
	if err != nil {
		return err
	}

	now := time.Now()
	status := common.MRSuccess

	if err := w.dataAPI.UpdateMedicalRecord(mr.ID, &api.MedicalRecordUpdate{
		Status:     &status,
		StorageURL: &url,
		Completed:  &now,
	}); err != nil {
		if err := w.store.Delete(url); err != nil {
			golog.Errorf("Failed to delete failed medical record %d %s: %s", mr.ID, url, err.Error())
		}
		return err
	}

	downloadURL := fmt.Sprintf("https://%s/patient/medical-record", w.webDomain)

	if _, err := w.emailService.Send([]int64{patient.AccountID.Int64()}, emailType, nil, &mandrill.Message{
		GlobalMergeVars: []mandrill.Var{
			{
				Name:    "DownloadURL",
				Content: downloadURL,
			},
		},
	}, 0); err != nil {
		golog.Errorf("Failed to send medical record email for record %d to patient %d: %s",
			mr.ID, patient.ID.Int64(), err.Error())
	}

	return nil
}
