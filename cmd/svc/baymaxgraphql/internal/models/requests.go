package models

// RequestStatus represents the status of an async api request
type RequestStatus struct {
	ID                 string   `json:"id"`
	Type               string   `json:"type"`
	Status             string   `json:"status"`
	Description        string   `json:"description"`
	Errors             []string `json:"errors"`
	TasksRequested     uint64   `json:"tasksRequested"`
	TasksCompleted     uint64   `json:"tasksCompleted"`
	TasksErrored       uint64   `json:"tasksErrored"`
	CreatedTimestamp   uint64   `json:"createdTimestamp"`
	CompletedTimestamp uint64   `json:"completedTimestamp,omitempty"`
}
