package httpsvc

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/re-connect-ai/reconn/db/dbgen"
	"github.com/re-connect-ai/reconn/shared"
)

// handleCreateAIPerson a gin handler that creates a voice sample record from waveforms of the request.
func (svc *HttpService) handleCreateVoiceSample(c *gin.Context) {
	if c.ContentType() != "audio/wav" && c.ContentType() != "audio/x-wav" && c.ContentType() != "audio/wave" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "request content type must be wave"})
		return
	}
	aiPersonID, err := strconv.Atoi(c.Params.ByName("ai_person_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "request path must contain ai person id"})
		return
	}
	wavContent, err := io.ReadAll(c.Request.Body)
	if err != nil || len(wavContent) < 100 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to read request body"})
		return
	}
	// Name the voice sample after the time of day.
	timestamp := time.Now()
	sampleFileName := fmt.Sprintf("%d-%s.wav", aiPersonID, timestamp.Format(time.RFC3339))
	// Save the file to blob storage and then write to database.
	if _, err := svc.UploadAndSave(c.Request.Context(), svc.Config.VoiceSampleContainer, sampleFileName, svc.Config.VoiceSampleDir, wavContent); err != nil {
		log.Printf("upload and save error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	// Save to file on disk and then write to database.
	voiceSample, err := svc.Database.CreateVoiceSample(c.Request.Context(), dbgen.CreateVoiceSampleParams{
		AiPersonID: int64(aiPersonID),
		FileName:   sql.NullString{String: sampleFileName, Valid: true},
		Timestamp:  timestamp,
	})
	if err != nil {
		log.Printf("create voice sample error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, voiceSample)
}

// handleListUsers is a gin handler that lists all voice samples of an AI person.
func (svc *HttpService) handleListVoiceSamples(c *gin.Context) {
	aiPersonID, _ := strconv.Atoi(c.Params.ByName("ai_person_id"))
	voiceSamples, err := svc.Database.ListVoiceSamples(c.Request.Context(), int64(aiPersonID))
	if err != nil {
		log.Printf("list voice sample error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, voiceSamples)
}

// handleGetLatestVoiceModel is a gin handler that retrieves the latest voice model of an AI person.
func (svc *HttpService) handleGetLatestVoiceModel(c *gin.Context) {
	aiPersonID, _ := strconv.Atoi(c.Params.ByName("ai_person_id"))
	latestModel, err := svc.Database.GetLatestVoiceModel(c.Request.Context(), int64(aiPersonID))
	if err != nil {
		log.Printf("get latest voice model error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, latestModel)
}

// Update voice model status by ID is not needed for debugging.

// handleCreateVoiceModel is a gin handler that creates a new voice model by relaying a clone request to voice service in real time.
func (svc *HttpService) handleCreateVoiceModel(c *gin.Context) {
	voiceSampleID, _ := strconv.Atoi(c.Params.ByName("voice_sample_id"))
	// Retrieve the sample record from database.
	voiceSample, err := svc.Database.GetVoiceSampleByID(c.Request.Context(), int64(voiceSampleID))
	if err != nil {
		log.Printf("get voice sample by id error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	// Retrieve the sample wave file from blob storage.
	localFilePath, err := svc.DownloadBlobToLocalFileIfNotExist(c.Request.Context(), svc.Config.VoiceSampleContainer, voiceSample.FileName.String, svc.Config.VoiceSampleDir)
	if err != nil {
		log.Printf("blob download file error: %v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	// Open the wave file from local disk.
	voiceSampleFile, err := os.Open(localFilePath)
	defer voiceSampleFile.Close()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	// Relay the clone request to voice service.
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/clone-rt/%d", svc.Config.VoiceServiceAddr, voiceSampleID), voiceSampleFile)
	req.Header.Set("content-type", "audio/wav")
	if err != nil {
		log.Printf("failed to construct clone-rt request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to construct voice service request"})
		return
	}
	resp, err := svc.VoiceClient.Do(req)
	if err != nil {
		log.Printf("failed to make clone-rt request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to make voice service request"})
		return
	}
	log.Printf("clone-rt responded with status %d and content length %d", resp.StatusCode, resp.ContentLength)
	var cloneResp shared.CloneRealTimeResponse
	if err := json.NewDecoder(resp.Body).Decode(&cloneResp); err != nil {
		log.Printf("failed to deserialise clone-rt response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to make voice service request"})
		return
	}
	// Back to this handler, create the cloned voice model record in database.
	voiceModel, err := svc.Database.CreateVoiceModel(c.Request.Context(), dbgen.CreateVoiceModelParams{
		VoiceSampleID: int64(voiceSampleID),
		Status:        "ready",
		FileName:      sql.NullString{String: cloneResp.ModelDestinationFile, Valid: true},
		Timestamp:     time.Now(),
	})
	// Store the voice model in blob storage.
	if err := svc.UploadFromLocalFile(c.Request.Context(), svc.Config.VoiceModelContainer, cloneResp.ModelDestinationFile, svc.Config.VoiceModelDir); err != nil {
		log.Printf("upload from local file error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, voiceModel)
}
