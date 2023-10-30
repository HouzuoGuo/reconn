package httpsvc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/re-connect-ai/reconn/shared"
	openai "github.com/sashabaranov/go-openai"
)

// handleReadback is a gin handler that reads back several parameters from the request.
// This is only used for experimenting, do not expose to the Internet.
func (svc *HttpService) handleReadback(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"address": c.Request.RemoteAddr,
		"headers": c.Request.Header,
		"method":  c.Request.Method,
		"url":     c.Request.URL.String(),
	})
}

// handleRelayCloneRealTime is a gin handler that relays a real time voice cloning request to the voice service.
// This is only used for experimenting, do not expose to the Internet.
func (svc *HttpService) handleRelayCloneRealTime(c *gin.Context) {
	if c.ContentType() != "audio/wav" && c.ContentType() != "audio/x-wav" && c.ContentType() != "audio/wave" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "request content type must be wave"})
		return
	}
	userID := c.Params.ByName("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "user_id must be present"})
		return
	}
	wavContent, err := io.ReadAll(c.Request.Body)
	if err != nil || len(wavContent) < 100 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to read request body"})
		return
	}
	// Relay to voice service.
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/clone-rt/%s", svc.Config.VoiceServiceAddr, userID), bytes.NewReader(wavContent))
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
	c.JSON(http.StatusOK, cloneResp)
}

// handleRelayTextToSpeechRealTime is a gin handler that relays a real time text-to-speech request to the voice service.
// This is only used for experimenting, do not expose to the Internet.
func (svc *HttpService) handleRelayTextToSpeechRealTime(c *gin.Context) {
	userID := c.Params.ByName("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "user_id must be present"})
		return
	}
	var ttsRequest shared.TextToSpeechRealTimeRequest
	if err := c.BindJSON(&ttsRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to deserialise request"})
		return
	}
	if len(ttsRequest.Text) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "text must be longer than 2 characters"})
		return
	}
	// Relay to voice service.
	relayRequest, err := json.Marshal(ttsRequest)
	if err != nil {
		log.Printf("failed to construct tts-rt request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to construct voice service request"})
		return
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/tts-rt/%s", svc.Config.VoiceServiceAddr, userID), bytes.NewReader(relayRequest))
	req.Header.Set("content-type", "application/json")
	if err != nil {
		log.Printf("failed to construct tts-rt request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to construct voice service request"})
		return
	}
	resp, err := svc.VoiceClient.Do(req)
	if err != nil {
		log.Printf("failed to make tts-rt request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to make voice service request"})
		return
	}
	log.Printf("tts-rt responded with status %d and content length %d", resp.StatusCode, resp.ContentLength)
	wavContent, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to make tts-rt request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to make voice service request"})
		return
	}
	c.DataFromReader(http.StatusOK, int64(len(wavContent)), "audio/wav", bytes.NewReader(wavContent), nil)
}

// ListVoiceModelResponse is the structure of GET /voice-model response.
type ListVoiceModelResponse struct {
	Models map[string]VoiceModel `json:"models"`
}

// ListVoiceModelResponse describes a single cloned voice model.
type VoiceModel struct {
	FileName     string    `json:"fileName"`
	UserID       string    `json:"userId"`
	LastModified time.Time `json:"lastModified"`
}

// handleListVoiceModel is a gin handler that responds with the current list of cloned voice models.
// It should only be used by the experimental web app.
func (svc *HttpService) handleListVoiceModel(c *gin.Context) {
	entries, err := os.ReadDir(svc.Config.VoiceModelDir)
	if err != nil {
		log.Printf("failed to read voice model directory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to read voice model directory"})
		return
	}
	resp := ListVoiceModelResponse{Models: map[string]VoiceModel{}}
	for _, entry := range entries {
		fileName := entry.Name()
		if filepath.Ext(fileName) != ".npz" {
			continue
		}
		userID := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		resp.Models[userID] = VoiceModel{
			FileName: fileName,
			UserID:   userID,
		}
	}
	c.JSON(http.StatusOK, resp)
}

// TextToSpeechRealTimeRequest is the structure of /converse-single-prompt/ request.
type ConverseSinglePromptRequest struct {
	SystemPrompt string `json:"systemPrompt"`
	UserPrompt   string `json:"userPrompt"`
}

// TextToSpeechRealTimeResponse is the structure of /converse-single-prompt/ response.
type ConverseSinglePromptResponse struct {
	Reply string `json:"reply"`
}

// handleConverseWithSystemRole is a gin handler that converses with chatgpt in a singular prompt - 1xQ for 1xA.
// This is only used for experimenting, do not expose to the Internet.
func (svc *HttpService) handleConverseSinglePrompt(c *gin.Context) {
	var converseRequest ConverseSinglePromptRequest
	if err := c.BindJSON(&converseRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to deserialise request"})
		return
	}
	if len(converseRequest.UserPrompt) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "user prompt must be longer than 2 characters"})
		return
	}

	resp, err := svc.OpenAIClient.CreateChatCompletion(c.Request.Context(), openai.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: converseRequest.SystemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: converseRequest.UserPrompt,
			},
		},
	})
	if err != nil {
		log.Printf("failed to invoke chat completion: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to invoke chat completion"})
		return
	}
	if len(resp.Choices) == 0 {
		log.Printf("failed to invoke chat completion due to empty reply: %+v", resp)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to invoke chat completion"})
		return
	}
	log.Printf("chat completion response: %+v", resp)
	var converseResponse ConverseSinglePromptResponse
	for _, choice := range resp.Choices {
		converseResponse.Reply += choice.Message.Content + " "
	}
	c.JSON(http.StatusOK, converseResponse)
}

// TranscribeRealTimeResponse is the structure of /transcribe-rt/ response.
type TranscribeRealTimeRespons struct {
	Language string `json:"language"`
	Text     string `json:"content"`
}

// handleTranscribeRealTime is a gin handler that uses ChatGPT Whisper API to transcribe the speech in the request body.
func (svc *HttpService) handleTranscribeRealTime(c *gin.Context) {
	if c.ContentType() != "audio/wav" && c.ContentType() != "audio/x-wav" && c.ContentType() != "audio/wave" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "request content type must be wave"})
		return
	}
	wavContent, err := io.ReadAll(c.Request.Body)
	if err != nil || len(wavContent) < 100 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to read request body"})
		return
	}
	// Reference: https://platform.openai.com/docs/api-reference/audio/createTranscription
	resp, err := svc.OpenAIClient.CreateTranscription(c.Request.Context(), openai.AudioRequest{
		Model: "whisper-1",
		// The file path is part of the form submission, the extension name must accurately indicate the audio format.
		FilePath: "input.wav",
		Reader:   bytes.NewReader(wavContent),
		Format:   openai.AudioResponseFormatJSON,
	})
	if err != nil {
		log.Printf("failed to invoke whisper: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to transcribe"})
		return
	}
	transcribeRealTimeRespons := TranscribeRealTimeRespons{
		Language: resp.Language,
		Text:     resp.Text,
	}
	log.Printf("transcription response: %+v", resp)
	c.JSON(http.StatusOK, transcribeRealTimeRespons)
}
