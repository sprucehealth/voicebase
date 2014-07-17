package treatment_plan

import (
	"bytes"
	"html/template"
	"io"
	"strings"

	"github.com/sprucehealth/backend/common"
)

var templateFuncMap = map[string]interface{}{
	"renderView": func(view tpView) (template.HTML, error) {
		buf := &bytes.Buffer{}
		if err := rxGuideTemplate.ExecuteTemplate(buf, view.TypeName(), view); err != nil {
			return "", err
		}
		return template.HTML(buf.String()), nil
	},
	"mapImageURL": func(url string) (string, error) {
		if strings.HasPrefix(url, "spruce:///image/") {
			// TODO: this URL should be taken from the config
			return "https://carefront-static.s3.amazonaws.com/" + url[16:], nil
		}
		return url, nil
	},
}

const templateText = `
{{define "base"}}
	<div class="treatment-plan">
		{{range .Views}}
			{{renderView .}}
		{{end}}
	</div>
{{end}}

{{define "treatment:image"}}
	{{if .ImageURL}}
		<img src="{{mapImageURL .ImageURL}}"> <!-- width="{{.ImageWidth}}" height="{{.ImageHeight}}" -->
	{{end}}
{{end}}

{{define "treatment:small_divider"}}
	<hr class="small">
{{end}}

{{define "treatment:large_divider"}}
	<div class="large-divider-view">&nbsp;</div>
{{end}}

{{define "treatment:list_element"}}
	<div class="list-element content-view">
		{{if eq .ElementStyle "numbered"}}
			<div style="float:left; width:20px; text-align:right;">{{.Number}}.</div><div style="margin-left:25px;">{{.Text}}</div>
		{{else}}
			<div style="float:left; width:15px; text-align:center;">●</div><div style="margin-left:20px;">{{.Text}}</div>
		{{end}}
	</div>
{{end}}

{{define "treatment:icon_title_subtitle_view"}}
	<div class="icon-title-subtitle-view content-view">
		{{if .IconURL}}<img src="{{mapImageURL .IconURL.String}}" width="32" height="32">{{end}}
		<h2>{{.Title}}</h2>
		<h3>{{.Subtitle}}</h3>
	</div>
{{end}}

{{define "treatment:icon_text_view"}}
	<div class="content-view {{.Style}}">
		{{if .IconURL}}<img src="{{mapImageURL .IconURL.String}}" width="{{.IconWidth}}" height="{{.IconHeight}}">{{end}}
		<span class="{{.TextStyle}}">{{.Text}}</span>
	</div>
{{end}}

{{define "treatment:text"}}
	<div class="text-view content-view text-view-style-{{.Style}}">
		{{.Text}}
	</div>
{{end}}

{{define "treatment:button"}}
	<div class="button-view content-view">
		<a href="{{.TapURL}}"><img src="{{mapImageURL .IconURL.String}}"> {{.Text}}</a>
	</div>
{{end}}
`

var rxGuideTemplate *template.Template

func init() {
	rxGuideTemplate = template.Must(template.New("").Funcs(templateFuncMap).Parse(templateText))
}

type rxGuideTemplateContext struct {
	Views []tpView
}

func RenderRXGuide(w io.Writer, details *common.DrugDetails, treatment *common.Treatment, treatmentPlan *common.TreatmentPlan) error {
	views, err := treatmentGuideViews(details, treatment, treatmentPlan)
	if err != nil {
		return err
	}
	return rxGuideTemplate.ExecuteTemplate(w, "base", &rxGuideTemplateContext{Views: views})
}
