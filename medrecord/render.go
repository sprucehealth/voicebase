package medrecord

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/sig"
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
	Visit      *common.PatientVisit
	Diagnosis  string
	IntakeHTML template.HTML
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

		{{with .Patient.Pharmacy}}
		<div class="pharmacy">
			<h4>Pharmacy</h4>
			<div>
				{{.Name}}<br>
				{{.AddressLine1}}<br>
				{{with .AddressLine2}}{{.}}<br>{{end}}
				Phone: {{.Phone}}<br>
				{{.City}}, {{.State}}<br>
				{{.Postal}}<br>
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

			{{range .Visits}}
				<div class="visit">
					{{with .Diagnosis}}
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
		{{end}}
	</div>
</body>
</html>`))
