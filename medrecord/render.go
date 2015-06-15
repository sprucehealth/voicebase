package medrecord

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/SpruceHealth/mapstructure"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/treatment_plan"
)

func signedMediaURL(signer *sig.Signer, webDomain string, patientID, mediaID int64) (string, error) {
	sig, err := signer.Sign([]byte(fmt.Sprintf("patient:%d:media:%d", patientID, mediaID)))
	if err != nil {
		return "", err
	}
	sigStr := base64.URLEncoding.EncodeToString(sig)
	return fmt.Sprintf("https://%s/patient/medical-record/media/%d?s=%s", webDomain, mediaID, sigStr), nil
}

type visitContext struct {
	Visit            *common.PatientVisit
	DiagnosisSet     *common.VisitDiagnosisSet
	DiagnosisDetails map[string]*diagnosis.Diagnosis
	IntakeHTML       template.HTML
}

type treatmentPlanContext struct {
	TreatmentPlan *common.TreatmentPlan
	Doctor        *common.Doctor
	HTML          template.HTML
}

type caseMedia struct {
	Type string
	URL  string
}

type caseMessage struct {
	Time       time.Time
	SenderName string
	Body       string
	Media      []*caseMedia
}

type caseContext struct {
	Case           *common.PatientCase
	CareTeam       []*common.CareProviderAssignment
	Messages       []*caseMessage
	Visits         []*visitContext
	TreatmentPlans []*treatmentPlanContext
}

type templateContext struct {
	Patient           *common.Patient
	PCP               *common.PCP
	EmergencyContacts []*common.EmergencyContact
	Cases             []*caseContext
	Agreements        map[string]time.Time
}

