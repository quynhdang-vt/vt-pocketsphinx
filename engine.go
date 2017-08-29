package main

import (
	"github.com/quynhdang-vt/vt-pocketsphinx/models"
	"io"
	"io/ioutil"
	"net/http"
	"github.com/xlab/pocketsphinx-go/sphinx"
	veritoneAPI "github.com/veritone/go-veritone-api"	
	"github.com/quynhdang-vt/vt-pocketsphinx/cmu_sphinx"
	"context"
	"fmt"
	"sort"
	"bytes"
	"log"
)

func downloadFile(url string, prefix string) (filepath *string, err error) {

    log.Printf("downloadFile ENTER url=%s --> %s\n", url, prefix)
	out, err := ioutil.TempFile("", prefix)
	if err != nil {
		return nil, err
	}
	defer func() { out.Close(); log.Printf("downloadFile EXIT") } ()

	s := out.Name()
	filepath = &s

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}

	return filepath, nil
}

/** as outlined in https://veritone-developer.atlassian.net/wiki/spaces/DOC/pages/15106076/Engine+Execution+Process */
func RunEngine(payload models.Payload, engineContext models.EngineContext,
	dec *sphinx.Decoder,
	veritoneAPIClient veritoneAPI.VeritoneAPIClient) (err error) {
		
		log.Println("RunEngine started..")
		defer log.Println("RunEngine exited..")
		
	// Set task to running
	err = veritoneAPIClient.UpdateTaskStatus(context.Background(), payload.JobID, payload.TaskID, veritoneAPI.TaskStatusRunning, nil)
	if err != nil {
		// continue on failure
		fmt.Printf("failed to set task to running, continuing execution regardless: %s\n", err)
	}

	// Get Recording Assets
	log.Printf("Getting recording %v\n", payload.RecordingID)
	recording, err := veritoneAPIClient.GetRecording(context.Background(), payload.RecordingID)
	if err != nil {
		return fmt.Errorf("error getting recording: %s", err)
	}

	if recording == nil {
		return fmt.Errorf("recording not found")
	}

	if len(recording.Assets) == 0 {
		return fmt.Errorf("recording has no assets")
	}

	assetURI := ""
/*
	if payload.AssetID != "" {
		for _, asset := range recording.Assets {
			if asset.AssetID == payload.AssetID {
				assetURI = asset.SignedURI
				break
			}
		}

		if assetURI == "" {
			return fmt.Errorf("unable to find specified assetId")
		}
	} else {
*/
		// Use oldest asset as default
		sort.SliceStable(recording.Assets, func(i, j int) bool {
			return recording.Assets[i].CreatedDateTime < recording.Assets[j].CreatedDateTime
		})

        // Need to loop thru and looking for files that we can do,
        // either audio/wav or audio/mpeg
		assetURI = recording.Assets[0].SignedURI
//	}

	// assetURI has the file
	prefix := "vt-pocketsphinx-" + payload.RecordingID + "-" + payload.JobID + "-" + payload.TaskID
	filepath, err := downloadFile(assetURI, prefix)
	if err != nil {
		return fmt.Errorf("Failed to download recording with prefix == " + prefix)
	}
	w := &cmu_sphinx.UnitOfWork{InfileName: filepath, Dec: dec}
	ttml, transcriptDurationMs, err := w.Decode()
	if err != nil {
		return fmt.Errorf("Failed to process file " + *filepath)
	}

	ttmlBytes := []byte(*ttml)

	ttmlAsset := veritoneAPI.Asset{
		AssetType:   "transcript",
		ContentType: "application/ttml+xml",
		Metadata: map[string]interface{}{
			"fileName": "qd-pocketsphinx.ttml",
			"source":   "qd-pocketsphinx",
			"size":     len(ttmlBytes),
		},
	}

	asset, _, err := veritoneAPIClient.CreateAsset(context.Background(), payload.RecordingID, bytes.NewReader(ttmlBytes), ttmlAsset)
	if err != nil {
		return fmt.Errorf("failed to create ttmlAsset: %s", err)
	}

    log.Printf("Created asset %v", asset)
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
		return fmt.Errorf("failed to set task to complete: %s", err)
	}
    return err
}
