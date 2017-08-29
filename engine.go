package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/quynhdang-vt/vt-pocketsphinx/cmu_sphinx"
	"github.com/quynhdang-vt/vt-pocketsphinx/models"
	qdUtils "github.com/quynhdang-vt/vt-pocketsphinx/utils"
	veritoneAPI "github.com/veritone/go-veritone-api"
	"github.com/xlab/pocketsphinx-go/sphinx"
	"log"
	"sort"
	"time"
)

func getAsset(payload models.Payload, recording *veritoneAPI.Recording) (chosenAsset *veritoneAPI.Asset) {

	if payload.AssetID != "" {
		for _, asset := range recording.Assets {
			if asset.AssetID == payload.AssetID {
				return &asset
			}
		}
	} else {

		// Use oldest asset as default
		sort.SliceStable(recording.Assets, func(i, j int) bool {
			return recording.Assets[i].CreatedDateTime < recording.Assets[j].CreatedDateTime
		})

		// Need to loop thru and looking for files that we can do,
		// either audio/wav or audio/mpeg
		for _, asset := range recording.Assets {
			if qdUtils.IsSupportedContentType(asset.ContentType) {
				return &asset
			}
		}
	}

	return nil
}

/** as outlined in https://veritone-developer.atlassian.net/wiki/spaces/DOC/pages/15106076/Engine+Execution+Process */
func RunEngine(payload models.Payload, engineContext models.EngineContext,
	dec *sphinx.Decoder,
	veritoneAPIClient veritoneAPI.VeritoneAPIClient) (err error) {

	log.Println("==== RunEngine started..")
	defer log.Println("==== RunEngine exited..")

	// Set task to running
	err = veritoneAPIClient.UpdateTaskStatus(context.Background(), payload.JobID, payload.TaskID, veritoneAPI.TaskStatusRunning, nil)
	if err != nil {
		// continue on failure
		fmt.Printf("Failed to set task to running, ignored: %s\n", err)
	}

	// Get Recording Assets
	log.Printf("Getting recording %v\n", payload.RecordingID)
	recording, err := veritoneAPIClient.GetRecording(context.Background(), payload.RecordingID)
	if err != nil {
		return fmt.Errorf("error getting recording: %s", err)
	}

	if recording == nil {
		return fmt.Errorf("recording not found for %s", payload.RecordingID)
	}

	if len(recording.Assets) == 0 {
		return fmt.Errorf("recording has no assets")
	}

	chosenAsset := getAsset(payload, recording)
	if chosenAsset == nil {
		return fmt.Errorf("No suitable asset can be found..")
	}
	log.Printf("FOUND ASSET: %v, %v...\n", chosenAsset.AssetID, chosenAsset.ContentType)

	prefix := "vt-pocketsphinx-" + payload.RecordingID + "-" + payload.JobID + "-" + payload.TaskID
	origFilepath, err := qdUtils.DownloadFile(chosenAsset.SignedURI, prefix)
	if err != nil {
		return fmt.Errorf("Failed to download recording with prefix == " + prefix)
	}
	// go ahead and convert the file -- for now
	// TODO eventually we may have to "break" the file up if it is too long 5' is maxed for now
	filepath, err := qdUtils.ConvertFileToWave16KMono(origFilepath)
	w := &cmu_sphinx.UnitOfWork{InfileName: &filepath, Dec: dec}
	ttml, transcriptDurationMs, err := w.Decode()
	if err != nil {
		return fmt.Errorf("Failed to process file %v, err=%v", filepath, err)
	}

	ttmlBytes := []byte(*ttml)

	ttmlAsset := veritoneAPI.Asset{
		AssetType:   "transcript",
		ContentType: "application/ttml+xml",
		Metadata: map[string]interface{}{
			"fileName": fmt.Sprintf("qd-pocketsphinx-%d.ttml", time.Now().Unix()),
			"source":   fmt.Sprintf("qd-pocketsphinx:%v", time.Now()),
			"size":     len(ttmlBytes),
		},
	}

	asset, _, err := veritoneAPIClient.CreateAsset(context.Background(), payload.RecordingID, bytes.NewReader(ttmlBytes), ttmlAsset)
	if err != nil {
		return fmt.Errorf("Failed to create ttmlAsset: %s", err)
	}

	log.Printf("Created asset %v\n", asset)
	// Set task to complete
	err = veritoneAPIClient.UpdateTaskStatus(
		context.Background(),
		payload.JobID,
		payload.TaskID,
		veritoneAPI.TaskStatusComplete,
		map[string]interface{}{
			"transcriptDurationMs": transcriptDurationMs,
		},
	)
	if err != nil {
		return fmt.Errorf("Failed to set task %d to complete: %s", err)
	}
	return err
}
