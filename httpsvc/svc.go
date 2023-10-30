package httpsvc

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
	"github.com/re-connect-ai/reconn/db"
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

	// Database configuration.
	Database db.Config

	// VoiceSampleDir is the path to the directory of incoming user voice samples.
	VoiceSampleDir string
	// VoiceModelDir is path to the directory of constructed user voice models.
	VoiceModelDir string
	// VoiceTempModelDir is the path to the directory of temporary user voice models used during TTS.
	VoiceTempModelDir string
	// VoiceOutputDir is the path to the directory of TTS output files.
	VoiceOutputDir string

	// VoiceSampleContainer is the blob container name of the voice samples.
	VoiceSampleContainer string
	// VoiceSampleContainer is the blob container name of the voice models.
	VoiceModelContainer string
	// VoiceSampleContainer is the blob container name of the voice output files.
	VoiceOutputContainer string

	// BlobConnectionString is the azure sas connection string of blob storage.
	BlobConnectionString string
	// ServiceBusConnectionString  the azure sas connection string of service bus.
	ServiceBusConnection string

	// ServiceBusQueue is the name of azure service bus queue.
	ServiceBusQueue string
}

// HttpService implements HTTP handlers for serving static content, relaying to voice service, and more.
type HttpService struct {
	// Config has the configuration of the web server and its external dependencies.
	Config *Config
	// VoiceClient is an HTTP client for the voice service (reconn/voicesvc).
	VoiceClient *http.Client
	// OpenAIClient is a ChatGPT client.
	OpenAIClient *openai.Client

	// LowLevelDB is an initialised low-level sql.DB database client.
	LowLevelDB *sql.DB
	// Database is the high level & strongly typed reconn DB client.
	Database *dbgen.Queries
	// BlobClient is the azure blob storage client.
	BlobClient *azblob.Client
	// ServiceBusClient is the azure service bus client.
	ServiceBusClient *azservicebus.Client
	// ServiceBusClient is the azure service bus sender client.
	ServiceBusSender *azservicebus.Sender
}

// New returns an initialised HTTP service.
func New(conf *Config) (*HttpService, error) {
	svc := &HttpService{
		Config:       conf,
		OpenAIClient: openai.NewClient(conf.OpenAIKey),
		// The real-time voice service endpoint relays (mainly for development & testing) require a generous amount of timeout.
		VoiceClient: &http.Client{Timeout: 5 * time.Minute},
	}
	// Connect to DB.
	var err error
	svc.LowLevelDB, svc.Database, err = db.Connect(conf.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
		return nil, err
	}
	log.Printf("successfully connected to database %v:%v, stats: %+v", conf.Database.Host, conf.Database.Port, svc.LowLevelDB.Stats())
	// Connect to Azure blob storage.
	svc.BlobClient, err = azblob.NewClientFromConnectionString(conf.BlobConnectionString, nil)
	if err != nil {
		log.Fatalf("failed to connect to azure blob storage: %v", err)
		return nil, err
	}
	blobProps, err := svc.BlobClient.ServiceClient().GetProperties(context.Background(), nil)
	if err != nil {
		log.Fatalf("failed to connect to azure blob storage: %v", err)
		return nil, err
	}
	log.Printf("successfully connected to azure storage (err? %v), %+#v", err, blobProps)
	// Connect to azure service bus.
	svc.ServiceBusClient, err = azservicebus.NewClientFromConnectionString(conf.ServiceBusConnection, nil)
	if err != nil {
		log.Fatalf("failed to connect to azure service bus: %v", err)
		return nil, err
	}
	svc.ServiceBusSender, err = svc.ServiceBusClient.NewSender(conf.ServiceBusQueue, nil)
	if err != nil {
		log.Fatalf("failed to connect to azure service bus: %v", err)
		return nil, err
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
		// Use GPU-enabled workers for asynchronous processing.
		router.POST("/api/debug/voice_sample/:voice_sample_id/create_model_async", svc.handleCreateVoiceModelAsync)
		router.POST("/api/debug/ai_person/:ai_person_id/post_text_message_async", svc.handlePostTextMessageAsync)
		router.POST("/api/debug/ai_person/:ai_person_id/post_voice_message_async", svc.handlePostVoiceMessageAsync)
	}
	return router
}
