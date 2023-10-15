package httpsvc

import (
	"bytes"
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
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	textPrompt, err := svc.Config.Database.CreateUserTextPrompt(c.Request.Context(), dbgen.CreateUserTextPromptParams{
		UserPromptID: prompt.ID,
		Message:      req.Message,
	})
	log.Printf("prompt: %+v, text prompt: %+v", prompt, textPrompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	// Read the voice model and context prompt from this AI person.
	aiPersonAndModel, err := svc.Config.Database.GetLatestVoiceModel(c.Request.Context(), int64(aiPersonID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("ai person and model: %+v", aiPersonAndModel)
	// Feed both context prompt and text prompt to LLM.
	resp, err := svc.OpenAIClient.CreateChatCompletion(c.Request.Context(), openai.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: aiPersonAndModel.AiContextPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: req.Message,
			},
		},
	})
	if err != nil {
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
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	// Save the converted speech.
	fileName := fmt.Sprintf("%d-%s.wav", aiPersonID, timestamp.Format(time.RFC3339))
	if err := ioutil.WriteFile(path.Join(svc.Config.VoiceOutputDir, fileName), ttsWaveContent, 0644); err != nil {
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
	// aiPersonID, _ := strconv.Atoi(c.Params.ByName("ai_person_id"))
}

// handleGetAIPersonConversation is a gin handler that returns the full conversation going back and forth between a user and an AI person.
func (svc *HttpService) handleGetAIPersonConversation(c *gin.Context) {
	aiPersonID, _ := strconv.Atoi(c.Params.ByName("ai_person_id"))
	conversations, err := svc.Config.Database.ListConversations(c.Request.Context(), int64(aiPersonID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, conversations)
}

// handleGetVoiceOutputFile returns the waveform file content of the requested file name.
func (svc *HttpService) handleGetVoiceOutputFile(c *gin.Context) {
	fileName := path.Base(c.Params.ByName("file_name"))
	fullPath := path.Join(svc.Config.VoiceOutputDir, fileName)
	// Read file info for the content length.
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	fileContent, err := os.Open(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.DataFromReader(http.StatusOK, fileInfo.Size(), "audio/wav", fileContent, nil)
}