package models

import (
	"encoding/json"
	"fmt"
)

// Payload represents the JSON structure of the provided payload file.
type Payload struct {
	ApplicationID string `json:"applicationId"`
	JobID         string `json:"jobId"`
	TaskID        string `json:"taskId"`
	RecordingID   string `json:"recordingId"`
}
func (e Payload) String() string {
	pretty, _ := json.MarshalIndent(&e, "", "\t")
	return string(pretty)
}

type EngineContext struct {
	APIToken     *string
	APIUrl       *string
	APIUsername  *string
	APIPassword  *string
}

func (e EngineContext) String() string {
    return fmt.Sprintf("API: %v, token: %v, username: %v..", *e.APIUrl, *e.APIToken, *e.APIUsername)
}