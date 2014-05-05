package messages

import (
	"carefront/api"
	"carefront/apiservice"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
)

type DoctorConversationHandler struct {
	dataAPI api.DataAPI
}

type DoctorMessagesHandler struct {
	dataAPI api.DataAPI
}

func NewDoctorConversationHandler(dataAPI api.DataAPI) *DoctorConversationHandler {
	return &DoctorConversationHandler{
		dataAPI: dataAPI,
	}
}

func (h *DoctorConversationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorId, err := h.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get patient: "+err.Error())
		return
	}
	personId, err := h.dataAPI.GetPersonIdByRole(api.DOCTOR_ROLE, doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get person object for patient: "+err.Error())
		return
	}

	switch r.Method {
	case apiservice.HTTP_GET:
		h.listConversations(w, r, doctorId, personId)
	case apiservice.HTTP_POST:
		h.newConversation(w, r, doctorId, personId)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (h *DoctorConversationHandler) listConversations(w http.ResponseWriter, r *http.Request, doctorId, personId int64) {
	participants := []int64{doctorId}
	if pidStr := r.FormValue("patient_id"); pidStr != "" {
		pid, err := strconv.ParseInt(pidStr, 10, 64)
		if err != nil {
			apiservice.WriteUserError(w, http.StatusBadRequest, "Invalid patient_id")
			return
		}
		patientPersonId, err := h.dataAPI.GetPersonIdByRole(api.PATIENT_ROLE, pid)
		if err != nil {
			apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Failed to get person object for patient: "+err.Error())
			return
		}
		participants = append(participants, patientPersonId)
	}

	con, par, err := h.dataAPI.GetConversationsWithParticipants(participants)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get conversations: "+err.Error())
		return
	}
	res := &conversationListResponse{
		Conversations: conversationsToConversationList(con, personId),
		Participants:  peopleToParticipants(par),
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}

func (h *DoctorConversationHandler) newConversation(w http.ResponseWriter, r *http.Request, doctorId, personId int64) {
	req := &newConversationRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}
	if req.PatientId <= 0 {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Invalid patient_id")
		return
	}

	// Verify the doctor is assigned to the patient
	careTeam, err := h.dataAPI.GetCareTeamForPatient(req.PatientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to get care team based on patient id: "+err.Error())
		return
	}
	primaryDoctorId := apiservice.GetPrimaryDoctorIdFromCareTeam(careTeam)
	if doctorId != primaryDoctorId {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to get the patient information by doctor when this doctor is not the primary doctor for patient")
		return
	}

	toPersonId, err := h.dataAPI.GetPersonIdByRole(api.PATIENT_ROLE, req.PatientId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Failed to get person object for patient: "+err.Error())
		return
	}

	attachments, err := parseAttachments(h.dataAPI, req.Attachments, personId)
	if err != api.NoRowsError {
		apiservice.WriteUserError(w, http.StatusInternalServerError, "Unknown photo")
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get attachments: "+err.Error())
		return
	}
	_, err = h.dataAPI.CreateConversation(personId, toPersonId, req.TopicId, req.Message, attachments)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to create conversation: "+err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}

func NewDoctorMessagesHandler(dataAPI api.DataAPI) *DoctorMessagesHandler {
	return &DoctorMessagesHandler{
		dataAPI: dataAPI,
	}
}

func (h *DoctorMessagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	doctorId, err := h.dataAPI.GetDoctorIdFromAccountId(apiservice.GetContext(r).AccountId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get doctor: "+err.Error())
		return
	}
	personId, err := h.dataAPI.GetPersonIdByRole(api.DOCTOR_ROLE, doctorId)
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "messages: failed to get person object for doctor: "+err.Error())
		return
	}

	switch r.Method {
	case apiservice.HTTP_GET:
		h.listMessages(w, r, doctorId, personId)
	case apiservice.HTTP_POST:
		h.postMessage(w, r, doctorId, personId)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (h *DoctorMessagesHandler) listMessages(w http.ResponseWriter, r *http.Request, doctorId, personId int64) {
	if err := r.ParseForm(); err != nil {
		apiservice.WriteUserError(w, http.StatusBadRequest, "Unable to parse request data: "+err.Error())
		return
	}

	var req conversationRequest
	if err := schema.NewDecoder().Decode(&req, r.Form); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to parse input parameters: "+err.Error())
		return
	}

	// Verify doctor can read the conversation
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

func (h *DoctorMessagesHandler) postMessage(w http.ResponseWriter, r *http.Request, doctorId, personId int64) {
	req := &replyRequest{}
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
	if err != api.NoRowsError {
		apiservice.WriteUserError(w, http.StatusInternalServerError, "Unknown photo")
		return
	} else if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get attachments: "+err.Error())
		return
	}
	if _, err := h.dataAPI.ReplyToConversation(req.ConversationId, personId, req.Message, attachments); err != nil {
		apiservice.WriteDeveloperError(w, http.StatusBadRequest, "Unable to create reply to conversation: "+err.Error())
		return
	}

	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, apiservice.SuccessfulGenericJSONResponse())
}
