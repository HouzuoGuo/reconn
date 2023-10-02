package httpsvc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
)

// Config has the configuration of the web server itself and its external dependencies.
type Config struct {
	// BasicAuthUser is the optional user name of basic authentication used by all handlers.
	BasicAuthUser string
	// BasicAuthUser is the password of basic authentication used by all handlers.
	BasicAuthPassword string
	// VoiceServiceAddr is the address ("host:port") of the voice service (reconn/voicesvc).
	VoiceServiceAddr string
	// OpenAIKey is the API key of OpenAI / ChatGPT.
	OpenAIKey string
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
	// Text is the text to be converted into speech.
	Text string `json:"text"`
}

// handleReadback is a gin handler that reads back several parameters from the request.
// This is often used for testing.
func (svc *HttpService) handleReadback(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"address": c.Request.RemoteAddr,
		"headers": c.Request.Header,
		"method":  c.Request.Method,
		"url":     c.Request.URL.String(),
	})
}

// handleRelayCloneRealTime is a gin handler that relays a real time voice cloning request to the voice service.
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

func (svc *HttpService) SetupRouter() *gin.Engine {
	router := gin.Default()
	if svc.Config.BasicAuthUser != "" {
		router.Use(gin.BasicAuth(gin.Accounts{svc.Config.BasicAuthUser: svc.Config.BasicAuthPassword}))
	}
	// Read back several parameters of the client's request.
	router.GET("/api/readback", svc.handleReadback)
	router.POST("/api/clone-rt/:user_id", svc.handleRelayCloneRealTime)
	router.POST("/api/tts-rt/:user_id", svc.handleRelayTextToSpeechRealTime)
	router.Static("/resource", "./resource")
	router.StaticFile("/", "./resource/index.html")
	return router
}