var mrTemplate = template.Must(template.New("").Funcs(map[string]interface{}{
	"formatDOB": func(dob encoding.Date) string {
		// Not set
		if dob.Month == 0 {
			return ""
		}
		return fmt.Sprintf("%s %d, %d", time.Month(dob.Month).String(), dob.Day, dob.Year)
	},
	"formatDateTime": func(t time.Time) string {
		return t.Format("Jan _2, 2006 3:04pm MST")
	},
	"titleCase": func(s string) string {
		return strings.Title(strings.ToLower(s))
	},
}).Parse(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>Medical Record</title>
	<link rel="stylesheet" type="text/css" href="//maxcdn.bootstrapcdn.com/bootstrap/3.2.0/css/bootstrap.min.css">
	<style type="text/css">
	html,body {
		padding-top: 20px;
		padding-bottom: 20px;
	}
	h1 {
		margin-bottom: 20px;
	}
	.case {
		border: 1px solid #ccc;
		padding: 13px;
		margin-bottom: 25px;
	}
	.case h2 {
		background-color: #444;
		color: #fff;
		padding: 10px;
		margin: -14px -14px 10px -14px;
	}
	.case h2 span {
		float: right;
		font-size: 20px;
		line-height: 31px;
	}
	.title-labels-list {
		font-weight: bold;
	}
	.title-photos-items-list img {
		width: 100%;
		height: 100%;
	}
	.visit > h3,
	.messages > h3 {
		margin: 5px -14px 10px -14px;
		background-color: #777;
		color: #fff;
		padding: 13px;
	}
	.visit > h3 span,
	.treatment-plan > h3 span {
		float: right;
		font-size: 18px;
		line-height: 28px;
		font-weight: normal;
	}
	.section {
	}
	.section h3,
	.treatment-plan > h3 {
		border-radius: 5px;
		background-color: #eee;
		padding: 10px;
	}
	.standard-section {

	}
	.standard-subsection,
	.standard-photos-subsection,
	.treatment-plan .doctor,
	.treatment-plan .header,
	.treatment-plan .treatments,
	.treatment-plan .instructions {
		margin-left: 10px;
	}
	.treatment-plan > div > .title {
		font-size: 24px;
		line-height: 30px;
		margin-top: 20px;
		border-bottom: 1px solid #444;
		margin-bottom: 15px;
	}
	.treatment-plan .button-footer {
		margin-top: 20px;
	}
	.standard-two-column-row > div {
		border-top: 1px solid #ddd;
	}
	.standard-subsection h4 {
		margin-top: 20px;
		margin-bottom: 20px;
		font-weight: bold;
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
		display: none;
	}
	.hero-header .title {
		font-size: 16px;
	}
	.message {
		padding-top: 10px;
	}
	.message .sender {
		font-weight: bold;
	}
	.message .time {
	}
	.message img {
		width: 100%;
		height: 100%;
		margin-top: 10px;
	}
	.prescription {
		margin-top: 5px;
	}
	.prescription .title {
		font-weight: bold;
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

		<h1>Patient Information</h1>

		<div class="row">
			<div class="col-md-6 col-sm-6">
				<table role="table" class="table">
				<tr>
					<th>Name</th>
					<td>{{.Patient.FirstName}} {{.Patient.LastName}}</td>
				</tr>
				<tr>
					<th>Gender</th>
					<td>{{.Patient.Gender}}</td>
				</tr>
				<tr>
					<th>DOB</th>
					<td>{{formatDOB .Patient.DOB}}</td>
				</tr>
				<tr>
					<th>Email</th>
					<td>{{.Patient.Email}}</td>
				</tr>
				<tr>
					<th>Primary Care Provider</th>
					<td>
						{{with .PCP}}
						<strong>Physician name:</strong> {{.PhysicianName}}<br>
						<strong>Practice name:</strong> {{.PracticeName}}<br>
						<strong>Email:</strong> {{.Email}}<br>
						<strong>Phone number:</strong> {{.PhoneNumber}}<br>
						<strong>Fax number:</strong> {{.FaxNumber}}
						{{end}}
					</td>
				</tr>
				<tr>
					<th>Emergency Contacts</th>
					<td>
						{{range .EmergencyContacts}}
						<strong>Name:</strong> {{.FullName}}<br>
						<strong>Phone number:</strong> {{.PhoneNumber}}<br>
						<strong>Relationship:</strong> {{.Relationship}}
						{{end}}
					</td>
				</tr>
				</table>
			</div>
			<div class="col-md-6 col-sm-6">
				<table role="table" class="table">
				<tr>
					<th>Phone Numbers</th>
					<td>
						{{range $i, $num := .Patient.PhoneNumbers}}
							{{if $i}}<br>{{end}}
							<strong>{{.Type}}:</strong> {{.Phone}}
						{{end}}
					</td>
				</tr>
				<tr>
					<th>Address</th>
					<td>
						{{with .Patient.PatientAddress}}
							{{.AddressLine1}}<br>
							{{with .AddressLine2}}{{.}}<br>{{end}}
							{{.City}}, {{.State}}<br>
							{{.ZipCode}}
						{{end}}
					</td>
				</tr>
				<tr>
					<th>Preferred Pharmacy</th>
					<td>
						{{with .Patient.Pharmacy}}
							{{.Name}}<br>
							{{.AddressLine1}}<br>
							{{with .AddressLine2}}{{.}}<br>{{end}}
							Phone: {{.Phone}}<br>
							{{.City}}, {{.State}}<br>
							{{.Postal}}
						{{end}}
					</td>
				</tr>
				</table>
			</div>
		</div>

		{{if .Cases}}
		<h1>Cases</h1>
		{{end}}

		{{range .Cases}}
		<div class="case">
			<h2>
				{{.Case.Name}} Case
				<span>{{.Case.Status}}</span>
			</h2>

			{{with .CareTeam}}
				<div class="section care-team">
					<h3>Care Team</h3>
					<table class="table" role="table">
					{{range .}}
					<tr>
						<th>{{.ProviderRole}}</th><td>{{.LongDisplayName}}</td>
					</tr>
					{{end}}
					</table>
				</div>
			{{end}}

			{{range .Visits}}
				<div class="visit">
					<h3>
						{{if .Visit.IsFollowup}}Follow-up {{end}}Visit
						<span>Submitted {{.Visit.SubmittedDate|formatDateTime}}</span>
					</h3>

					{{$diagnosisDetails := .DiagnosisDetails}}
					{{with .DiagnosisSet}}
						<div class="diagnosis section">
							<h3>Diagnosis</h3>
							<div class="standard-subsection">
								{{if .Unsuitable}}
									<strong>Unsuitable for Spruce:</strong> {{.UnsuitableReason}}
								{{else}}
									{{range .Items}}
										{{with index $diagnosisDetails .CodeID}}
										<strong>{{.Code}}</strong> {{.Description}}<br>
										{{end}}
									{{end}}
									{{with .Notes}}
									<h4>Notes</h4>
									<pre>{{.}}
									{{end}}
								{{end}}
							</div>
						</div>
					{{end}}

					{{.IntakeHTML}}
				</div>
			{{end}}

			{{range .TreatmentPlans}}
				<div class="treatment-plan">
					<hr>
					<h3>
						{{.TreatmentPlan.Status.String|titleCase}} Treatment Plan
						{{with .TreatmentPlan.SentDate}}
							<span>Sent {{.|formatDateTime}}
						{{else}}
							{{with .TreatmentPlan.CreationDate}}
							<span>Created {{.|formatDateTime}}
							{{end}}
						{{end}}
					</h3>
					<div class="doctor">
						<h4>Doctor</h4>
						<div>{{.Doctor.LongDisplayName}}</div>
						<div>{{.Doctor.LongTitle}}</div>
					</div>
					{{with .TreatmentPlan.Note}}
					<div class="note">
						<h4>Note</h4>
						<pre>{{.}}</pre>
					</div>
					{{end}}
					{{.HTML}}
				</div>
			{{end}}

			{{with .Messages}}
				<div class="messages">
					<h3>Messages</h3>
					{{range .}}
						<div class="message">
							<div class="header">
								<span class="sender">{{.SenderName}}</span>
								<span class="time">at {{formatDateTime .Time}}</span>
							</div>
							<pre class="body">{{.Body}}</pre>
							<div class="media row">
								{{range .Media}}
									<div class="col-xs-4">
										{{if eq .Type "photo"}}
											<img src="{{.URL}}">
										{{end}}
										{{if eq .Type "audio"}}
											<audio controls>
												<source src="{{.URL}}">
												Your browser does not support the audio element.
											</audio>
										{{end}}
									</div>
								{{end}}
							</div>
						</div>
					{{end}}
				</div>
			{{end}}
		</div>
		{{end}}
	</div>
</body>
</html>`))

type Renderer struct {
	DataAPI            api.DataAPI
	DiagnosisSvc       diagnosis.API
	MediaStore         *media.Store
	APIDomain          string
	WebDomain          string
	Signer             *sig.Signer
	ExpirationDuration time.Duration
}

func (r *Renderer) Render(patient *common.Patient) ([]byte, error) {
	ctx := &templateContext{
		Patient: patient,
	}

	ag, err := r.DataAPI.PatientAgreements(patient.ID.Int64())
	if err != nil {
		return nil, err
	}
	ctx.Agreements = ag

	pcp, err := r.DataAPI.GetPatientPCP(patient.ID.Int64())
	if err != nil {
		return nil, err
	}
	ctx.PCP = pcp

	ec, err := r.DataAPI.GetPatientEmergencyContacts(patient.ID.Int64())
	if err != nil {
		return nil, err
	}
	ctx.EmergencyContacts = ec

	cases, err := r.DataAPI.GetCasesForPatient(patient.ID.Int64(), append(common.SubmittedPatientCaseStates(), common.PCStatusUnsuitable.String()))
	if err != nil {
		return nil, err
	}

	for _, pcase := range cases {
		visits, err := r.DataAPI.GetVisitsForCase(pcase.ID.Int64(), common.TreatedPatientVisitStates())
		if err != nil {
			return nil, err
		}
		careTeam, err := r.DataAPI.GetActiveMembersOfCareTeamForCase(pcase.ID.Int64(), true)
		if err != nil {
			return nil, err
		}

		caseCtx := &caseContext{
			Case:     pcase,
			CareTeam: careTeam,
		}
		ctx.Cases = append(ctx.Cases, caseCtx)

		msgs, err := r.DataAPI.ListCaseMessages(pcase.ID.Int64(), api.LCMOIncludePrivate)
		if err != nil {
			return nil, err
		}
		if len(msgs) != 0 {
			pars, err := r.DataAPI.CaseMessageParticipants(pcase.ID.Int64(), true)
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
				case api.RoleDoctor, api.RoleCC:
					msg.SenderName = p.Person.Doctor.LongDisplayName
				case api.RolePatient:
					msg.SenderName = p.Person.Patient.FirstName + " " + p.Person.Patient.LastName
				}
				for _, a := range m.Attachments {
					switch a.ItemType {
					case common.AttachmentTypePhoto, common.AttachmentTypeAudio:
						mediaURL, err := signedMediaURL(r.Signer, r.WebDomain, pcase.PatientID.Int64(), a.ItemID)
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
			layout, err := patient_file.VisitReviewLayout(r.DataAPI, r.MediaStore, r.ExpirationDuration, visit, r.APIDomain)
			if err != nil {
				return nil, err
			}

			visitCtx := &visitContext{
				Visit: visit,
			}

			visitCtx.DiagnosisSet, err = r.DataAPI.ActiveDiagnosisSet(visit.PatientVisitID.Int64())
			if !api.IsErrNotFound(err) {
				if err != nil {
					return nil, err
				}
				codeIDs := make([]string, len(visitCtx.DiagnosisSet.Items))
				for i, d := range visitCtx.DiagnosisSet.Items {
					codeIDs[i] = d.CodeID
				}
				visitCtx.DiagnosisDetails, err = r.DiagnosisSvc.DiagnosisForCodeIDs(codeIDs)
				if err != nil {
					return nil, err
				}
			}

			caseCtx.Visits = append(caseCtx.Visits, visitCtx)

			buf := &bytes.Buffer{}
			lr := &intakeLayoutRenderer{
				wr:        buf,
				webDomain: r.WebDomain,
				signer:    r.Signer,
				patientID: patient.ID.Int64(),
			}
			if err := lr.render(layout); err != nil {
				return nil, err
			}

			visitCtx.IntakeHTML = template.HTML(buf.String())
		}

		treatmentPlans, err := r.DataAPI.GetTreatmentPlansForCase(pcase.ID.Int64())
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

			doctor, err := r.DataAPI.GetDoctorFromID(tp.DoctorID.Int64())
			if err != nil {
				return nil, err
			}
			tpCtx.Doctor = doctor

			buf := &bytes.Buffer{}
			if err := treatment_plan.RenderTreatmentPlan(buf, r.DataAPI, tp, doctor, patient); err != nil {
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
		if len(v.Rows) != 0 {
			lr.wr.WriteString(`<div class="standard-subsection">`)
			lr.wr.WriteString(`<h4>` + v.Title + `</h4>`)
			for _, r := range v.Rows {
				if err := lr.renderView(r); err != nil {
					return err
				}
			}
			lr.wr.WriteString(`</div>`)
		}
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
