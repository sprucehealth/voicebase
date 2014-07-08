package admin

import (
	"html/template"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/www"
)

var (
	baseTemplate         *template.Template
	doctorSearchTemplate *template.Template
	doctorTemplate       *template.Template
)

func init() {
	baseTemplate = www.MustLoadTemplate("admin/base.html", template.Must(www.BaseTemplate.Clone()))
	doctorSearchTemplate = www.MustLoadTemplate("admin/doctor_search.html", template.Must(baseTemplate.Clone()))
	doctorTemplate = www.MustLoadTemplate("admin/doctor.html", template.Must(baseTemplate.Clone()))
}

type doctorSearchTemplateContext struct {
	Query   string
	Doctors []*common.DoctorSearchResult
}

type doctorTemplateContext struct {
	Doctor          *common.Doctor
	Attributes      map[string]template.HTML
	MedicalLicenses []*common.MedicalLicense
}
