package httpsvc

import (
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/re-connect-ai/reconn/db/dbgen"
	openai "github.com/sashabaranov/go-openai"
)

// handleCreateVoiceModelAsync is a gin handler that posts a message to the GPU worker queue to create a voice model.
func (svc *HttpService) handleCreateVoiceModelAsync(c *gin.Context) {
	voiceSampleID, _ := strconv.Atoi(c.Params.ByName("voice_sample_id"))
	// Retrieve the sample record from database.
	voiceSample, err := svc.Config.Database.GetVoiceSampleByID(c.Request.Context(), int64(voiceSampleID))
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
	// Back to this handler, create the cloned voice model record in database.
	voiceModel, err := svc.Config.Database.CreateVoiceModel(c.Request.Context(), dbgen.CreateVoiceModelParams{
		VoiceSampleID: int64(voiceSampleID),
		Status:        "processing",
		FileName:      sql.NullString{String: "TODO FIXME.TODO FIXME", Valid: true},
		Timestamp:     time.Now(),
	})
	// TODO FIXME post to queue
	c.JSON(http.StatusOK, voiceModel)
}

// handlePostTextMessageAsync is a gin handler that posts a text message to an AI person, and post a message to the GPU worker queue for a TTS reply.
func (svc *HttpService) handlePostTextMessageAsync(c *gin.Context) {
	aiPersonID, _ := strconv.Atoi(c.Params.ByName("ai_person_id"))
	var req PostTextMessage
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	// Create the text prompt in database.
	prompt, err := svc.Config.Database.CreateUserPrompt(c.Request.Context(), dbgen.CreateUserPromptParams{
		AiPersonID: int64(aiPersonID),
		Timestamp:  time.Now(),
	})
	if err != nil {
		log.Printf("create user prompt error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	textPrompt, err := svc.Config.Database.CreateUserTextPrompt(c.Request.Context(), dbgen.CreateUserTextPromptParams{
		UserPromptID: prompt.ID,
		Message:      req.Message,
	})
	log.Printf("prompt: %+v, text prompt: %+v", prompt, textPrompt)
	if err != nil {
		log.Printf("create user text prompt error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	// Read the voice model and context prompt from this AI person.
	aiPersonAndModel, err := svc.Config.Database.GetLatestVoiceModel(c.Request.Context(), int64(aiPersonID))
	if err != nil {
		log.Printf("get latest voice model error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("ai person and model: %+v", aiPersonAndModel)
	// Generate the chat completion request, given the recent history.
	completionRequest, err := svc.chatCompletionRequest(c.Request.Context(), aiPersonID, aiPersonAndModel.AiContextPrompt, req.Message)
	if err != nil {
		log.Printf("chat completion request construction error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("Chat completion request for AI person %d is: %+v", aiPersonID, completionRequest)
	// Feed both context prompt and text prompt to LLM.
	resp, err := svc.OpenAIClient.CreateChatCompletion(c.Request.Context(), completionRequest)
	if err != nil {
		log.Printf("create chat completion error: %v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	var llmReply string
	for _, choice := range resp.Choices {
		llmReply += choice.Message.Content + " "
	}
	// Create the AI person reply in database.
	timestamp := time.Now()
	aiReply, err := svc.Config.Database.CreateAIPersonReply(c.Request.Context(), dbgen.CreateAIPersonReplyParams{
		UserPromptID: prompt.ID,
		Status:       "ready",
		Message:      llmReply,
		Timestamp:    timestamp,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("ai reply: %+v", aiReply)
	// TODO FIXME post to queue
	// Create the AI reply record in database.
	aiReplyVoice, err := svc.Config.Database.CreateAIPersonReplyVoice(c.Request.Context(), dbgen.CreateAIPersonReplyVoiceParams{
		AiPersonReplyID: aiReply.ID,
		Status:          "processing",
		FileName:        sql.NullString{String: "TODO FIXME.TODO FIXME", Valid: true},
	})
	c.JSON(http.StatusOK, aiReplyVoice)
}

// handlePostTextMessageAsync is a gin handler that posts a text message to an AI person, and post a message to the GPU worker queue for a TTS reply.
func (svc *HttpService) handlePostVoiceMessageAsync(c *gin.Context) {
	aiPersonID, _ := strconv.Atoi(c.Params.ByName("ai_person_id"))
	if c.ContentType() != "audio/wav" && c.ContentType() != "audio/x-wav" && c.ContentType() != "audio/wave" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "request content type must be wave"})
		return
	}
	voiceWaveform, err := ioutil.ReadAll(c.Request.Body)
	if err != nil || len(voiceWaveform) < 100 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to read request body"})
		return
	}
	// Save the voice message to disk.
	timestamp := time.Now()
	sampleFileName := fmt.Sprintf("prompt-%d-%s.wav", aiPersonID, timestamp.Format(time.RFC3339))
	if _, err := svc.UploadAndSave(c.Request.Context(), svc.Config.VoiceOutputContainer, sampleFileName, svc.Config.VoiceOutputDir, voiceWaveform); err != nil {
		log.Printf("upload and save error: %v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	// Transcribe the message in real time.
	transcriptionResponse, err := svc.OpenAIClient.CreateTranscription(c.Request.Context(), openai.AudioRequest{
		Model: "whisper-1",
		// The file path is part of the form submission, the extension name must accurately indicate the audio format.
		FilePath: "input.wav",
		Reader:   bytes.NewReader(voiceWaveform),
		Format:   openai.AudioResponseFormatJSON,
	})
	if err != nil {
		log.Printf("failed to invoke whisper: %v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	// Create the user voice prompt in database.
	prompt, err := svc.Config.Database.CreateUserPrompt(c.Request.Context(), dbgen.CreateUserPromptParams{
		AiPersonID: int64(aiPersonID),
		Timestamp:  time.Now(),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	voicePrompt, err := svc.Config.Database.CreateUserVoicePrompt(c.Request.Context(), dbgen.CreateUserVoicePromptParams{
		UserPromptID:  prompt.ID,
		Status:        "ready",
		FileName:      sampleFileName,
		Transcription: sql.NullString{String: transcriptionResponse.Text, Valid: true},
	})
	if err != nil {
		log.Printf("create user voice prompt error: %v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("prompt: %+v, voice prompt: %+v", prompt, voicePrompt)
	// Read the voice model and context prompt from this AI person.
	aiPersonAndModel, err := svc.Config.Database.GetLatestVoiceModel(c.Request.Context(), int64(aiPersonID))
	if err != nil {
		log.Printf("get latest voice model error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("ai person and model: %+v", aiPersonAndModel)
	// Generate the chat completion request, given the recent history.
	completionRequest, err := svc.chatCompletionRequest(c.Request.Context(), aiPersonID, aiPersonAndModel.AiContextPrompt, transcriptionResponse.Text)
	if err != nil {
		log.Printf("chat completion request construction error: %+v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("Chat completion request for AI person %d is: %+v", aiPersonID, completionRequest)
	resp, err := svc.OpenAIClient.CreateChatCompletion(c.Request.Context(), completionRequest)
	if err != nil {
		log.Printf("create chat completion error: %v", err)
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	var llmReply string
	for _, choice := range resp.Choices {
		llmReply += choice.Message.Content + " "
	}
	// Create the AI person reply in database.
	aiReply, err := svc.Config.Database.CreateAIPersonReply(c.Request.Context(), dbgen.CreateAIPersonReplyParams{
		UserPromptID: prompt.ID,
		Status:       "ready",
		Message:      llmReply,
		Timestamp:    timestamp,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("ai reply: %+v", aiReply)
	// TODO FIXME post to queue
	// Create the AI reply record in database.
	aiReplyVoice, err := svc.Config.Database.CreateAIPersonReplyVoice(c.Request.Context(), dbgen.CreateAIPersonReplyVoiceParams{
		AiPersonReplyID: aiReply.ID,
		Status:          "processing",
		FileName:        sql.NullString{String: "TODO FIXME.TODO FIXME", Valid: true},
	})
	c.JSON(http.StatusOK, aiReplyVoice)
}
