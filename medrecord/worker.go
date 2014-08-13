package medrecord

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/email"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/third_party/github.com/SpruceHealth/mapstructure"
	"github.com/sprucehealth/backend/treatment_plan"
)

const (
	batchSize         = 1
	visibilityTimeout = 60 * 5
	waitTimeSeconds   = 60 * 10
)

type worker struct {
	dataAPI      api.DataAPI
	queue        *common.SQSQueue
	emailService email.Service
	supportEmail string
	store        storage.Store
	apiDomain    string
	webDomain    string
	signer       *common.Signer
}

func StartWorker(dataAPI api.DataAPI, queue *common.SQSQueue, emailService email.Service, supportEmail, apiDomain, webDomain string, signer *common.Signer, store storage.Store) {
	(&worker{
		dataAPI:      dataAPI,
		queue:        queue,
		emailService: emailService,
		supportEmail: supportEmail,
		store:        store,
		apiDomain:    apiDomain,
		webDomain:    webDomain,
		signer:       signer,
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
		msg := &queueMessage{}
		if err := json.Unmarshal([]byte(m.Body), msg); err != nil {
			golog.Errorf(err.Error())
			continue
		}
		if err := w.processMessage(msg); err != nil {
			golog.Errorf(err.Error())
		} else {
			if err := w.queue.QueueService.DeleteMessage(w.queue.QueueUrl, m.ReceiptHandle); err != nil {
				golog.Errorf(err.Error())
			}
		}
	}

	return nil
}

func (w *worker) processMessage(msg *queueMessage) error {
	mr, err := w.dataAPI.MedicalRecord(msg.MedicalRecordID)
	if err == api.NoRowsError {
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

	patient, err := w.dataAPI.GetPatientFromId(mr.PatientID)
	if err == api.NoRowsError {
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

	if err := w.emailService.SendEmail(&email.Email{
		From:    w.supportEmail,
		To:      patient.Email,
		Subject: "Spruce medical record",
		BodyText: `Hello,

We have generated your Spruce medical record which you may download from our website at the following URL.

` + downloadURL,
	}); err != nil {
		golog.Errorf("Failed to send medical record email for record %d to patient %d: %s", mr.ID, patient.PatientId.Int64(), err.Error())
	}

	return nil
}

func (w *worker) generateHTML(patient *common.Patient) ([]byte, error) {
	ctx := &templateContext{
		Patient: patient,
	}

	ag, err := w.dataAPI.PatientAgreements(patient.PatientId.Int64())
	if err != nil {
		return nil, err
	}
	ctx.Agreements = ag

	pcp, err := w.dataAPI.GetPatientPCP(patient.PatientId.Int64())
	if err != nil {
		return nil, err
	}
	ctx.PCP = pcp

	ec, err := w.dataAPI.GetPatientEmergencyContacts(patient.PatientId.Int64())
	if err != nil {
		return nil, err
	}
	ctx.EmergencyContacts = ec

	cases, err := w.dataAPI.GetCasesForPatient(patient.PatientId.Int64())
	if err != nil {
		return nil, err
	}

	for _, pcase := range cases {
		visits, err := w.dataAPI.GetVisitsForCase(pcase.Id.Int64())
		if err != nil {
			return nil, err
		}
		careTeam, err := w.dataAPI.GetActiveMembersOfCareTeamForCase(pcase.Id.Int64(), true)
		if err != nil {
			return nil, err
		}

		caseCtx := &caseContext{
			Case:     pcase,
			CareTeam: careTeam,
		}
		ctx.Cases = append(ctx.Cases, caseCtx)

		caseCtx.Messages, err = w.dataAPI.ListCaseMessages(pcase.Id.Int64(), api.PATIENT_ROLE)
		if err != nil {
			return nil, err
		}
		if len(caseCtx.Messages) != 0 {
			caseCtx.MessageParticipants, err = w.dataAPI.CaseMessageParticipants(pcase.Id.Int64(), true)
			if err != nil {
				return nil, err
			}
		}

		for _, visit := range visits {
			layout, err := patient_file.VisitReviewLayout(w.dataAPI, visit, w.apiDomain)
			if err != nil {
				return nil, err
			}

			visitCtx := &visitContext{
				Visit: visit,
			}
			caseCtx.Visits = append(caseCtx.Visits, visitCtx)

			buf := &bytes.Buffer{}
			lr := &intakeLayoutRenderer{
				wr:        buf,
				webDomain: w.webDomain,
				signer:    w.signer,
				patientID: patient.PatientId.Int64(),
			}
			if err := lr.render(layout); err != nil {
				return nil, err
			}

			visitCtx.IntakeHTML = template.HTML(buf.String())
		}

		treatmentPlans, err := w.dataAPI.GetAllTreatmentPlansForCase(pcase.Id.Int64())
		if err == api.NoRowsError {
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

			doctor, err := w.dataAPI.GetDoctorFromId(tp.DoctorId.Int64())
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
func (tp byStatus) Less(i, j int) bool { return tp[i].Status == "ACTIVE" }

type intakeLayoutRenderer struct {
	wr        *bytes.Buffer
	webDomain string
	signer    *common.Signer
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
				sig, err := lr.signer.Sign([]byte(fmt.Sprintf("patient:%d:photo:%d", lr.patientID, p.PhotoID)))
				if err != nil {
					return err
				}
				sigStr := base64.URLEncoding.EncodeToString(sig)
				lr.wr.WriteString(fmt.Sprintf(`<div class="col-xs-4"><img src="https://%s/patient/medical-record/photo/%d?s=%s"></div>`, lr.webDomain, p.PhotoID, sigStr))
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

type visitContext struct {
	Visit      *common.PatientVisit
	IntakeHTML template.HTML
}

type treatmentPlanContext struct {
	TreatmentPlan *common.TreatmentPlan
	Doctor        *common.Doctor
	HTML          template.HTML
}

type caseContext struct {
	Case                *common.PatientCase
	CareTeam            []*common.CareProviderAssignment
	Messages            []*common.CaseMessage
	MessageParticipants map[int64]*common.CaseMessageParticipant
	Visits              []*visitContext
	TreatmentPlans      []*treatmentPlanContext
}

type templateContext struct {
	Patient           *common.Patient
	PCP               *common.PCP
	EmergencyContacts []*common.EmergencyContact
	Cases             []*caseContext
	Agreements        map[string]time.Time
}

var mrTemplate = template.Must(template.New("").Funcs(map[string]interface{}{
	"formatDOB": func(dob encoding.DOB) string {
		return fmt.Sprintf("%s %d, %d", time.Month(dob.Month).String(), dob.Day, dob.Year)
	},
	"formatDateTime": func(t time.Time) string {
		return t.Format("Jan _2 15:04:05 MST")
	},
}).Parse(`<!DOCTYPE html>
<html>
<head>
	<title>Medical Record</title>
	<link rel="stylesheet" type="text/css" href="//maxcdn.bootstrapcdn.com/bootstrap/3.2.0/css/bootstrap.min.css">
	<style type="text/css">
	html,body {
		padding-top: 20px;
		padding-bottom: 20px;
	}
	.title-labels-list {
		font-weight: bold;
	}
	.title-photos-items-list img {
		width: 100%;
		height: 100%;
	}
	.standard-two-column-row > div {
		border-top: 1px solid #ddd;
	}
	/* .standard-two-column-row > div.left {
		background-color: #eee;
	} */
	.standard-subsection h4 {
		margin-top: 20px;
		margin-bottom: 20px;
	}
	.alert {
		color: red;
	}
	.check-x-items-list .checked {
		color: #0a0;
	}
	.check-x-items-list .notchecked {
		color: #aaa;
	}
	.small-divider {
		border-top: 0;
	}
	.treatment-plan h3 {
	}
	.hero-header .title {
		font-size: 16px;
	}
	</style>
	<style type="text/css" media="print">
	.print {
		display: none;
	}
	</style>
</head>
<body>
	<div class="container">
		<div class="pull-right print">
			<button type="button" class="btn btn-default btn-lg" onclick="javascript:window.print()">
				<span class="glyphicon glyphicon-print"></span> print
			</button>
		</div>

		<h2>Patient Information</h2>

		<div><strong>Name:</strong> {{.Patient.FirstName}} {{.Patient.LastName}}</div>
		<div><strong>Gender:</strong> {{.Patient.Gender}}</div>
		<div><strong>DOB:</strong> {{formatDOB .Patient.DOB}}</div>
		<div><strong>Email:</strong> {{.Patient.Email}}</div>

		{{with .Patient.PhoneNumbers}}
		<div class="phone-numbers">
			<h4>Phone Numbers</h4>
			{{range .}}
				<div><strong>{{.Type}}:</strong> {{.Phone}}</div>
			{{end}}
		</div>
		{{end}}

		{{with .Patient.PatientAddress}}
		<div class="address">
			<h4>Address</h4>
			<div>
				{{.AddressLine1}}<br>
				{{with .AddressLine2}}{{.}}<br>{{end}}
				{{.City}}, {{.State}}<br>
				{{.ZipCode}}<br>
			<div>
		</div>
		{{end}}

		{{with .PCP}}
		<div class="pcp">
			<h4>Primary Care Provider</h4>
			<div><strong>Physician name:</strong> {{.PhysicianName}}</div>
			<div><strong>Practice name:</strong> {{.PracticeName}}</div>
			<div><strong>Email:</strong> {{.Email}}</div>
			<div><strong>Phone number:</strong> {{.PhoneNumber}}</div>
			<div><strong>Fax number:</strong> {{.FaxNumber}}</div>
		</div>
		{{end}}

		{{with .EmergencyContacts}}
		<div class="emergency-contacts">
			<h4>Emergency Contacts</h4>
			{{range .}}
			<div>
				<div><strong>Name:</strong> {{.FullName}}</div>
				<div><strong>Phone number:</strong> {{.PhoneNumber}}</div>
				<div><strong>Relationship:</strong> {{.Relationship}}</div>
			</div>
			{{end}}
		</div>
		{{end}}

		{{with .Agreements}}
		<div class="agreements">
			<h4>Agreements</h4>
			<ul>
				{{range $name, $date := .}}
				<li>{{$name}} on {{formatDateTime $date}}</li>
				{{end}}
			</ul>
		</div>
		{{end}}

		{{range .Cases}}
			<h2>{{.Case.MedicineBranch}} Case</h2>

			{{with .CareTeam}}
				<div class="care-team">
					<h3>Care Team</h3>
					{{range .}}
						<div><strong>{{.ProviderRole}}:</strong> {{.LongDisplayName}}</div>
					{{end}}
				</div>
			{{end}}

			{{with .Case.Diagnosis}}
				<div class="diagnosis">
					<strong>Diagnosis:</strong> {{.}}
				</div>
			{{end}}

			{{range .Visits}}
				<div class="visit">
					{{with .Visit.Diagnosis}}
						<div class="diagnosis">
							<strong>Diagnosis:</strong> {{.}}
						</div>
					{{end}}

					{{.IntakeHTML}}
				</div>
			{{end}}

			{{range .TreatmentPlans}}
				<div class="treatment-plan">
					<hr>
					<h3>{{.TreatmentPlan.Status}} Treatment Plan</h3>
					<div class="doctor">
						<h4>Doctor</h4>
						<div>{{.Doctor.LongDisplayName}}</div>
						<div>{{.Doctor.LongTitle}}</div>
					</div>
					{{.HTML}}
				</div>
			{{end}}
		{{end}}
	</div>
</body>
</html>`))
