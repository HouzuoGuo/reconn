package workersvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/re-connect-ai/reconn/db"
	"github.com/re-connect-ai/reconn/db/dbgen"
	"github.com/re-connect-ai/reconn/shared"
)

// Config has the configuration of the GPU worker service itself and its external dependencies.
type Config struct {
	// Database configuration.
	Database db.Config
	// ServiceBusConnectionString  the azure sas connection string of service bus.
	ServiceBusConnection string
	// ServiceBusQueue is the name of azure service bus queue.
	ServiceBusQueue string
	// BlobConnectionString is the azure sas connection string of blob storage.
	BlobConnectionString string

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

	// VoiceServiceAddr is the address ("host:port") of the voice service (reconn/voicesvc).
	VoiceServiceAddr string
}

type GPUWorker struct {
	// Config has the GPU worker configuration and its external dependencies.
	Config *Config
	// VoiceClient is an HTTP client for the voice service (reconn/voicesvc).
	VoiceClient *http.Client
	// LowLevelDB is an initialised low-level sql.DB database client.
	LowLevelDB *sql.DB
	// Database is the high level & strongly typed reconn DB client.
	Database *dbgen.Queries
	// BlobClient is the azure blob storage client.
	BlobClient *azblob.Client
	// ServiceBusClient is the azure service bus client.
	ServiceBusClient *azservicebus.Client
	// ServiceBusSender is the azure service bus receiver client.
	ServiceBusReceiver *azservicebus.Receiver
}

// New returns a newly initialised instance of the GPU worker service.
func New(conf *Config) (*GPUWorker, error) {
	worker := &GPUWorker{
		Config: conf,
		// The real-time voice service endpoint relays (mainly for development & testing) require a generous amount of timeout.
		VoiceClient: &http.Client{Timeout: 5 * time.Minute},
	}
	var err error
	// Connect to DB.
	worker.LowLevelDB, worker.Database, err = db.Connect(conf.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
		return nil, err
	}
	log.Printf("successfully connected to database %v:%v, stats: %+v", conf.Database.Host, conf.Database.Port, worker.LowLevelDB.Stats())
	// Connect to Azure blob storage.
	worker.BlobClient, err = azblob.NewClientFromConnectionString(conf.BlobConnectionString, nil)
	if err != nil {
		log.Fatalf("failed to connect to azure blob storage: %v", err)
		return nil, err
	}
	blobProps, err := worker.BlobClient.ServiceClient().GetProperties(context.Background(), nil)
	if err != nil {
		log.Fatalf("failed to connect to azure blob storage: %v", err)
		return nil, err
	}
	log.Printf("successfully connected to azure storage (err? %v), %+#v", err, blobProps)
	// Connect to azure service bus.
	worker.ServiceBusClient, err = azservicebus.NewClientFromConnectionString(conf.ServiceBusConnection, nil)
	if err != nil {
		log.Fatalf("failed to connect to azure service bus: %v", err)
		return nil, err
	}
	// Connect to azure service bus.
	worker.ServiceBusReceiver, err = worker.ServiceBusClient.NewReceiverForQueue(conf.ServiceBusQueue, nil)
	if err != nil {
		log.Fatalf("failed to connect to azure service bus: %v", err)
		return nil, err
	}
	return worker, nil
}

func (worker *GPUWorker) Run() error {
	for {
		log.Printf("waiting for the next message")
		messages, err := worker.ServiceBusReceiver.ReceiveMessages(context.Background(), 1, nil)
		if err != nil {
			log.Printf("failed to receive messages: %v", err)
			return err
		}
		for _, msg := range messages {
			var gpuTask shared.GPUTask
			if err := json.Unmarshal(msg.Body, &gpuTask); err != nil {
				log.Printf("failed to unmarshal message body %q: %v", string(msg.Body), err)
			}
			worker.Process(gpuTask)
		}
		log.Printf("received messages: %+v", messages)
	}
}

func (worker *GPUWorker) Process(task shared.GPUTask) {
	if task.VoiceModelID > 0 {
		worker.createVoiceModel(context.Background(), task.VoiceModelID)
	} else if task.AIReplyVoiceID > 0 {
		worker.convertReplyToSpeech(context.Background(), task.AIReplyVoiceID)
	}
}

func (worker *GPUWorker) createVoiceModel(ctx context.Context, voiceModelID int) {
	wipModel, err := worker.Database.GetVoiceModelByID(ctx, int64(voiceModelID))
	if err != nil {
		log.Printf("failed to get voice model by id: %v", err)
		return
	}
	// Retrieve the sample record from database.
	voiceSample, err := worker.Database.GetVoiceSampleByID(ctx, wipModel.VoiceSampleID)
	if err != nil {
		log.Printf("get voice sample by id error: %+v", err)
		return
	}
	// Retrieve the sample wave file from blob storage.
	localFilePath, err := shared.DownloadBlobToLocalFileIfNotExist(ctx, worker.BlobClient, worker.Config.VoiceSampleContainer, voiceSample.FileName.String, worker.Config.VoiceSampleDir)
	if err != nil {
		log.Printf("blob download file error: %v", err)
		return
	}
	// Open the wave file from local disk.
	voiceSampleFile, err := os.Open(localFilePath)
	defer voiceSampleFile.Close()
	if err != nil {
		return
	}
	// Relay the clone request to voice service.
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/clone-rt/%d", worker.Config.VoiceServiceAddr, voiceModelID), voiceSampleFile)
	req.Header.Set("content-type", "audio/wav")
	if err != nil {
		log.Printf("failed to construct clone-rt request: %v", err)
		return
	}
	resp, err := worker.VoiceClient.Do(req)
	if err != nil {
		log.Printf("failed to make clone-rt request: %v", err)
		return
	}
	log.Printf("clone-rt responded with status %d and content length %d", resp.StatusCode, resp.ContentLength)
	var cloneResp shared.CloneRealTimeResponse
	if err := json.NewDecoder(resp.Body).Decode(&cloneResp); err != nil {
		log.Printf("failed to deserialise clone-rt response: %v", err)
		return
	}
}

func (worker *GPUWorker) convertReplyToSpeech(ctx context.Context, aiReplyVoiceID int) {
}
