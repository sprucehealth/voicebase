package medrecord

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/patient_file"
	"github.com/sprucehealth/backend/treatment_plan"
	"github.com/sprucehealth/mapstructure"
)

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
	Parent            *parentContext
	Cases             []*caseContext
	Agreements        map[string]time.Time
}

type parentContext struct {
	Patient *common.Patient
	Consent *common.ParentalConsent
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
	<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no">
	<meta http-equiv="X-UA-Compatible" content="IE=Edge">
	<title>Medical Record</title>
	<link rel="stylesheet" type="text/css" href="//maxcdn.bootstrapcdn.com/bootstrap/3.2.0/css/bootstrap.min.css">
	<style type="text/css">
	html,body {
		padding-top: 20px;
		padding-bottom: 20px;
		font-family: HelveticaNeue;
		font-size: 14px;
		line-height: 26px;
		color: #000000;
		background-color: #fff;
	}
	font-300 {
		font-family: HelveticaNeue; /*MuseoSans-300;*/
	}
	font-bold {
		font-family: HelveticaNeue-Bold;
	}
	font-size-14 {
		font-size: 14px;
		line-height: 26px;
	}
	font-size-16 {
		font-size: 16px;
		line-height: 19px;
	}
	color-dark {
		color: #1E333A;
	}
	strong {
		font-family: HelveticaNeue-Bold;
	}
	h1 {
		margin-bottom: 20px;
		padding: 10px;
		border-radius: 4px;
		background: #1E333A;
		font-family: HelveticaNeue-Bold;
		font-size: 16px;
		color: #FFFFFF;
		line-height: 19px;
		text-align: center;
	}
	div.table-wrapper {
		border: 1px solid #DAE6EB;
		border-radius: 4px;
		margin-bottom: 20px;
	}
	.box {
		border: 1px solid #DAE6EB;
		border-radius: 4px;
		margin-bottom: 20px;
	}
	.box-header {
		font-family: HelveticaNeue-Bold;
		font-size: 18px;
		line-height: 20px;
		border-bottom: 1px solid #DAE6EB;
		padding: 15px 12px;
	}
	.box-body {
		padding: 12px;
	}
	table.table {
		border: 0;
		margin-bottom: 0;
	}
	table.table th {
		font-family: HelveticaNeue-Bold;
		border-right: 1px solid #DAE6EB;
	}
	.case {
		margin-bottom: 25px;
	}
	h2 {
		background: #F1F5F6;
		border-radius: 4px;
		font-family: HelveticaNeue-Bold;
		font-size: 16px;
		color: #1E333A;
		line-height: 19px;
		padding: 12px;
		text-align: center;
	}
	.title-labels-list {
		font-weight: bold;
	}
	.title-labels-list,
	.content-labels-list {
		margin-top: 10px;
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
		border: 1px solid #DAE6EB;
		border-radius: 4px;
		margin-bottom: 20px;
	}
	.section h3,
	.treatment-plan > h3 {
		margin: 0;
		padding: 15px;
		border-bottom: 1px solid #DAE6EB;
		font-family: HelveticaNeue-Bold;
		font-size: 18px;
		color: #1E333A;
		line-height: 20px;
	}
	.standard-section {

	}
	.standard-subsection,
	.standard-photos-subsection {
		padding: 15px;
	}
	.treatment-plan .doctor,
	.treatment-plan .header,
	.treatment-plan .treatments,
	.treatment-plan .instructions {
		padding: 15px;
		margin: 10px 0;
		border: 1px solid #DAE6EB;
		border-radius: 4px;
	}
	.treatment-plan > div > .title {
		font-family: HelveticaNeue-Bold;
		font-size: 18px;
		color: #1E333A;
		line-height: 20px;
		margin-bottom: 10px;
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
		padding: 0;
		margin: 0;
	}
	.check-x-items-list .checked {
		color: #000;
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
	.message-body {
		border: 0;
		background-color: #fff;
		font-family: HelveticaNeue;
		padding: 0;
		margin: 0;
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
		<div style="margin-bottom:15px;">
			<div class="pull-right print">
				<button type="button" class="btn btn-default btn-lg" onclick="javascript:window.print()">
					<span class="glyphicon glyphicon-print"></span> print
				</button>
			</div>
			<div>
				<img src="https://d2bln09x7zhlg8.cloudfront.net/logo@2x.png" width="150" height="41" alt="spruce" />
			</div>
			<div class="clearfix"></div>
		</div>

		<p class="font-300 font-size-16 color-dark">
			This care record contains all information pertaining to the case(s) submitted, including symptoms, medical history, treatment plan and messages. If you have any questions or concerns, please contact support at support@sprucehealth.com or 800-975-7618.
		</p>

		<h1>Patient Information</h1>

		<div class="row">
			<div class="col-md-6 col-sm-12">
				<div class="table-wrapper">
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
				{{with .Patient.PhoneNumbers}}
				<tr>
					<th>Phone</th>
					<td>
						{{range $i, $num := .}}
							{{if $i}}<br>{{end}}
							{{.Phone}}
						{{end}}
					</td>
				</tr>
				{{end}}
				</table>
				</div>
			</div>
			<div class="col-md-6 col-sm-12">
				<div class="table-wrapper">
				<table role="table" class="table">
				{{with .Patient.PatientAddress}}
				<tr>
					<th>Address</th>
					<td>
						{{with .}}
							{{.AddressLine1}}<br>
							{{with .AddressLine2}}{{.}}<br>{{end}}
							{{.City}}, {{.State}}<br>
							{{.ZipCode}}
						{{end}}
					</td>
				</tr>
				{{end}}
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
		</div>

		{{with .Parent}}
		<h1>Parent Information</h1>
		<div class="table-wrapper">
			<table role="table" class="table">
			<tr>
				<th>Name</th><td>{{.Patient.FirstName}} {{.Patient.LastName}}</td>
			</tr>
			<tr>
				<th>Gender</th><td>{{.Patient.Gender}}</td>
			</tr>
			<tr>
				<th>DOB</th><td>{{formatDOB .Patient.DOB}}</td>
			</tr>
			<tr>
				<th>Phone</th>
				<td>
					{{range $i, $num := .Patient.PhoneNumbers}}
						{{if $i}}<br>{{end}}
						{{.Phone}}
					{{end}}
				</td>
			</tr>
			<tr>
				<th>Relationship to Patient</th><td>{{.Consent.Relationship}}</td>
			</tr>
			</table>
		</div>
		{{end}}

		{{range .Cases}}
		<div class="case">
			<h1>{{.Case.Name}} Case</h1>

			{{$careTeam := .CareTeam}}
			{{range .Visits}}
				<div class="visit">
					<div class="row">
						<div class="col-md-6 col-sm-12">
							<div class="box">
								<div class="box-header">
									{{if .Visit.IsFollowup}}Follow-up {{end}}Visit
								</div>
								<div class="box-body">
									{{if not .Visit.SubmittedDate.IsZero}}<div><strong>Submitted:</strong> {{.Visit.SubmittedDate|formatDateTime}}</div>{{end}}
									{{$diagnosisDetails := .DiagnosisDetails}}
									{{with .DiagnosisSet}}
									<div>
										<strong>Diagnosis:</strong>
										{{if .Unsuitable}}
											Unsuitable for Spruce: {{.UnsuitableReason}}
										{{else}}
											{{range .Items}}
												{{with index $diagnosisDetails .CodeID}}
												{{.Code}} {{.Description}}<br>
												{{end}}
											{{end}}
											{{with .Notes}}
											<h4>Notes</h4>
											<pre>{{.}}</pre>
											{{end}}
										{{end}}
									</div>
									{{end}}
								</div>
							</div>
						</div>

						<div class="col-md-6 col-sm-12">
							{{with $careTeam}}
								<div class="box">
									<div class="box-header">Care Team</div>
									<div class="box-body">
										{{range .}}
											<div>
												{{if eq .ProviderRole "DOCTOR"}}
													<strong>Doctor:</strong>
												{{else if eq .ProviderRole "MA"}}
													<strong>Care Coordinator:</strong>
												{{else}}
													<strong>{{.ProviderRole}}:</strong>
												{{end}}
												{{.LongDisplayName}}
											</div>
										{{end}}
										</table>
									</div>
								</div>
							{{end}}
						</div>
					</div>


					{{.IntakeHTML}}
				</div>
			{{end}}

			{{range .TreatmentPlans}}
				<div class="treatment-plan">
					<h2>{{.TreatmentPlan.Status.String|titleCase}} Treatment Plan</h2>

					{{.HTML}}
				</div>
			{{end}}

			{{with .Messages}}
				<div class="box" style="margin-top: 15px;">
					<div class="box-header">
						Messages
					</div>
					<div class="box-body">
					{{range .}}
						<div class="message">
							<strong>
								<span class="sender">{{.SenderName}}</span>
								<span class="time">at {{formatDateTime .Time}}</span>
							</strong>
							<br>
							<pre class="message-body">{{.Body}}</pre>
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
				</div>
			{{end}}
		</div>
		{{end}}
	</div>
</body>
</html>`))

// RenderOption is the options type for the medical record renderer.
type RenderOption int

// Render options for the medical record
const (
	ROIncludeUnsubmittedVisits RenderOption = 1 << iota
)

// Has returns true if the option is set
func (o RenderOption) Has(ro RenderOption) bool {
	return o&ro != 0
}

// Renderer can render a patient's medical record as HTML.
type Renderer struct {
	DataAPI            api.DataAPI
	DiagnosisSvc       diagnosis.API
	MediaStore         *media.Store
	APIDomain          string
	WebDomain          string
	Signer             *sig.Signer
	ExpirationDuration time.Duration
}

// Render returns the HTML version of a medical record for a patient.
func (r *Renderer) Render(patient *common.Patient, opt RenderOption) ([]byte, error) {
	ctx := &templateContext{
		Patient: patient,
	}

	ag, err := r.DataAPI.PatientAgreements(patient.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ctx.Agreements = ag

	pcp, err := r.DataAPI.GetPatientPCP(patient.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ctx.PCP = pcp

	ec, err := r.DataAPI.GetPatientEmergencyContacts(patient.ID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	ctx.EmergencyContacts = ec

	if patient.IsUnder18() {
		consent, err := r.DataAPI.ParentalConsent(patient.ID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		var con *common.ParentalConsent
		// Find either the first parent if none consented or then one that consented
		for _, c := range consent {
			if con == nil || c.Consented {
				con = c
			}
		}
		if con != nil {
			parent, err := r.DataAPI.Patient(con.ParentPatientID, false)
			if err != nil {
				return nil, errors.Trace(err)
			}
			ctx.Parent = &parentContext{
				Patient: parent,
				Consent: con,
			}
		}
	}

	caseStatuses := append(common.SubmittedPatientCaseStates(), common.PCStatusUnsuitable.String())
	visitStatuses := common.TreatedPatientVisitStates()

	if opt.Has(ROIncludeUnsubmittedVisits) {
		caseStatuses = append(caseStatuses, common.OpenPatientCaseStates()...)
		visitStatuses = append(visitStatuses, common.OpenPatientVisitStates()...)
		visitStatuses = append(visitStatuses, common.SubmittedPatientVisitStates()...)
	}

	cases, err := r.DataAPI.GetCasesForPatient(patient.ID, caseStatuses)
	if err != nil {
		return nil, errors.Trace(err)
	}

	for _, pcase := range cases {
		visits, err := r.DataAPI.GetVisitsForCase(pcase.ID.Int64(), visitStatuses)
		if err != nil {
			return nil, errors.Trace(err)
		}
		careTeam, err := r.DataAPI.GetActiveMembersOfCareTeamForCase(pcase.ID.Int64(), true)
		if err != nil {
			return nil, errors.Trace(err)
		}

		caseCtx := &caseContext{
			Case:     pcase,
			CareTeam: careTeam,
		}
		ctx.Cases = append(ctx.Cases, caseCtx)

		msgs, err := r.DataAPI.ListCaseMessages(pcase.ID.Int64(), api.LCMOIncludePrivate)
		if err != nil {
			return nil, errors.Trace(err)
		}
		if len(msgs) != 0 {
			pars, err := r.DataAPI.CaseMessageParticipants(pcase.ID.Int64(), true)
			if err != nil {
				return nil, errors.Trace(err)
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
						mediaURL, err := r.MediaStore.SignedURL(a.ItemID, r.ExpirationDuration)
						if err != nil {
							return nil, errors.Trace(err)
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
			layout, err := patient_file.VisitReviewLayout(r.DataAPI, patient, r.MediaStore, r.ExpirationDuration, visit, r.APIDomain, r.WebDomain)
			if err != nil {
				return nil, errors.Trace(err)
			}

			visitCtx := &visitContext{
				Visit: visit,
			}

			visitCtx.DiagnosisSet, err = r.DataAPI.ActiveDiagnosisSet(visit.ID.Int64())
			if !api.IsErrNotFound(err) {
				if err != nil {
					return nil, errors.Trace(err)
				}
				codeIDs := make([]string, len(visitCtx.DiagnosisSet.Items))
				for i, d := range visitCtx.DiagnosisSet.Items {
					codeIDs[i] = d.CodeID
				}
				visitCtx.DiagnosisDetails, err = r.DiagnosisSvc.DiagnosisForCodeIDs(codeIDs)
				if err != nil {
					return nil, errors.Trace(err)
				}
			}

			caseCtx.Visits = append(caseCtx.Visits, visitCtx)

			buf := &bytes.Buffer{}
			lr := &intakeLayoutRenderer{
				wr:         buf,
				webDomain:  r.WebDomain,
				patientID:  patient.ID.Int64(),
				mediaStore: r.MediaStore,
				expiration: r.ExpirationDuration,
			}
			if err := lr.render(layout); err != nil {
				return nil, errors.Trace(err)
			}

			visitCtx.IntakeHTML = template.HTML(buf.String())
		}

		treatmentPlans, err := r.DataAPI.GetTreatmentPlansForCase(pcase.ID.Int64())
		if api.IsErrNotFound(err) {
			continue
		} else if err != nil {
			return nil, errors.Trace(err)
		}

		sort.Sort(byStatus(treatmentPlans))

		for _, tp := range treatmentPlans {
			tpCtx := &treatmentPlanContext{
				TreatmentPlan: tp,
			}
			caseCtx.TreatmentPlans = append(caseCtx.TreatmentPlans, tpCtx)

			doctor, err := r.DataAPI.GetDoctorFromID(tp.DoctorID.Int64())
			if err != nil {
				return nil, errors.Trace(err)
			}
			tpCtx.Doctor = doctor

			buf := &bytes.Buffer{}
			if err := treatment_plan.RenderTreatmentPlan(buf, r.DataAPI, tp, doctor, patient); err != nil {
				return nil, errors.Trace(err)
			}
			tpCtx.HTML = template.HTML(buf.String())
		}
	}

	buf := &bytes.Buffer{}

	if err := mrTemplate.Execute(buf, ctx); err != nil {
		return nil, errors.Trace(err)
	}

	return buf.Bytes(), nil
}

type byStatus []*common.TreatmentPlan

func (tp byStatus) Len() int           { return len(tp) }
func (tp byStatus) Swap(i, j int)      { tp[i], tp[j] = tp[j], tp[i] }
func (tp byStatus) Less(i, j int) bool { return tp[i].Status == common.TPStatusActive }

type intakeLayoutRenderer struct {
	wr         *bytes.Buffer
	webDomain  string
	patientID  int64
	mediaStore *media.Store
	expiration time.Duration
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
				mediaURL, err := lr.mediaStore.SignedURL(p.PhotoID, lr.expiration)
				if err != nil {
					return err
				}
				lr.wr.WriteString(fmt.Sprintf(`<div class="col-xs-4"><a href="%s"><img src="%s&width=640"></a></div>`, mediaURL, mediaURL))
			}
			lr.wr.WriteString(`</div>`)
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewStandardSectionView:
		lr.wr.WriteString(`<div class="standard-section">`)
		if v.Title != "" {
			lr.wr.WriteString(`<h3>` + v.Title + `</h3>`)
		}
		for _, s := range v.Subsections {
			if err := lr.renderView(s); err != nil {
				return err
			}
		}
		lr.wr.WriteString(`</div>`)
	case *info_intake.DVisitReviewStandardSubsectionView:
		if len(v.Rows) != 0 {
			lr.wr.WriteString(`<div class="standard-subsection">`)
			if v.Title != "" && v.Title != "Alerts" { // The "Alerts" title gets repeated twice
				lr.wr.WriteString(`<h4>` + v.Title + `</h4>`)
			}
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
