package httpsvc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
)

// Config has the configuration of the web server itself and its external dependencies.
type Config struct {
	// DebugMode flag indicates that the http service shall run in debug mode for development and testing only.
	DebugMode bool
	// BasicAuthUser is the optional user name of basic authentication used by all handlers.
	BasicAuthUser string
	// BasicAuthUser is the password of basic authentication used by all handlers.
	BasicAuthPassword string
	// VoiceServiceAddr is the address ("host:port") of the voice service (reconn/voicesvc).
	VoiceServiceAddr string
	// OpenAIKey is the API key of OpenAI / ChatGPT.
	OpenAIKey string
	// VoiceModelDir is the directory of constructed user voice models used by the voice service.
	VoiceModelDir string
}

// HttpService implements HTTP handlers for serving static content, relaying to voice service, and more.
type HttpService struct {
	// Config has the configuration of the web server and its external dependencies.
	Config Config
	// VoiceClient is an HTTP client for the voice service (reconn/voicesvc).
	VoiceClient *http.Client
	// OpenAIClient is a ChatGPT client.
	OpenAIClient *openai.Client
}

// New returns an initialised HTTP service.
func New(conf Config) (*HttpService, error) {
	svc := &HttpService{
		Config:       conf,
		OpenAIClient: openai.NewClient(conf.OpenAIKey),
		// The real-time voice service endpoint relays (mainly for development & testing) require a generous amount of timeout.
		VoiceClient: &http.Client{Timeout: 3 * time.Minute},
	}
	return svc, nil
}

// CloneRealTimeResponse is the structure of /clone-rt/ response.
type CloneRealTimeResponse struct {
	// ModelDestinationFile is the relative path of the newly cloned voice model.
	ModelDestinationFile string `json:"model"`
}

// TextToSpeechRealTimeRequest is the structure of /tts-rt/ request.
type TextToSpeechRealTimeRequest struct {
	Text         string  `json:"text"`
	TopK         float64 `json:"topK"`
	TopP         float64 `json:"topP"`
	MineosP      float64 `json:"mineosP"`
	SemanticTemp float64 `json:"semanticTemp"`
	WaveformTemp float64 `json:"waveformTemp"`
	FineTemp     float64 `json:"fineTemp"`
}

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
	wavContent, err := ioutil.ReadAll(c.Request.Body)
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
	var cloneResp CloneRealTimeResponse
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
	var ttsRequest TextToSpeechRealTimeRequest
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
	wavContent, err := ioutil.ReadAll(resp.Body)
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
	entries, err := ioutil.ReadDir(svc.Config.VoiceModelDir)
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
			FileName:     fileName,
			UserID:       userID,
			LastModified: entry.ModTime(),
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
	userID := c.Params.ByName("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "user_id must be present"})
		return
	}
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
	log.Printf("chat completion for user ID %v: %+v", userID, resp)
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
	userID := c.Params.ByName("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "user_id must be present"})
		return
	}
	wavContent, err := ioutil.ReadAll(c.Request.Body)
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
		Format:   openai.AudioResponseFormatText,
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
	log.Printf("transcription for user ID %v: %+v", userID, resp)
	c.JSON(http.StatusOK, transcribeRealTimeRespons)
}

func (svc *HttpService) SetupRouter() *gin.Engine {
	if svc.Config.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s from: %s \"%s\", request: %s %s %s, response: %d in %dus and %v bytes, err: %s\\n",
			param.TimeStamp.Format(time.RFC3339),
			param.ClientIP,
			param.Request.UserAgent(),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency.Microseconds(),
			param.BodySize,
			param.ErrorMessage,
		)
	}))
	router.Use(gin.Recovery())

	if svc.Config.BasicAuthUser != "" {
		router.Use(gin.BasicAuth(gin.Accounts{svc.Config.BasicAuthUser: svc.Config.BasicAuthPassword}))
	}
	// Read back several parameters of the client's request.
	router.GET("/api/readback", svc.handleReadback)
	router.POST("/api/clone-rt/:user_id", svc.handleRelayCloneRealTime)
	router.POST("/api/tts-rt/:user_id", svc.handleRelayTextToSpeechRealTime)
	router.GET("/api/voice-model", svc.handleListVoiceModel)
	router.POST("/api/converse-single-prompt/:user_id", svc.handleConverseSinglePrompt)
	router.POST("/api/transcribe-rt/:user_id", svc.handleTranscribeRealTime)
	router.Static("/resource", "./resource")
	router.StaticFile("/", "./resource/index.html")
	return router
}
