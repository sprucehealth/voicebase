package messages

import (
	"carefront/api"
	"carefront/apiservice"
	"net/http"
)

type TopicsHandler struct {
	dataAPI api.DataAPI
}

type topic struct {
	Id    int64  `json:"id,string"`
	Title string `json:"title"`
}

type topicsResponse struct {
	Topics []*topic `json:"topics"`
}

func NewTopicsHandler(dataAPI api.DataAPI) *TopicsHandler {
	return &TopicsHandler{dataAPI: dataAPI}
}

func (h *TopicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topics, err := h.dataAPI.GetConversationTopics()
	if err != nil {
		apiservice.WriteDeveloperError(w, http.StatusInternalServerError, "Failed to get topics: "+err.Error())
		return
	}
	res := &topicsResponse{
		Topics: make([]*topic, len(topics)),
	}
	for i, t := range topics {
		res.Topics[i] = &topic{
			Id:    t.Id,
			Title: t.Title,
		}
	}
	apiservice.WriteJSONToHTTPResponseWriter(w, http.StatusOK, res)
}
