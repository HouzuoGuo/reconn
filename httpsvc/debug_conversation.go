package httpsvc

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/re-connect-ai/reconn/db/dbgen"
	openai "github.com/sashabaranov/go-openai"
)

func (svc *HttpService) chatCompletionRequest(ctx context.Context, aiPersonID int, contextPrompt, newUserPrompt string) (ret openai.ChatCompletionRequest, err error) {
	// Read the latest 10 messages back and forth.
	recentMessages, err := svc.Config.Database.ListConversations(ctx, dbgen.ListConversationsParams{
		AiPersonID: int64(aiPersonID),
		Limit:      10,
	})
	if err != nil {
		log.Printf("get latest conversations error: %v", err)
		return
	}
	ret = openai.ChatCompletionRequest{
		Model: "gpt-4",
		// TODO FIXME: the response abruptly ends after exceeding the token limit.
		MaxTokens: 50, // good for about 250 characters of response.

		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: contextPrompt,
			},
		},
	}
	// Give the latest back and forth message to the completion request.
	for i := len(recentMessages) - 1; i >= 0; i-- {
		recent := recentMessages[i]
		if userPrompt := recent.VoiceTranscription.String; userPrompt != "" {
			ret.Messages = append(ret.Messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			})
		} else if userPrompt := recent.TextMessage.String; userPrompt != "" {
			ret.Messages = append(ret.Messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			})
		}
		if aiReply := recent.ReplyMessage.String; aiReply != "" {
			ret.Messages = append(ret.Messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: aiReply,
			})
		}
	}
	// And here goes the prompt from the user. Feed the completion request to LLM.
	ret.Messages = append(ret.Messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: newUserPrompt,
	})
	return
}

type PostTextMessage struct {
	Message string `json:"message"`
}

// handlePostTextMessage is a gin handler that posts a text message to an AI person and synchronously awaits for a response.
func (svc *HttpService) handlePostTextMessage(c *gin.Context) {
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
	// Convert the reply into voice in real time.
	ttsRequestBody, err := json.Marshal(TextToSpeechRealTimeRequest{
		Text:         llmReply,
		TopK:         99,
		TopP:         0.8,
		MineosP:      0.01,
		SemanticTemp: 0.7,
		WaveformTemp: 0.6,
		FineTemp:     0.5,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	// Download the model file to local disk and then relay to python voice server.
	if _, err := svc.DownloadModelIfNotExist(c.Request.Context(), aiPersonAndModel.FileName.String); err != nil {
		log.Printf("download model error: %v", err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	ttsRequest, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/tts-rt/%s", svc.Config.VoiceServiceAddr, strings.TrimSuffix(aiPersonAndModel.FileName.String, ".npz")), bytes.NewReader(ttsRequestBody))
	ttsRequest.Header.Set("content-type", "application/json")
	if err != nil {
		log.Printf("tts request construction error: %v", err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	ttsResponse, err := svc.VoiceClient.Do(ttsRequest)
	if err != nil {
		log.Printf("tts request error: %v", err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	log.Printf("tts-rt responded with status %d and content length %d", ttsResponse.StatusCode, ttsResponse.ContentLength)
	ttsWaveContent, err := ioutil.ReadAll(ttsResponse.Body)
	if err != nil {
		log.Printf("failed to read tts response body: %v", err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	// Save the converted speech.
	fileName := fmt.Sprintf("%d-%s.wav", aiPersonID, timestamp.Format(time.RFC3339))
	if _, err := svc.UploadAndSave(c.Request.Context(), svc.Config.VoiceOutputContainer, fileName, svc.Config.VoiceOutputDir, ttsWaveContent); err != nil {
		log.Printf("upload and save error: %v", err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	// Create the AI reply record in database.
	aiReplyVoice, err := svc.Config.Database.CreateAIPersonReplyVoice(c.Request.Context(), dbgen.CreateAIPersonReplyVoiceParams{
		AiPersonReplyID: aiReply.ID,
		Status:          "ready",
		FileName:        sql.NullString{String: fileName, Valid: true},
	})
	c.JSON(http.StatusOK, aiReplyVoice)
}

// handlePostTextMessage is a gin handler that posts a voice message to an AI person and synchronously awaits for a response.
func (svc *HttpService) handlePostVoiceMessage(c *gin.Context) {
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
	// Convert the reply into voice in real time.
	ttsRequestBody, err := json.Marshal(TextToSpeechRealTimeRequest{
		Text:         llmReply,
		TopK:         99,
		TopP:         0.8,
		MineosP:      0.01,
		SemanticTemp: 0.7,
		WaveformTemp: 0.6,
		FineTemp:     0.5,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	ttsRequest, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/tts-rt/%s", svc.Config.VoiceServiceAddr, strings.TrimSuffix(aiPersonAndModel.FileName.String, ".npz")), bytes.NewReader(ttsRequestBody))
	ttsRequest.Header.Set("content-type", "application/json")
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	ttsResponse, err := svc.VoiceClient.Do(ttsRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	log.Printf("tts-rt responded with status %d and content length %d", ttsResponse.StatusCode, ttsResponse.ContentLength)
	ttsWaveContent, err := ioutil.ReadAll(ttsResponse.Body)
	if err != nil {
		log.Printf("failed to read tts response body: %v", err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	// Save the converted speech.
	fileName := fmt.Sprintf("reply-%d-%s.wav", aiPersonID, timestamp.Format(time.RFC3339))
	if _, err := svc.UploadAndSave(c.Request.Context(), svc.Config.VoiceOutputContainer, fileName, svc.Config.VoiceOutputDir, ttsWaveContent); err != nil {
		log.Printf("upload and save error: %v", err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	// Create the AI reply record in database.
	aiReplyVoice, err := svc.Config.Database.CreateAIPersonReplyVoice(c.Request.Context(), dbgen.CreateAIPersonReplyVoiceParams{
		AiPersonReplyID: aiReply.ID,
		Status:          "ready",
		FileName:        sql.NullString{String: fileName, Valid: true},
	})
	c.JSON(http.StatusOK, aiReplyVoice)
}

// handleGetAIPersonConversation is a gin handler that returns the full conversation going back and forth between a user and an AI person.
func (svc *HttpService) handleGetAIPersonConversation(c *gin.Context) {
	aiPersonID, _ := strconv.Atoi(c.Params.ByName("ai_person_id"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	conversations, err := svc.Config.Database.ListConversations(c.Request.Context(), dbgen.ListConversationsParams{
		AiPersonID: int64(aiPersonID),
		Limit:      int32(limit),
	})
	if err != nil {
		log.Printf("list conversation error: %v", err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, conversations)
}

// handleGetVoiceOutputFile returns the waveform file content of the requested file name.
func (svc *HttpService) handleGetVoiceOutputFile(c *gin.Context) {
	fileName := path.Base(c.Params.ByName("file_name"))
	log.Printf("reading voice output file: %q", fileName)
	localFilePath, err := svc.DownloadBlobToLocalFileIfNotExist(c.Request.Context(), svc.Config.VoiceOutputContainer, fileName, svc.Config.VoiceOutputDir)
	if err != nil {
		log.Printf("download blob error: %v", err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	// Read file info for the content length.
	fileInfo, err := os.Stat(localFilePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	fileContent, err := os.Open(localFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.DataFromReader(http.StatusOK, fileInfo.Size(), "audio/wav", fileContent, nil)
}
