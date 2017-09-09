package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/quynhdang-vt/vt-pocketsphinx/cmu_sphinx"
	"github.com/quynhdang-vt/vt-pocketsphinx/models"
	utils "github.com/quynhdang-vt/vt-pocketsphinx/utils"
	veritoneAPI "github.com/veritone/go-veritone-api"
	"github.com/xlab/pocketsphinx-go/sphinx"
	"log"
	"sort"
	"time"
	"os"
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
			if utils.IsSupportedContentType(asset.ContentType) {
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

	// Set task to running
	err = veritoneAPIClient.UpdateTaskStatus(context.Background(), payload.JobID, payload.TaskID, veritoneAPI.TaskStatusRunning, nil)
	if err != nil {
		// continue on failure
		fmt.Printf("Failed to set task to running, ignored: %s\n", err)
	}

	defer func() {
		// if there's error, we should update the task as failing
		if err != nil {
			_ = veritoneAPIClient.UpdateTaskStatus(context.Background(), payload.JobID, payload.TaskID, veritoneAPI.TaskStatusFailed, err)
		}
		log.Println("==== RunEngine exited..")
	}()
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
	origFilepath, err := utils.DownloadFile(chosenAsset.SignedURI, prefix)
	if err != nil {
		return fmt.Errorf("Failed to download recording with prefix == " + prefix)
	}
	// go ahead and convert the file -- for now
	// TODO eventually we may have to "break" the file up if it is too long 5' is maxed for now
	filepath, err := utils.ConvertFileToWave16KMono(origFilepath)
	w := &cmu_sphinx.UnitOfWork{InfileName: &filepath, Dec: dec}
	ttml, latticeJson, interesting_tidbits, err := w.Decode()
	if err != nil {
		return fmt.Errorf("Failed to process file %v, err=%v", filepath, err)
	}

	engineId := os.Getenv("ENGINE_ID")
	nowTime := time.Now()
	interesting_tidbits["engine_id"]=engineId
	interesting_tidbits["now"]=nowTime.String()
	// VLF file
	log.Printf("Creating VLF asset..")
	jsonAsset := veritoneAPI.Asset{
		AssetType:   "v-vlf",
		ContentType: "application/json",
		Metadata: map[string]interface{}{
			"fileName": fmt.Sprintf("qd-pocketsphinx-%d.json", time.Now().Unix()),
			"source":   engineId,
			"utime": time.Now().String(),
			"size":     len(latticeJson),
		},
	}

	asset, _, err := veritoneAPIClient.CreateAsset(context.Background(), payload.RecordingID, bytes.NewReader(latticeJson), jsonAsset)
	if err != nil {
		return fmt.Errorf("Failed to create JSON Asset: %s", err)
	}
	log.Printf("Created VLF asset %v\n", asset)


	// TTML file
	log.Printf("Creating TTML asset..")

	ttmlAsset := veritoneAPI.Asset{
		AssetType:   "transcript",
		ContentType: "application/ttml+xml",
		Metadata: map[string]interface{}{
			"fileName": fmt.Sprintf("qd-pocketsphinx-%d.ttml", nowTime.Unix()),
			"source":   engineId,
			"utime": nowTime.String(),
			"size":     len(ttml),
		},
	}

	asset, _, err = veritoneAPIClient.CreateAsset(context.Background(), payload.RecordingID, bytes.NewReader(ttml), ttmlAsset)
	if err != nil {
		return fmt.Errorf("Failed to create TTML Asset: %s", err)
	}
	log.Printf("Created TTML asset %v\n", asset)
	// now creating ttml

	// Set task to complete
	err = veritoneAPIClient.UpdateTaskStatus(
		context.Background(),
		payload.JobID,
		payload.TaskID,
		veritoneAPI.TaskStatusComplete,
		interesting_tidbits,
	)
	if err != nil {
		return fmt.Errorf("Failed to set task %d to complete: %s", err)
	}
	return err
}
