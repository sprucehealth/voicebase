package admin

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/doctor_treatment_plan"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/pharmacy"
	"github.com/sprucehealth/backend/treatment_plan"
	"github.com/sprucehealth/backend/views"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/libs/erx"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/www"
)

var (
	NoteRXDescription          = regexp.MustCompile(`note:rx_\d+_description`)
	NoteAdditionalInfo         = regexp.MustCompile(`note:additional_info_\d+`)
	ScheduledMessageDuration   = regexp.MustCompile(`scheduled_message_\d+_duration`)
	ScheduledMessageUnit       = regexp.MustCompile(`scheduled_message_\d+_unit`)
	ScheduledMessageAttachment = regexp.MustCompile(`scheduled_message_\d+_attachment`)
	ScheduledMessage           = regexp.MustCompile(`scheduled_message_\d+`)
	RXName                     = regexp.MustCompile(`rx_\d+_name`)
	RXDosage                   = regexp.MustCompile(`rx_\d+_dosage`)
	RXDispenseType             = regexp.MustCompile(`rx_\d+_dispense_type`)
	RXDispenseNumber           = regexp.MustCompile(`rx_\d+_dispense_number`)
	RXRefills                  = regexp.MustCompile(`rx_\d+_refills`)
	RXSubstitutions            = regexp.MustCompile(`rx_\d+_substitutions`)
	RXSig                      = regexp.MustCompile(`rx_\d+_sig`)
	RXRoute                    = regexp.MustCompile(`rx_\d+_route`)
	RXForm                     = regexp.MustCompile(`rx_\d+_form`)
	RXGenericName              = regexp.MustCompile(`rx_\d+_generic_name`)
	SectionTitle               = regexp.MustCompile(`section_\d+_title`)
	SectionStep                = regexp.MustCompile(`section_\d+_step_\d+`)
	ResourceGuide              = regexp.MustCompile(`resource_guide_\d+`)
	Digits                     = regexp.MustCompile(`\d+`)
)

type ftp struct {
	FrameworkTag      string
	FrameworkName     string
	SFTPName          string
	Diagnosis         string
	Note              note
	ScheduledMessages map[string]scheduledMessage
	Sections          map[string]section
	RXs               map[string]rx
	ResourceGuideTags []string
	IsSTP             bool
}

type note struct {
	Welcome              string
	ConditionDescription string
	MDRecommendation     string
	RXDescriptions       map[string]string
	AssitionalInfo       map[string]string
	Closing              string
}

func (n note) String() string {
	var rxDescription string
	for _, v := range n.RXDescriptions {
		rxDescription += v + "\n\n"
	}
	var additionalInfo string
	for _, v := range n.AssitionalInfo {
		additionalInfo += v + "\n\n"
	}
	return fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s%s%s\n", n.Welcome, n.ConditionDescription, n.MDRecommendation, rxDescription, additionalInfo, n.Closing)
}

type rx struct {
	Name           string
	Dosage         string
	DispenseType   string
	DispenseNumber string
	Refills        string
	Substitutions  string
	Sig            string
}

type scheduledMessage struct {
	Duration   string
	Unit       string
	Message    string
	Attachment string
}

func (sm scheduledMessage) DurationInDays() (int, error) {
	multiplier := 1
	if sm.Unit == "weeks" {
		multiplier = 7
	} else if sm.Unit == "months" {
		multiplier = 30
	} else if sm.Unit == "years" {
		multiplier = 365
	}
	dur, err := strconv.ParseInt(sm.Duration, 10, 64)
	if err != nil {
		return 0, err
	}
	return (int(dur) * multiplier), nil
}

func (sm scheduledMessage) RequiresFollowup() bool {
	return strings.ToLower(sm.Attachment) == "yes" || strings.ToLower(sm.Attachment) == "true"
}

type section struct {
	Title string
	Steps []step
}

type step struct {
	Text  string
	Order int64
}

type treatmentPlanCSVHandler struct {
	dataAPI api.DataAPI
	erxAPI  erx.ERxAPI
}

