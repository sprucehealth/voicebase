package medrecord

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/media"
)

const emailType = "medical-record-ready"

type emailContext struct {
	DownloadURL string
}

func init() {
	email.MustRegisterType(&email.Type{
		Key:  emailType,
		Name: "Medical Record Ready",
		TestContext: &emailContext{
			DownloadURL: "https://www.sprucehealth.com/patient/medical-record",
		},
	})
}

const (
	batchSize         = 1
	visibilityTimeout = 60 * 5
	waitTimeSeconds   = 20
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
			}
		}
	}()
}

func (w *Worker) Do() error {
	msgs, err := w.queue.QueueService.ReceiveMessage(w.queue.QueueURL, nil, batchSize, visibilityTimeout, waitTimeSeconds)
	if err != nil {
		return err
	}

	for _, m := range msgs {
		msg := &queueMessage{}
		if err := json.Unmarshal([]byte(m.Body), msg); err != nil {
			golog.Errorf(err.Error())
			continue
		}
		if err := w.processMessage(msg); err != nil {
			w.statFailed.Inc(1)
			golog.Errorf(err.Error())
		} else {
			if err := w.queue.QueueService.DeleteMessage(w.queue.QueueURL, m.ReceiptHandle); err != nil {
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
		golog.Errorf("Patient %d does not exist for medical record %d", mr.PatientID, mr.ID)
		return nil
	} else if err != nil {
		return err
	}

	recordFile, err := w.renderer.Render(patient)
	if err != nil {
		return fmt.Errorf("Failed to render medical record: %s", err)
	}

	headers := http.Header{"Content-Type": []string{"text/html"}}
	// TODO: caching headers
	url, err := w.store.Put(fmt.Sprintf("%d.html", mr.ID), recordFile, headers)
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

	if err := w.emailService.SendTemplateType(&mail.Address{Address: patient.Email}, emailType, &emailContext{
		DownloadURL: downloadURL,
	}); err != nil {
		golog.Errorf("Failed to send medical record email for record %d to patient %d: %s",
			mr.ID, patient.PatientID.Int64(), err.Error())
	}

	return nil
}
