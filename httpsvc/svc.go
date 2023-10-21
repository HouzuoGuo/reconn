package httpsvc

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/re-connect-ai/reconn/db/dbgen"
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
	// LowLevelDB is an initialised low-level sql.DB database client.
	LowLevelDB *sql.DB
	// Database is the high level & strongly typed reconn DB client.
	Database *dbgen.Queries

	// VoiceSampleDir is the path to the directory of incoming user voice samples.
	VoiceSampleDir string
	// VoiceModelDir is path to the directory of constructed user voice models.
	VoiceModelDir string
	// VoiceTempModelDir is the path to the directory of temporary user voice models used during TTS.
	VoiceTempModelDir string
	// VoiceOutputDir is the path to the directory of TTS output files.
	VoiceOutputDir string
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
	// Generic endpoints.
	router.Static("/resource", "./resource")
	router.StaticFile("/", "./resource/index.html")
	if svc.Config.DebugMode {
		router.GET("/api/debug/readback", svc.handleReadback)
		// Debug AI & LLM interaction endpoints.
		router.POST("/api/debug/clone-rt/:user_id", svc.handleRelayCloneRealTime)
		router.POST("/api/debug/tts-rt/:user_id", svc.handleRelayTextToSpeechRealTime)
		router.GET("/api/debug/voice-model", svc.handleListVoiceModel)
		router.POST("/api/debug/converse-single-prompt", svc.handleConverseSinglePrompt)
		router.POST("/api/debug/transcribe-rt", svc.handleTranscribeRealTime)
		// Debug user endpoints.
		router.POST("/api/debug/user", svc.handleCreateUser)
		router.GET("/api/debug/user", svc.handleListUsers)
		// Debug AI person endpoints.
		router.POST("/api/debug/ai_person", svc.handleCreateAIPerson)
		router.GET("/api/debug/user/:user_id/ai_person", svc.handleListAIPersons)
		router.PUT("/api/debug/ai_person/:ai_person_id", svc.handleUpdateAIPerson)
		// Debug voice sample and model endpoints.
		router.POST("/api/debug/ai_person/:ai_person_id/voice_sample", svc.handleCreateVoiceSample)
		router.GET("/api/debug/ai_person/:ai_person_id/voice_sample", svc.handleListVoiceSamples)
		router.GET("/api/debug/ai_person/:ai_person_id/latest_model", svc.handleGetLatestVoiceModel)
		router.POST("/api/debug/voice_sample/:voice_sample_id/create_model", svc.handleCreateVoiceModel)
		// Debug conversations.
		router.POST("/api/debug/ai_person/:ai_person_id/post_text_message", svc.handlePostTextMessage)
		router.POST("/api/debug/ai_person/:ai_person_id/post_voice_message", svc.handlePostVoiceMessage)
		router.GET("/api/debug/ai_person/:ai_person_id/conversation", svc.handleGetAIPersonConversation)
		router.GET("/api/debug/voice_output_file/:file_name", svc.handleGetVoiceOutputFile)
	}
	return router
}