type treatmentPlanCSVPUTRequest struct {
	ColData [][]string
}

func newFTP() *ftp {
	return &ftp{
		Note: note{
			RXDescriptions: make(map[string]string),
			AssitionalInfo: make(map[string]string),
		},
		ScheduledMessages: make(map[string]scheduledMessage),
		Sections:          make(map[string]section),
		RXs:               make(map[string]rx),
		ResourceGuideTags: make([]string, 0),
	}
}

func NewTreatmentPlanCSVHandler(dataAPI api.DataAPI, erxAPI erx.ERxAPI) http.Handler {
	return httputil.SupportedMethods(&treatmentPlanCSVHandler{dataAPI: dataAPI, erxAPI: erxAPI}, []string{"PUT"})
}

func (h *treatmentPlanCSVHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		requestData, err := h.parsePUTRequest(r)
		if err != nil {
			www.APIBadRequestError(w, r, err.Error())
			return
		}
		h.servePUT(w, r, requestData)
	}
}

func (h *treatmentPlanCSVHandler) parsePUTRequest(r *http.Request) (*treatmentPlanCSVPUTRequest, error) {
	var err error
	rd := &treatmentPlanCSVPUTRequest{}
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return nil, err
	}
	f, _, err := r.FormFile("csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rd.ColData, err = csvDataFromFile(f)
	if err != nil {
		return nil, err
	}

	return rd, nil
}

func (h *treatmentPlanCSVHandler) servePUT(w http.ResponseWriter, r *http.Request, req *treatmentPlanCSVPUTRequest) {
	threads := len(req.ColData)
	ftps, err := parseFTPs(req.ColData, threads)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	for _, v := range ftps {
		if v.FrameworkTag == "" {
			www.APIBadRequestError(w, r, fmt.Sprintf("Empty framework_tag detected. Cannot complete request"))
			return
		}
	}

	err = h.createGlobalFTPs(ftps)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}

	_, err = h.transformFTPsToSTPs(ftps, threads)
	if err != nil {
		www.APIInternalError(w, r, err)
		return
	}
}

type completedSTP struct {
	PathwayTag string
	STPJSON    []byte
}

func (h *treatmentPlanCSVHandler) transformFTPsToSTPs(ftps []*ftp, threads int) (map[string][]byte, error) {
	errs := make(chan error, len(ftps))
	complete := make(chan *completedSTP, len(ftps))
	stps := make(map[string][]byte)
	done := 0
	started := 0
	for i := 0; i < threads && i < len(ftps); i++ {
		go h.transformFTPToSTP(*ftps[started], complete, errs)
		started++
	}
	for done != len(ftps) {
		select {
		case stp := <-complete:
			stps[stp.PathwayTag] = stp.STPJSON
			done++
		case err := <-errs:
			return nil, err
		}
		if started-done < threads && started < len(ftps) {
			go h.transformFTPToSTP(*ftps[started], complete, errs)
			started++
		}
	}

	return stps, nil
}

