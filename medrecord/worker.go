package medrecord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/mail"
	"sort"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/mapstructure"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/treatment_plan"
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
	dataAPI            api.DataAPI
	queue              *common.SQSQueue
	emailService       email.Service
	supportEmail       string
	store              storage.Store
	mediaStore         *media.Store
	apiDomain          string
	webDomain          string
	signer             *sig.Signer
	expirationDuration time.Duration
	stopChan           chan bool
}

func NewWorker(
	dataAPI api.DataAPI,
	queue *common.SQSQueue,
	emailService email.Service,
	supportEmail, apiDomain, webDomain string,
	signer *sig.Signer,
	store storage.Store,
	mediaStore *media.Store,
	expirationDuration time.Duration) *Worker {
	return &Worker{
		dataAPI:            dataAPI,
		queue:              queue,
		emailService:       emailService,
		supportEmail:       supportEmail,
		store:              store,
		mediaStore:         mediaStore,
		apiDomain:          apiDomain,
		webDomain:          webDomain,
		signer:             signer,
		expirationDuration: expirationDuration,
		stopChan:           make(chan bool),
	}
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
			golog.Errorf(err.Error())
		} else {
			if err := w.queue.QueueService.DeleteMessage(w.queue.QueueURL, m.ReceiptHandle); err != nil {
				golog.Errorf(err.Error())
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

	recordFile, err := w.generateHTML(patient)
	if err != nil {
		return err
	}

	headers := http.Header{}
	headers.Set("Content-Type", "text/html")
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

func (w *Worker) generateHTML(patient *common.Patient) ([]byte, error) {
	ctx := &templateContext{
		Patient: patient,
	}

	ag, err := w.dataAPI.PatientAgreements(patient.PatientID.Int64())
	if err != nil {
		return nil, err
	}
	ctx.Agreements = ag

	pcp, err := w.dataAPI.GetPatientPCP(patient.PatientID.Int64())
	if err != nil {
		return nil, err
	}
	ctx.PCP = pcp

	ec, err := w.dataAPI.GetPatientEmergencyContacts(patient.PatientID.Int64())
	if err != nil {
		return nil, err
	}
	ctx.EmergencyContacts = ec

	cases, err := w.dataAPI.GetCasesForPatient(patient.PatientID.Int64())
	if err != nil {
		return nil, err
	}

	for _, pcase := range cases {
		visits, err := w.dataAPI.GetVisitsForCase(pcase.ID.Int64(), nil)
		if err != nil {
			return nil, err
		}
		careTeam, err := w.dataAPI.GetActiveMembersOfCareTeamForCase(pcase.ID.Int64(), true)
		if err != nil {
			return nil, err
		}

		caseCtx := &caseContext{
			Case:     pcase,
			CareTeam: careTeam,
		}
		ctx.Cases = append(ctx.Cases, caseCtx)

		msgs, err := w.dataAPI.ListCaseMessages(pcase.ID.Int64(), api.PATIENT_ROLE)
		if err != nil {
			return nil, err
		}
		if len(msgs) != 0 {
			pars, err := w.dataAPI.CaseMessageParticipants(pcase.ID.Int64(), true)
			if err != nil {
				return nil, err
			}

			for _, m := range msgs {
				msg := &caseMessage{
					Time: m.Time,
					Body: m.Body,
				}
				p := pars[m.PersonID]
				switch p.Person.RoleType {
				case api.DOCTOR_ROLE, api.MA_ROLE:
					msg.SenderName = p.Person.Doctor.LongDisplayName
				case api.PATIENT_ROLE:
					msg.SenderName = p.Person.Patient.FirstName + " " + p.Person.Patient.LastName
				}
				for _, a := range m.Attachments {
					switch a.ItemType {
					case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
						mediaURL, err := signedMediaURL(w.signer, w.webDomain, pcase.PatientID.Int64(), a.ItemID)
						if err != nil {
							return nil, err
						}
						msg.Media = append(msg.Media, &caseMedia{
							Type: a.ItemType,
							URL:  mediaURL,
						})
					}
				}
				caseCtx.Messages = append(caseCtx.Messages, msg)
			}
		}

		for _, visit := range visits {
			layout, err := patient_file.VisitReviewLayout(w.dataAPI, w.mediaStore, w.expirationDuration, visit, w.apiDomain)
			if err != nil {
				return nil, err
			}

			visitCtx := &visitContext{
				Visit: visit,
			}

			visitCtx.Diagnosis, err = w.dataAPI.DiagnosisForVisit(visit.PatientVisitID.Int64())
			if !api.IsErrNotFound(err) && err != nil {
				return nil, err
			}

			caseCtx.Visits = append(caseCtx.Visits, visitCtx)

			buf := &bytes.Buffer{}
			lr := &intakeLayoutRenderer{
				wr:        buf,
				webDomain: w.webDomain,
				signer:    w.signer,
				patientID: patient.PatientID.Int64(),
			}
			if err := lr.render(layout); err != nil {
				return nil, err
			}

			visitCtx.IntakeHTML = template.HTML(buf.String())
		}

		treatmentPlans, err := w.dataAPI.GetTreatmentPlansForCase(pcase.ID.Int64())
		if api.IsErrNotFound(err) {
			continue
		} else if err != nil {
			return nil, err
		}

		sort.Sort(byStatus(treatmentPlans))

		for _, tp := range treatmentPlans {
			tpCtx := &treatmentPlanContext{
				TreatmentPlan: tp,
			}
			caseCtx.TreatmentPlans = append(caseCtx.TreatmentPlans, tpCtx)

			doctor, err := w.dataAPI.GetDoctorFromID(tp.DoctorID.Int64())
			if err != nil {
				return nil, err
			}
			tpCtx.Doctor = doctor

			buf := &bytes.Buffer{}
			if err := treatment_plan.RenderTreatmentPlan(buf, w.dataAPI, tp, doctor, patient); err != nil {
				return nil, err
			}
			tpCtx.HTML = template.HTML(buf.String())
		}
	}

	buf := &bytes.Buffer{}

	if err := mrTemplate.Execute(buf, ctx); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type byStatus []*common.TreatmentPlan

func (tp byStatus) Len() int           { return len(tp) }
func (tp byStatus) Swap(i, j int)      { tp[i], tp[j] = tp[j], tp[i] }
func (tp byStatus) Less(i, j int) bool { return tp[i].Status == common.TPStatusActive }

type intakeLayoutRenderer struct {
	wr        *bytes.Buffer
	webDomain string
	signer    *sig.Signer
	patientID int64
}

func (lr *intakeLayoutRenderer) render(layout map[string]interface{}) error {
	sectionList := &info_intake.DVisitReviewSectionListView{}
	decoderConfig := &mapstructure.DecoderConfig{
		Result:   sectionList,
		TagName:  "json",
		Registry: *info_intake.DVisitReviewViewTypeRegistry,
	}

	d, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return err
	}

	if err := d.Decode(layout); err != nil {
		return err
	}

	lr.wr.WriteString(`<div class="intake">`)
	for _, s := range sectionList.Sections {
		lr.wr.WriteString(`<div class="section">`)
		if err := lr.renderView(s); err != nil {
			return err
		}
		lr.wr.WriteString(`</div>`)
	}
	lr.wr.WriteString(`</div>`)

	return nil
}

func (lr *intakeLayoutRenderer) renderView(view common.View) error {
	if view == nil {
		return nil
	}

	switch v := view.(type) {
	default:
		return fmt.Errorf("unknown view type %T", view)
	case *info_intake.DVisitReviewStandardPhotosSectionView:
		lr.wr.WriteString(`<div class="standard-photos-section">`)
		lr.wr.WriteString(`<h3>` + v.Title + `</h3>`)
		for _, s := range v.Subsections {
			if err := lr.renderView(s); err != nil {
				return err
			}
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewStandardPhotosSubsectionView:
		lr.wr.WriteString(`<div class="standard-photos-subsection">`)
		if err := lr.renderView(v.SubsectionView); err != nil {
			return err
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewStandardPhotosListView:
		// TODO: this seems currently unused
		return fmt.Errorf("DVisitReviewStandardPhotosListView not supported")
	case *info_intake.DVisitReviewTitlePhotosItemsListView:
		lr.wr.WriteString(`<div class="title-photos-items-list">`)
		for _, it := range v.Items {
			lr.wr.WriteString(`<h4>` + it.Title + `</h4>`)
			lr.wr.WriteString(`<div class="row">`)
			for _, p := range it.Photos {
				lr.wr.WriteString(fmt.Sprintf(`<div class="col-xs-4">%s</div>`, p.Title))
			}
			lr.wr.WriteString(`</div>`)
			lr.wr.WriteString(`<div class="row">`)
			for _, p := range it.Photos {
				mediaURL, err := signedMediaURL(lr.signer, lr.webDomain, lr.patientID, p.PhotoID)
				if err != nil {
					return err
				}
				lr.wr.WriteString(fmt.Sprintf(`<div class="col-xs-4"><img src="%s"></div>`, mediaURL))
			}
			lr.wr.WriteString(`</div>`)
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewStandardSectionView:
		lr.wr.WriteString(`<div class="standard-section">`)
		lr.wr.WriteString(`<h3>` + v.Title + `</h3>`)
		for _, s := range v.Subsections {
			if err := lr.renderView(s); err != nil {
				return err
			}
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewStandardSubsectionView:
		lr.wr.WriteString(`<div class="standard-subsection">`)
		lr.wr.WriteString(`<h4>` + v.Title + `</h4>`)
		for _, r := range v.Rows {
			if err := lr.renderView(r); err != nil {
				return err
			}
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewStandardOneColumnRowView:
		lr.wr.WriteString(`<div class="standard-one-column-row">`)
		if err := lr.renderView(v.SingleView); err != nil {
			return err
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewStandardTwoColumnRowView:
		lr.wr.WriteString(`<div class="standard-two-column-row row">`)
		lr.wr.WriteString(`<div class="col-xs-6 left">`)
		if err := lr.renderView(v.LeftView); err != nil {
			return err
		}
		lr.wr.WriteString(`</div>`)
		lr.wr.WriteString(`<div class="col-xs-6 right">`)
		if err := lr.renderView(v.RightView); err != nil {
			return err
		}
		lr.wr.WriteString(`</div>`)
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewDividedViewsList:
		lr.wr.WriteString(`<div class="divided-views-list">`)
		for _, d := range v.DividedViews {
			if err := lr.renderView(d); err != nil {
				return err
			}
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewAlertLabelsList:
		lr.wr.WriteString(`<div class="alert-labels-list">`)
		if len(v.Values) == 0 {
			if err := lr.renderView(v.EmptyStateView); err != nil {
				return err
			}
		} else {
			lr.wr.WriteString(`<h4>Alerts</h4>`)
			lr.wr.WriteString(`<ul>`)
			for _, a := range v.Values {
				lr.wr.WriteString(`<li class="alert">` + a + `</li>`)
			}
			lr.wr.WriteString(`</ul>`)
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewTitleLabelsList:
		lr.wr.WriteString(`<div class="title-labels-list">`)
		for _, s := range v.Values {
			lr.wr.WriteString(`<div>` + s + `</div>`)
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewContentLabelsList:
		lr.wr.WriteString(`<div class="content-labels-list">`)
		if len(v.Values) == 0 {
			if err := lr.renderView(v.EmptyStateView); err != nil {
				return err
			}
		} else {
			for _, s := range v.Values {
				lr.wr.WriteString(`<div class="content-label">` + s + `</div>`)
			}
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewCheckXItemsList:
		lr.wr.WriteString(`<div class="check-x-items-list">`)
		for _, it := range v.Items {
			if it.IsChecked {
				lr.wr.WriteString(`<div class="checked"><span class="glyphicon glyphicon-ok"></span> ` + it.Value + `</div>`)
			} else {
				lr.wr.WriteString(`<div class="notchecked"><span class="glyphicon glyphicon-remove"></span> ` + it.Value + `</div>`)
			}
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewTitleSubItemsLabelContentItemsList:
		lr.wr.WriteString(`<div class="title-sub-items-label-content-items-list">`)
		if len(v.Items) == 0 {
			if err := lr.renderView(v.EmptyStateView); err != nil {
				return err
			}
		} else {
			lr.wr.WriteString(`<div class="item">`)
			for _, it := range v.Items {
				lr.wr.WriteString(`<h4>` + it.Title + `</h4>`)
				for _, d := range it.SubItems {
					lr.wr.WriteString(`<div><strong>` + d.Content + `</strong></div>`)
					lr.wr.WriteString(`<div>` + d.Description + `</div>`)
				}
			}
			lr.wr.WriteString(`</div>`)
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewTitleSubtitleLabels:
		lr.wr.WriteString(`<div class="title-subtitle-labels">`)
		if v.Title == "" {
			if err := lr.renderView(v.EmptyStateView); err != nil {
				return err
			}
		} else {
			lr.wr.WriteString(`<h4>` + v.Title + `</h4>`)
			if v.Subtitle != "" {
				lr.wr.WriteString(`<div class="subtitle">` + v.Subtitle + `</div>`)
			}
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewEmptyLabelView:
		lr.wr.WriteString(`<div class="empty-label-view">`)
		lr.wr.WriteString(v.Text)
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewEmptyTitleSubtitleLabelView:
		lr.wr.WriteString(`<div class="empty-title-subtitle-label-view">`)
		lr.wr.WriteString(`<div class="title"><strong>` + v.Title + `</strong></div>`)
		if v.Subtitle != "" {
			lr.wr.WriteString(`<div class="subtitle">` + v.Subtitle + `</div>`)
		}
		lr.wr.WriteString(`</div>`)
	}
	return nil
}
