package manager

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

type screenType string

func (s screenType) String() string {
	return string(s)
}

const (
	screenTypeQuestion      screenType = "screen_type_questions"
	screenTypePharmacy      screenType = "screen_type_pharmacy"
	screenTypeTriage        screenType = "screen_type_triage"
	screenTypeWarningPopup  screenType = "screen_type_warning_popup"
	screenTypeMedia         screenType = "screen_type_media"
	screenTypeGenericPopup  screenType = "screen_type_generic_popup"
	screenTypeVisitOverview screenType = "visit_overview"
)

// screenTypeToProtoBufType represents a mapping between the internally
// understood screen types to the protobuf screen types to serialize the
// objects into.
var screenTypeToProtoBufType = map[string]*intake.ScreenData_Type{
	screenTypePharmacy.String():      intake.ScreenData_PHARMACY.Enum(),
	screenTypeMedia.String():         intake.ScreenData_MEDIA.Enum(),
	screenTypeTriage.String():        intake.ScreenData_TRIAGE.Enum(),
	screenTypeQuestion.String():      intake.ScreenData_QUESTION.Enum(),
	screenTypeGenericPopup.String():  intake.ScreenData_GENERIC_POPUP.Enum(),
	screenTypeWarningPopup.String():  intake.ScreenData_IMAGE_POPUP.Enum(),
	screenTypeVisitOverview.String(): intake.ScreenData_VISIT_OVERVIEW.Enum(),
}

func init() {
	mustRegisterScreen(screenTypeQuestion.String(), &questionScreen{})
	mustRegisterScreen(screenTypeMedia.String(), &mediaScreen{})
	mustRegisterScreen(screenTypeTriage.String(), &triageScreen{})
	mustRegisterScreen(screenTypePharmacy.String(), &pharmacyScreen{})
	mustRegisterScreen(screenTypeWarningPopup.String(), &warningPopupScreen{})
	mustRegisterScreen(screenTypeGenericPopup.String(), &genericPopupScreen{})
}

// screenInfo represents the common properties of all screen types.
type screenInfo struct {
	LayoutUnitID string     `json:"-"`
	Title        string     `json:"screen_title"`
	Cond         condition  `json:"condition"`
	Parent       layoutUnit `json:"-"`
	*screenClientData

	v visibility
}

func (s *screenInfo) staticInfoCopy(context map[string]string) interface{} {
	sCopy := &screenInfo{
		Title:  s.Title,
		Parent: s.Parent,
	}

	if s.screenClientData != nil {
		sCopy.screenClientData = s.screenClientData.staticInfoCopy(nil).(*screenClientData)
	}

	if s.Cond != nil {
		sCopy.Cond = s.Cond.staticInfoCopy(nil).(condition)
	}

	return sCopy
}

func (s *screenInfo) title() string {
	return s.Title
}

func (s *screenInfo) layoutParent() layoutUnit {
	return s.Parent
}

func (s *screenInfo) setLayoutParent(node layoutUnit) {
	s.Parent = node
}

func (s *screenInfo) condition() condition {
	return s.Cond
}

func (s *screenInfo) setCondition(cond condition) {
	s.Cond = cond
}

func (s *screenInfo) children() []layoutUnit {
	return nil
}

func (s *screenInfo) setLayoutUnitID(str string) {
	s.LayoutUnitID = str
}

func (s *screenInfo) layoutUnitID() string {
	return s.LayoutUnitID
}

func (s *screenInfo) descriptor() string {
	return screenDescriptor
}

func (s *screenInfo) setVisibility(v visibility) {
	s.v = v
}

func (s *screenInfo) visibility() visibility {
	return s.v
}

// screenClientData represents the common client data properties
// of all screen types.
type screenClientData struct {
	IsTriageScreen       bool   `json:"is_triage_screen"`
	TriagePathwayID      string `json:"pathway_id"`
	TriageParametersJSON []byte `json:"triage_params"`
}

func (s *screenClientData) staticInfoCopy(context map[string]string) interface{} {
	sCopy := &screenClientData{
		IsTriageScreen:       s.IsTriageScreen,
		TriagePathwayID:      s.TriagePathwayID,
		TriageParametersJSON: make([]byte, len(s.TriageParametersJSON)),
	}

	copy(sCopy.TriageParametersJSON, s.TriageParametersJSON)

	return sCopy
}

type button struct {
	Text    string `json:"button_text"`
	Style   string `json:"style"`
	TapLink string `json:"tap_url"`
}

func (b *button) staticInfoCopy(context map[string]string) interface{} {
	return &button{
		Text:    b.Text,
		Style:   b.Style,
		TapLink: b.TapLink,
	}
}

func (b *button) transformToProtobuf() (proto.Message, error) {
	return &intake.Button{
		Text:    proto.String(b.Text),
		TapLink: proto.String(b.TapLink),
	}, nil
}

func populateButton(data dataMap) (*button, error) {

	buttonDataMap, err := data.dataMapForKey("button")
	if err != nil {
		return nil, err
	} else if buttonDataMap == nil {
		buttonDataMap = data
	}

	if err := buttonDataMap.requiredKeys("button", "button_text"); err != nil {
		return nil, err
	}

	return &button{
		Text:    buttonDataMap.mustGetString("button_text"),
		Style:   buttonDataMap.mustGetString("style"),
		TapLink: buttonDataMap.mustGetString("tap_url"),
	}, nil
}