func (h *treatmentPlanCSVHandler) createGlobalFTPs(ftps []*ftp) error {
	// For now default this to be language_id EN
	dispenseUnitIDs, dispenseUnits, err := h.dataAPI.GetMedicationDispenseUnits(1)
	if err != nil {
		return err
	}
	dispenseUnitIDMapping := make(map[string]int64)
	for i, dispenseUnit := range dispenseUnits {
		dispenseUnitIDMapping[dispenseUnit] = dispenseUnitIDs[i]
	}
	ftpModels := make(map[int64][]*common.FavoriteTreatmentPlan)
	for _, ftp := range ftps {
		regimineSections := make([]*common.RegimenSection, 0)
		sectionKeys := make([]string, 0, len(ftp.Sections))
		for k, _ := range ftp.Sections {
			sectionKeys = append(sectionKeys, k)
		}
		sort.Strings(sectionKeys)
		for _, k := range sectionKeys {
			steps := make([]*common.DoctorInstructionItem, 0)
			for _, st := range ftp.Sections[k].Steps {
				steps = append(steps, &common.DoctorInstructionItem{Text: st.Text})
			}
			regimineSection := &common.RegimenSection{
				Name:  ftp.Sections[k].Title,
				Steps: steps,
			}
			regimineSections = append(regimineSections, regimineSection)
		}
		regiminePlan := &common.RegimenPlan{
			Sections: regimineSections,
			Title:    ftp.SFTPName,
			Status:   "ACTIVE",
		}

		treatmentList := &common.TreatmentList{}
		rxKeys := make([]string, 0, len(ftp.Sections))
		for k, _ := range ftp.RXs {
			rxKeys = append(rxKeys, k)
		}
		sort.Strings(rxKeys)
		for _, k := range rxKeys {
			msr, err := h.erxAPI.SelectMedication(0, ftp.RXs[k].Name, ftp.RXs[k].Dosage)
			if err != nil {
				return err
			}
			if msr != nil {
				treatment, _ := doctor_treatment_plan.CreateTreatmentFromMedication(msr, ftp.RXs[k].Dosage, ftp.RXs[k].Name)
				numberRefills := encoding.NullInt64{}
				numberRefills.Int64Value, err = strconv.ParseInt(ftp.RXs[k].Refills, 10, 64)
				if err != nil {
					return err
				}
				dispenseValue, err := strconv.ParseFloat(ftp.RXs[k].DispenseNumber, 64)
				if err != nil {
					return err
				}
				dispenseUnitID, ok := dispenseUnitIDMapping[ftp.RXs[k].DispenseType]
				if !ok {
					return fmt.Errorf("No dispense unit ID could be located for type %s", ftp.RXs[k].DispenseType)
				}
				treatment.NumberRefills = numberRefills
				treatment.DispenseValue = encoding.HighPrecisionFloat64(dispenseValue)
				treatment.DispenseUnitID = encoding.NewObjectID(dispenseUnitID)
				treatment.SubstitutionsAllowed = strings.ToLower(ftp.RXs[k].Substitutions) == "yes" || strings.ToLower(ftp.RXs[k].Substitutions) == "true"
				treatment.PatientInstructions = ftp.RXs[k].Sig
				treatmentList.Treatments = append(treatmentList.Treatments, treatment)
				treatmentList.Status = "ACTIVE"
			}
		}

		scheduledMessages := make([]*common.TreatmentPlanScheduledMessage, 0)
		scheduledMessagesKeys := make([]string, 0, len(ftp.Sections))
		for k, _ := range ftp.ScheduledMessages {
			scheduledMessagesKeys = append(scheduledMessagesKeys, k)
		}
		sort.Strings(scheduledMessagesKeys)
		for _, k := range scheduledMessagesKeys {
			dur, err := ftp.ScheduledMessages[k].DurationInDays()
			if err != nil {
				return err
			}
			attachments := make([]*common.CaseMessageAttachment, 0)
			if ftp.ScheduledMessages[k].RequiresFollowup() {
				attachments = append(attachments, &common.CaseMessageAttachment{
					ItemType: "followup_visit",
					Title:    "Follow-Up Visit",
				})
			}
			scheduledMessages = append(scheduledMessages, &common.TreatmentPlanScheduledMessage{
				ScheduledDays: dur,
				Message:       ftp.ScheduledMessages[k].Message,
				Attachments:   attachments,
			})
		}

		resourceGuides := make([]*common.ResourceGuide, len(ftp.ResourceGuideTags))
		for i, rgt := range ftp.ResourceGuideTags {
			guide, err := h.dataAPI.GetResourceGuideFromTag(rgt)
			if err != nil {
				return err
			}
			resourceGuides[i] = guide
		}

		ftpModel := &common.FavoriteTreatmentPlan{
			Name:              ftp.SFTPName,
			Note:              ftp.Note.String(),
			RegimenPlan:       regiminePlan,
			TreatmentList:     treatmentList,
			ScheduledMessages: scheduledMessages,
			ResourceGuides:    resourceGuides,
			Lifecycle:         "ACTIVE",
		}

		pathway, err := h.dataAPI.PathwayForTag(ftp.FrameworkTag, api.PONone)
		if err != nil {
			return err
		}

		list, ok := ftpModels[pathway.ID]
		if !ok {
			list = make([]*common.FavoriteTreatmentPlan, 0)
		}
		list = append(list, ftpModel)
		ftpModels[pathway.ID] = list
	}
	if err := h.dataAPI.InsertGlobalFTPsAndUpdateMemberships(ftpModels); err != nil {
		return err
	}
	return nil
}

