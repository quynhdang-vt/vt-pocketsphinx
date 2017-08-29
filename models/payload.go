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
	AssetID       string `json:"assetId"`
}

func (e Payload) String() string {
	pretty, _ := json.MarshalIndent(&e, "", "\t")
	return string(pretty)
}
func (e *Payload) IsInvalid() bool {
	return (len(e.ApplicationID) == 0 ||
		len(e.JobID) == 0 ||
		len(e.TaskID) == 0 ||
		len(e.RecordingID) == 0)
}

type EngineContext struct {
	APIToken    string `json:"apiToken"`
	APIUrl      string `json:"apiUrl"`
	APIUsername string `json:"apiUsername"`
	APIPassword string `json:"apiPassword"`
}

func (e EngineContext) String() string {
	return fmt.Sprintf("API: %v, token: %v, username: %v..", e.APIUrl, e.APIToken, e.APIUsername)
}
func (e *EngineContext) IsInvalid() bool {
	return (len(e.APIUrl) == 0 ||
		(len(e.APIToken) == 0 && len(e.APIUsername) == 0 && len(e.APIPassword) == 0))
}