type body struct {
	Text   string  `json:"text"`
	Button *button `json:"button"`
}

func (b *body) staticInfoCopy(context map[string]string) interface{} {
	bCopy := &body{
		Text: b.Text,
	}

	if b.Button != nil {
		bCopy.Button = b.Button.staticInfoCopy(nil).(*button)
	}

	return bCopy
}

func (b *body) transformToProtobuf() (proto.Message, error) {
	var button *intake.Button
	if b.Button != nil {
		transformedB, err := b.Button.transformToProtobuf()
		if err != nil {
			return nil, err
		}
		button = transformedB.(*intake.Button)
	}

	return &intake.Body{
		Text:   proto.String(b.Text),
		Button: button,
	}, nil
}

func populateBody(data dataMap) (*body, error) {

	bodyDataMap, err := data.dataMapForKey("body")
	if err != nil {
		return nil, err
	} else if bodyDataMap == nil {
		return nil, nil
	}

	var btn *button
	if data.exists("button") {
		btn, err = populateButton(bodyDataMap)
		if err != nil {
			return nil, err
		}
	}

	return &body{
		Text:   bodyDataMap.mustGetString("text"),
		Button: btn,
	}, nil
}

func transformScreenInfoToProtobuf(s *screenInfo) (proto.Message, error) {
	screenProtobuf := &intake.CommonScreenInfo{
		Id:    proto.String(s.LayoutUnitID),
		Title: proto.String(s.Title),
	}

	if s.screenClientData != nil {
		screenProtobuf.IsTriageScreen = proto.Bool(s.IsTriageScreen)
		screenProtobuf.TriagePathwayId = proto.String(s.TriagePathwayID)
		screenProtobuf.TriageParametersJson = s.TriageParametersJSON
	}

	return screenProtobuf, nil
}

func populateScreenInfo(data dataMap, parent layoutUnit) (*screenInfo, error) {
	s := &screenInfo{}
	s.Parent = parent
	s.Title = data.mustGetString("screen_title")

	if !data.exists("condition") {
		return s, nil
	}

	conditionDataMap, err := data.dataMapForKey("condition")
	if err != nil {
		return nil, err
	} else if conditionDataMap != nil {
		s.Cond, err = getCondition(conditionDataMap)
		if err != nil {
			return nil, err
		}
	}

	s.screenClientData, err = popuplateClientData(data)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func popuplateClientData(data dataMap) (*screenClientData, error) {
	clientData, err := data.dataMapForKey("client_data")
	if err != nil {
		return nil, err
	} else if clientData == nil {
		return &screenClientData{}, nil
	}

	triageParametersJSON, err := clientData.getJSONData("triage_params")
	if err != nil {
		return nil, err
	}

	return &screenClientData{
		IsTriageScreen:       clientData.mustGetBool("is_triage_screen"),
		TriagePathwayID:      clientData.mustGetString("pathway_id"),
		TriageParametersJSON: triageParametersJSON,
	}, nil
}

func (s *screenInfo) requirementsMet(dataSource questionAnswerDataSource) (bool, error) {
	if s.IsTriageScreen && s.visibility() == visible {
		return false, nil
	}
	return true, nil
}

// wrapScreen wraps the provided screen object into an object that
// explicitly states its type to know which data type to deserialize
// the data into on the receiving end.
func wrapScreen(s screen, progress *float32, serializerLib serializer) ([]byte, error) {
	transformedScreen, err := s.transformToProtobuf()
	if err != nil {
		return nil, err
	}

	data, err := serializerLib.marshal(transformedScreen)
	if err != nil {
		return nil, err
	}

	return serializerLib.marshal(&intake.ScreenData{
		Type:     screenTypeToProtoBufType[s.TypeName()],
		Data:     data,
		Progress: progress,
	})
}

func wrapScreenID(id, typeName string, serializerLib serializer) ([]byte, error) {
	return serializerLib.marshal(&intake.ScreenIDData{
		Type: screenTypeToProtoBufType[typeName],
		Id:   proto.String(id),
	})
}

// validateScreen wraps any validation errors in a custom object to be sent to the client to parse out
// and appropriately display validation errors. If there are no validation errors, then that is communicated
// via the data object as well.
func validateScreen(s screen, dataSource questionAnswerDataSource, serializerLib serializer) ([]byte, error) {
	var res intake.ValidateRequirementsResult

	requirementsMet, err := s.requirementsMet(dataSource)
	// ignore subquestions not having requirements met when just checking
	// for screen validation to proceed to next screen.
	if err == errSubQuestionRequirements || err == nil {
		res.Status = intake.ValidateRequirementsResult_OK.Enum()
	} else if err != nil {
		res.Status = intake.ValidateRequirementsResult_ERROR.Enum()
		res.Message = proto.String(err.Error())
	} else if !requirementsMet {
		res.Status = intake.ValidateRequirementsResult_ERROR.Enum()
	}

	return serializerLib.marshal(&res)
}

func (s *screenInfo) stringIndent(indent string, depth int) string {
	return fmt.Sprintf("%s%s: %s | %s", indentAtDepth(indent, depth), s.layoutUnitID(), s.Title, s.v)
}