func (h *treatmentPlanCSVHandler) transformFTPToSTP(ftp ftp, complete chan *completedSTP, errs chan error) {
	sftp := &treatment_plan.TreatmentPlanViewsResponse{}
	sftp.HeaderViews = []views.View{
		treatment_plan.NewTPHeroHeaderView("Sample Treatment Plan", "Your doctor will personalize a treatment plan for you."),
	}

	instruction_views := make([]views.View, len(ftp.Sections)+1)
	instruction_views[0] = treatment_plan.NewTPCardView([]views.View{
		treatment_plan.NewTPTextView("title1_medium", "Your doctor will explain how to use your treatments together in a personalized care routine."),
	})
	for _, v := range sftp.HeaderViews {
		v.Validate("treatment")
	}

	sectionKeys := make([]string, 0, len(ftp.Sections))
	for k, _ := range ftp.Sections {
		sectionKeys = append(sectionKeys, k)
	}
	sort.Strings(sectionKeys)
	sectionIndex := 1
	for _, k := range sectionKeys {
		section_instruction_views := make([]views.View, len(ftp.Sections[k].Steps)+1)
		section_instruction_views[0] = treatment_plan.NewTPCardTitleView(ftp.Sections[k].Title, "", false)
		for si, st := range ftp.Sections[k].Steps {
			section_instruction_views[si+1] = treatment_plan.NewTPListElement("bulleted", st.Text, si)
		}
		instruction_views[sectionIndex] = treatment_plan.NewTPCardView(section_instruction_views)
		sectionIndex++
	}
	sftp.InstructionViews = instruction_views
	for _, v := range sftp.InstructionViews {
		v.Validate("treatment")
	}

	treatment_views := make([]views.View, 0, len(ftp.RXs)+2)
	treatment_views = append(treatment_views, treatment_plan.NewTPCardView([]views.View{
		treatment_plan.NewTPTextView("title1_medium", "Your doctor will determine the right treatments for you."),
		treatment_plan.NewTPTextView("", "Prescriptions will be available to pick up at your preferred pharmacy."),
	}))
	treatment_list := &common.TreatmentList{}

	rxKeys := make([]string, 0, len(ftp.Sections))
	for k, _ := range ftp.RXs {
		rxKeys = append(rxKeys, k)
	}
	sort.Strings(rxKeys)
	for _, k := range rxKeys {
		msr, err := h.erxAPI.SelectMedication(0, ftp.RXs[k].Name, ftp.RXs[k].Dosage)
		if err != nil {
			errs <- err
			return
		}
		if msr != nil {
			treatment, _ := doctor_treatment_plan.CreateTreatmentFromMedication(msr, ftp.RXs[k].Dosage, ftp.RXs[k].Name)
			treatment.PatientInstructions = ftp.RXs[k].Sig
			treatment_list.Treatments = append(treatment_list.Treatments, treatment)
			treatment_list.Status = "ACTIVE"
		}
	}
	if len(treatment_list.Treatments) > 0 {
		treatment_views = append(treatment_views, treatment_plan.GenerateViewsForTreatments(treatment_list, 0, h.dataAPI, false)...)
	}
	treatment_views = append(treatment_views, treatment_plan.NewTPCardView([]views.View{
		treatment_plan.NewTPCardTitleView("Prescription Pickup", "", false),
		treatment_plan.NewPharmacyView("Your prescriptions should be ready soon. Call your pharmacy to confirm a pickup time.", nil,
			&pharmacy.PharmacyData{
				AddressLine1: "1101 Market St",
				City:         "San Francisco",
				SourceID:     8561,
				Latitude:     37.77959,
				Longitude:    -122.41363,
				Name:         "Cvs/Pharmacy",
				Phone:        "4155581538",
				Source:       "surescripts",
				State:        "CA",
				Url:          "",
				Postal:       "94103",
			}),
	}))
	sftp.TreatmentViews = treatment_views
	for _, v := range sftp.TreatmentViews {
		v.Validate("treatment")
	}

	jsonData, err := json.Marshal(sftp)
	if err != nil {
		errs <- err
		return
	}

	if ftp.IsSTP {
		if err := h.dataAPI.CreatePathwaySTP(ftp.FrameworkTag, jsonData); err != nil {
			errs <- err
			return
		}
	}

	complete <- &completedSTP{
		PathwayTag: ftp.FrameworkTag,
		STPJSON:    jsonData,
	}
}

