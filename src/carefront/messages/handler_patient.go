package messages

import (
	"carefront/api"
	"carefront/apiservice"
	"carefront/libs/dispatch"
	"encoding/json"
	"net/http"

	"github.com/gorilla/schema"
)

type PatientConversationHandler struct {
	dataAPI api.DataAPI
}

type PatientMessagesHandler struct {
	dataAPI api.DataAPI
}

type PatientReadHandler struct {
	dataAPI api.DataAPI
}

func NewPatientConversationHandler(dataAPI api.DataAPI) *PatientConversationHandler {
	return &PatientConversationHandler{
		dataAPI: dataAPI,
	}
}

func (h *PatientConversationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patientId, err := h.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get patient: "+err.Error())
		return
	}
	personId, err := h.dataAPI.GetPersonIdByRole(api.PATIENT_ROLE, patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get person object for patient: "+err.Error())
		return
	}

	switch r.Method {
	case apiservice.HTTP_GET:
		h.listConversations(w, r, patientId, personId)
	case apiservice.HTTP_POST:
		h.newConversation(w, r, patientId, personId)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (h *PatientConversationHandler) listConversations(w http.ResponseWriter, r *http.Request, patientId, personId int64) {
	con, par, err := h.dataAPI.GetConversationsWithParticipants([]int64{personId})
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get conversations: "+err.Error())
		return
	}
	res := &ConversationListResponse{
		Conversations: conversationsToConversationList(con, personId),
		Participants:  peopleToParticipants(par),
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}

func (h *PatientConversationHandler) newConversation(w http.ResponseWriter, r *http.Request, patientId, personId int64) {
	req := &NewConversationRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	// TODO: for now always assume the patient is sending a message to their primary doctor

	careTeam, err := h.dataAPI.GetCareTeamForPatient(patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get care team based on patient id: "+err.Error())
		return
	}

	doctorId := apiservice.GetPrimaryDoctorIdFromCareTeam(careTeam)
	if doctorId == 0 {
		apiservice.WriteUserError(w, http.StatusBadRequest, "No primary doctor assigned")
		return
	}
	doctorPersonId, err := h.dataAPI.GetPersonIdByRole(api.DOCTOR_ROLE, doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get person object for doctor: "+err.Error())
		return
	}

	attachments, err := parseAttachments(h.dataAPI, req.Attachments, personId)
	if err == api.NoRowsError {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unknown photo")
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get attachments: "+err.Error())
		return
	}
	cid, err := h.dataAPI.CreateConversation(personId, doctorPersonId, req.TopicId, req.Message, attachments)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to create conversation: "+err.Error())
		return
	}

	dispatch.Default.PublishAsync(&ConversationStartedEvent{
		ConversationId: cid,
		TopicId:        req.TopicId,
		FromId:         personId,
		ToId:           doctorPersonId,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, &NewConversationResponse{ConversationId: cid})
}

func NewPatientMessagesHandler(dataAPI api.DataAPI) *PatientMessagesHandler {
	return &PatientMessagesHandler{
		dataAPI: dataAPI,
	}
}

func (h *PatientMessagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patientId, err := h.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get patient: "+err.Error())
		return
	}
	personId, err := h.dataAPI.GetPersonIdByRole(api.PATIENT_ROLE, patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get person object for patient: "+err.Error())
		return
	}

	switch r.Method {
	case apiservice.HTTP_GET:
		h.listMessages(w, r, patientId, personId)
	case apiservice.HTTP_POST:
		h.postMessage(w, r, patientId, personId)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (h *PatientMessagesHandler) listMessages(w http.ResponseWriter, r *http.Request, patientId, personId int64) {
	if err := r.ParseForm(); err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var req conversationRequest
	if err := schema.NewDecoder().Decode(&req, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	// Verify patient can read the conversation
	if ok, err := isPersonAParticipant(h.dataAPI, req.ConversationId, personId); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Unable to get access info: "+err.Error())
		return
	} else if !ok {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Not allowed")
		return
	}

	con, err := h.dataAPI.GetConversation(req.ConversationId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Failed to get conversation: "+err.Error())
		return
	}

	res := &conversationResponse{
		Id:           req.ConversationId,
		Title:        con.Title,
		Items:        messageList(con.Messages, r),
		Participants: peopleToParticipants(con.Participants),
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}

func (h *PatientMessagesHandler) postMessage(w http.ResponseWriter, r *http.Request, patientId, personId int64) {
	req := &ReplyRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	if ok, err := isPersonAParticipant(h.dataAPI, req.ConversationId, personId); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to get conversation participants: "+err.Error())
		return
	} else if !ok {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Person is not a participant in the conversation")
		return
	}

	attachments, err := parseAttachments(h.dataAPI, req.Attachments, personId)
	if err == api.NoRowsError {
		apiservice.WriteUserError(w, http.StatusInternalServerError, "Unknown photo")
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get attachments: "+err.Error())
		return
	}
	mid, err := h.dataAPI.ReplyToConversation(req.ConversationId, personId, req.Message, attachments)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to create reply to conversation: "+err.Error())
		return
	}

	dispatch.Default.PublishAsync(&ConversationReplyEvent{
		ConversationId: req.ConversationId,
		MessageId:      mid,
		FromId:         personId,
	})

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}

func NewPatientReadHandler(dataAPI api.DataAPI) *PatientReadHandler {
	return &PatientReadHandler{
		dataAPI: dataAPI,
	}
}

func (h *PatientReadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patientId, err := h.dataAPI.GetPatientIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get patient: "+err.Error())
		return
	}
	personId, err := h.dataAPI.GetPersonIdByRole(api.PATIENT_ROLE, patientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get person object for patient: "+err.Error())
		return
	}
	markConversationAsRead(w, r, h.dataAPI, personId)
}
