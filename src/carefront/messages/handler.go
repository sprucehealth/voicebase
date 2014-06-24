package messages

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/common"
	"carefront/libs/dispatch"
	"errors"
	"net/http"
)

type handler struct {
	dataAPI api.DataAPI
}

type PostMessageRequest struct {
	CaseID      int64         `json:"case_id,string"`
	Message     string        `json:"message"`
	Attachments []*Attachment `json:"attachments,omitempty"`
}

func (r *PostMessageRequest) Validate() error {
	if r.CaseID <= 0 {
		return errors.New("case_id missing or invalid")
	}
	if r.Message == "" {
		return errors.New("message must not be blank")
	}
	return nil
}

type PostMessageResponse struct {
	MessageID int64  `json:"message_id,string"`
	Result    string `json:"result"`
}

type Attachment struct {
	Type string `json:"type"`
	ID   int64  `json:"id,string"`
	URL  string `json:"url,omitempty"`
}

func NewHandler(dataAPI api.DataAPI) http.Handler {
	return &handler{dataAPI: dataAPI}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != apiservice.HTTP_POST {
		http.NotFound(w, r)
		return
	}

	var req PostMessageRequest
	if err := apiservice.DecodeRequestData(&req, r); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}
	if err := req.Validate(); err != nil {
		apiservice.WriteValidationError(err.Error(), w, r)
		return
	}

	cas, err := h.dataAPI.GetPatientCase(req.CaseID)
	if err == api.NoRowsError {
		apiservice.WriteDeveloperError(w, http.StatusNotFound, "Case with the given ID does not exist")
		return
	}

	personID, doctorID, err := validateAccess(h.dataAPI, r, cas)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	people, err := h.dataAPI.GetPeople([]int64{personID})
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}
	person := people[personID]

	msg := &common.CaseMessage{
		CaseID:   req.CaseID,
		PersonID: personID,
		Body:     req.Message,
	}

	if req.Attachments != nil {
		// Validate all attachments
		for _, att := range req.Attachments {
			switch att.Type {
			default:
				apiservice.WriteValidationError("Unknown attachment type "+att.Type, w, r)
			case common.AttachmentTypeTreatmentPlan:
				// Make sure the treatment plan is a part of the same case
				if apiservice.GetContext(r).Role != api.DOCTOR_ROLE {
					apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Only a doctor is allowed to attach a treatment plan")
					return
				}
				tp, err := h.dataAPI.GetAbridgedTreatmentPlan(att.ID, doctorID)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				if tp.PatientCaseId.Int64() != req.CaseID {
					apiservice.WriteValidationError("Treatment plan does not belong to the case", w, r)
					return
				}
			case common.AttachmentTypePhoto:
				// Make sure the photo is uploaded by the same person and is unclaimed
				photo, err := h.dataAPI.GetPhoto(att.ID)
				if err != nil {
					apiservice.WriteError(err, w, r)
					return
				}
				if photo.UploaderId != personID || photo.ClaimerType != "" {
					apiservice.WriteValidationError("Invalid attachment", w, r)
					return
				}
			}
			msg.Attachments = append(msg.Attachments, &common.CaseMessageAttachment{
				ItemType: att.Type,
				ItemID:   att.ID,
			})
		}
	}

	msgID, err := h.dataAPI.CreateCaseMessage(msg)
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	dispatch.Default.Publish(&PostEvent{
		Message: msg,
		Case:    cas,
		Person:  person,
	})

	res := &PostMessageResponse{
		MessageID: msgID,
		Result:    "success",
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}