func parseFTPs(colData [][]string, threads int) ([]*ftp, error) {
	ftpCount := len(colData[0]) - 1
	iterationThreads := threads
	errs := make(chan error, ftpCount)
	complete := make(chan *ftp, ftpCount)
	ftps := make([]*ftp, ftpCount)

	// This is sub optimal threading here in the sense that any thread iteration group less than the number of FTPs is only a fast as the slowest thread
	for i := 0; i < ftpCount; i = i + iterationThreads {
		if ftpCount-i < threads {
			iterationThreads = ftpCount - i
		}
		for it := 0; it < iterationThreads; it++ {
			go parseFTP(colData, i+it+1, errs, complete)
		}
		completed := 0
		for completed != iterationThreads {
			select {
			case ftp, ok := <-complete:
				if !ok {
					return nil, errors.New("Something went wrong. Channel closed prematurely")
				}
				ftps[i+completed] = ftp
				completed++
			case err := <-errs:
				return nil, err
			}
		}
	}

	return ftps, nil
}

func parseFTP(colData [][]string, column int, errs chan error, complete chan *ftp) {
	ftp := newFTP()
	// This makes some assumptions that items that are numbered exist in series
	for i, data := range colData {
		t, d := data[0], data[column]
		t, d = strings.TrimSpace(t), strings.TrimSpace(d)
		switch {
		case "" == t:
		case "framework_name" == t:
			ftp.FrameworkName = d
		case "framework_tag" == t:
			ftp.FrameworkTag = d
		case "sftp_name" == t:
			ftp.SFTPName = d
		case "diagnosis" == t:
			ftp.Diagnosis = d
		case "note:welcome" == t:
			ftp.Note.Welcome = d
		case "note:condition_description" == t:
			ftp.Note.ConditionDescription = d
		case "note:md_recommendation" == t:
			ftp.Note.MDRecommendation = d
		case NoteRXDescription.MatchString(t):
			if d != "" {
				ftp.Note.RXDescriptions[Digits.FindString(t)] = d
			}
		case NoteAdditionalInfo.MatchString(t):
			if d != "" {
				ftp.Note.AssitionalInfo[Digits.FindString(t)] = d
			}
		case "note:closing" == t:
			ftp.Note.Closing = d
		case ScheduledMessageDuration.MatchString(t):
			if d != "" {
				si := Digits.FindString(t)
				sm, ok := ftp.ScheduledMessages[si]
				if !ok {
					sm = scheduledMessage{}
				}
				sm.Duration = d
				ftp.ScheduledMessages[si] = sm
			}
		case ScheduledMessageUnit.MatchString(t):
			if d != "" {
				si := Digits.FindString(t)
				sm, ok := ftp.ScheduledMessages[si]
				if !ok {
					sm = scheduledMessage{}
				}
				sm.Unit = d
				ftp.ScheduledMessages[si] = sm
			}
		case ScheduledMessageAttachment.MatchString(t):
			if d != "" {
				si := Digits.FindString(t)
				sm, ok := ftp.ScheduledMessages[si]
				if !ok {
					sm = scheduledMessage{}
				}
				sm.Attachment = d
				ftp.ScheduledMessages[si] = sm
			}
		case ScheduledMessage.MatchString(t):
			if d != "" {
				si := Digits.FindString(t)
				sm, ok := ftp.ScheduledMessages[si]
				if !ok {
					sm = scheduledMessage{}
				}
				sm.Message = d
				ftp.ScheduledMessages[si] = sm
			}
		case RXDosage.MatchString(t):
			if d != "" {
				ri := Digits.FindString(t)
				r, ok := ftp.RXs[ri]
				if !ok {
					r = rx{}
				}
				r.Dosage = d
				ftp.RXs[ri] = r
			}
		case RXName.MatchString(t):
			if d != "" {
				ri := Digits.FindString(t)
				r, ok := ftp.RXs[ri]
				if !ok {
					r = rx{}
				}
				r.Name = d
				ftp.RXs[ri] = r
			}
		case RXRefills.MatchString(t):
			if d != "" {
				ri := Digits.FindString(t)
				r, ok := ftp.RXs[ri]
				if !ok {
					r = rx{}
				}
				r.Refills = d
				ftp.RXs[ri] = r
			}
		case RXDispenseNumber.MatchString(t):
			if d != "" {
				ri := Digits.FindString(t)
				r, ok := ftp.RXs[ri]
				if !ok {
					r = rx{}
				}
				r.DispenseNumber = d
				ftp.RXs[ri] = r
			}
		case RXDispenseType.MatchString(t):
			if d != "" {
				ri := Digits.FindString(t)
				r, ok := ftp.RXs[ri]
				if !ok {
					r = rx{}
				}
				r.DispenseType = d
				ftp.RXs[ri] = r
			}
		case RXSubstitutions.MatchString(t):
			if d != "" {
				ri := Digits.FindString(t)
				r, ok := ftp.RXs[ri]
				if !ok {
					r = rx{}
				}
				r.Substitutions = d
				ftp.RXs[ri] = r
			}
		case RXSig.MatchString(t):
			if d != "" {
				ri := Digits.FindString(t)
				r, ok := ftp.RXs[ri]
				if !ok {
					r = rx{}
				}
				r.Sig = d
				ftp.RXs[ri] = r
			}
		case SectionTitle.MatchString(t):
			if d != "" {
				si := Digits.FindString(t)
				s, ok := ftp.Sections[si]
				if !ok {
					s = section{Steps: make([]step, 0)}
				}
				s.Title = d
				ftp.Sections[si] = s
			}
		case SectionStep.MatchString(t):
			if d != "" {
				si := Digits.FindAllString(t, 2)
				if len(si) != 2 {
					errs <- fmt.Errorf("Expected to find 2 digits in section step type `%s` but found %d", t, len(si))
					return
				}
				s, ok := ftp.Sections[si[0]]
				if !ok {
					s = section{Steps: make([]step, 0)}
				}
				order, err := strconv.ParseInt(si[1], 10, 64)
				if err != nil {
					errs <- fmt.Errorf("Expected to find 2 digits in section step type `%s` but found %d", t, len(si))
					return
				}
				s.Steps = append(s.Steps, step{Text: d, Order: order})
				ftp.Sections[si[0]] = s
			}
		case ResourceGuide.MatchString(t):
			if d != "" {
				ftp.ResourceGuideTags = append(ftp.ResourceGuideTags, d)
			}
		case "sample_ftp" == t:
			ftp.IsSTP = (strings.ToLower(d) == "true" || strings.ToLower(d) == "yes")
		default:
			errs <- fmt.Errorf("Unable to identify row type '%s' in row %d", data[0], i)
			return
		}
	}
	complete <- ftp
}

func csvDataFromFile(f multipart.File) ([][]string, error) {
	reader := csv.NewReader(f)
	data, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	return data, nil
}
