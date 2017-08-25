package models

// Payload represents the JSON structure of the provided payload file.
type Payload struct {
	ApplicationID string `json:"applicationId"`
	JobID         string `json:"jobId"`
	TaskID        string `json:"taskId"`
	RecordingID   string `json:"recordingId"`
}
