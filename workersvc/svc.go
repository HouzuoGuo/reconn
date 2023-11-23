package workersvc

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/HouzuoGuo/reconn-voice-clone/db"
	"github.com/HouzuoGuo/reconn-voice-clone/db/dbgen"
	"github.com/HouzuoGuo/reconn-voice-clone/shared"
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
			log.Printf("received message: %v", string(msg.Body))
			var gpuTask shared.GPUTask
			if err := json.Unmarshal(msg.Body, &gpuTask); err != nil {
				log.Printf("failed to unmarshal message body %q: %v", string(msg.Body), err)
			}
			worker.Process(gpuTask)
			if err := worker.ServiceBusReceiver.CompleteMessage(context.Background(), msg, nil); err != nil {
				log.Printf("failed to complete message: %v", err)
			}
		}
	}
}

func (worker *GPUWorker) Process(task shared.GPUTask) {
	if task.VoiceModelID > 0 {
		worker.createVoiceModel(context.Background(), task)
	} else if task.AIReplyVoiceID > 0 {
		worker.convertReplyToSpeech(context.Background(), task)
	}
}

func (worker *GPUWorker) createVoiceModel(ctx context.Context, task shared.GPUTask) {
	voiceModelID := task.VoiceModelID
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
	// Store the voice model in blob storage.
	if err := shared.UploadFromLocalFile(ctx, worker.BlobClient, worker.Config.VoiceModelContainer, cloneResp.ModelDestinationFile, worker.Config.VoiceModelDir); err != nil {
		log.Printf("upload from local file error: %+v", err)
		return
	}
	// Update the voice model record in database.
	err = worker.Database.UpdateVoiceModelByID(ctx, dbgen.UpdateVoiceModelByIDParams{
		ID:       int64(voiceModelID),
		Status:   "ready",
		FileName: sql.NullString{String: cloneResp.ModelDestinationFile, Valid: true},
	})
	if err != nil {
		log.Printf("update voice model by id error: %+v", err)
		return
	}
}

func (worker *GPUWorker) convertReplyToSpeech(ctx context.Context, task shared.GPUTask) {
	aiReplyVoiceID := task.AIReplyVoiceID
	wipReplyVoice, err := worker.Database.GetAIPersonReplyVoiceByID(ctx, int64(aiReplyVoiceID))
	if err != nil {
		log.Printf("failed to get reply voice by id: %v", err)
		return
	}
	// Retrieve the reply content record from database.
	aiReply, err := worker.Database.GetAIPersonReplyByID(ctx, wipReplyVoice.AiPersonReplyID)
	if err != nil {
		log.Printf("get voice sample by id error: %+v", err)
		return
	}
	// Read the voice model and context prompt from this AI person.
	aiPersonAndModel, err := worker.Database.GetLatestVoiceModel(ctx, int64(task.AIReplyPersonID))
	if err != nil {
		log.Printf("get latest voice model error: %+v", err)
		return
	}
	log.Printf("ai person and model: %+v", aiPersonAndModel)
	// Download the model file to local disk and then relay to python voice server.
	if _, err := shared.DownloadBlobToLocalFileIfNotExist(ctx, worker.BlobClient, worker.Config.VoiceModelContainer, aiPersonAndModel.FileName.String, worker.Config.VoiceModelDir); err != nil {
		log.Printf("download model error: %v", err)
		return
	}
	// Convert the reply into voice.
	ttsRequestBody, err := json.Marshal(shared.TextToSpeechRealTimeRequest{
		Text:         aiReply.Message,
		TopK:         99,
		TopP:         0.8,
		MineosP:      0.01,
		SemanticTemp: 0.8,
		WaveformTemp: 0.6,
		FineTemp:     0.5,
	})
	if err != nil {
		log.Printf("tts request construction error: %v", err)
		return
	}
	ttsRequest, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/tts-rt/%s", worker.Config.VoiceServiceAddr, strings.TrimSuffix(aiPersonAndModel.FileName.String, ".npz")), bytes.NewReader(ttsRequestBody))
	ttsRequest.Header.Set("content-type", "application/json")
	if err != nil {
		log.Printf("tts request construction error: %v", err)
		return
	}
	ttsResponse, err := worker.VoiceClient.Do(ttsRequest)
	if err != nil {
		log.Printf("tts request error: %v", err)
		return
	}
	log.Printf("tts-rt responded with status %d and content length %d", ttsResponse.StatusCode, ttsResponse.ContentLength)
	ttsWaveContent, err := io.ReadAll(ttsResponse.Body)
	if err != nil {
		log.Printf("failed to read tts response body: %v", err)
		return
	}
	// Save the converted speech.
	timestamp := time.Now()
	fileName := fmt.Sprintf("%d-%s.wav", task.AIReplyPersonID, timestamp.Format(time.RFC3339))
	if _, err := shared.UploadAndSave(ctx, worker.BlobClient, worker.Config.VoiceOutputContainer, fileName, worker.Config.VoiceOutputDir, ttsWaveContent); err != nil {
		log.Printf("upload and save error: %v", err)
		return
	}
	// Update the AI reply voice record.
	err = worker.Database.UpdateAIPersonReplyVoiceStatusByID(ctx, dbgen.UpdateAIPersonReplyVoiceStatusByIDParams{
		ID:       int64(aiReplyVoiceID),
		Status:   "ready",
		FileName: sql.NullString{String: fileName, Valid: true},
	})
	if err != nil {
		log.Printf("update ai person reply voice status by ID error: %v", err)
		return
	}
}
