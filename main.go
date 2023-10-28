package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/re-connect-ai/reconn/db"
	"github.com/re-connect-ai/reconn/httpsvc"
	"github.com/re-connect-ai/reconn/workersvc"
)

func main() {
	var httpDebugMode, gpuWorkerMode bool
	var port int
	var addr string
	var basicAuthUser, basicAuthPassword string
	var voiceServiceAddr, openaiKey string
	var dbConf db.Config
	var voiceSampleDir, voiceModelDir, voiceTempModelDir, voiceOutputDir string

	var azBlobConnString, azVoiceSampleContainer, azVoiceModelContainer, azVoiceOutputContainer string
	var azServiceBusConnString, azServiceBusQueue string

	flag.BoolVar(&httpDebugMode, "debug", false, "start http server in debug mode")
	flag.BoolVar(&gpuWorkerMode, "gpuworker", false, "start as GPU worker instead of an http server")

	flag.IntVar(&port, "port", 8080, "http server listener port")
	flag.StringVar(&addr, "addr", "0.0.0.0", "http server listener address")
	flag.StringVar(&basicAuthUser, "authuser", "reconn", "http basic auth user name")
	flag.StringVar(&basicAuthPassword, "authpass", "reconnreconn", "http basic auth user name")

	flag.StringVar(&voiceServiceAddr, "voicesvcaddr", "localhost:8081", "voice service address (host:port)")
	flag.StringVar(&openaiKey, "openaikey", "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", "openai API secret key")

	flag.StringVar(&dbConf.Host, "dbhost", "reconn-user-db.postgres.database.azure.com", "postgresql database host name")
	flag.IntVar(&dbConf.Port, "dbport", 5432, "postgresql database port")
	flag.StringVar(&dbConf.User, "dbuser", "reconnadmin", "postgresql database user name")
	flag.StringVar(&dbConf.Password, "dbpassword", "BOInscINOnioVc2RK", "postgresql database password")
	flag.StringVar(&dbConf.Database, "dbname", "reconn", "postgresql database password")

	flag.StringVar(&voiceSampleDir, "voicesampledir", "/tmp/voice_sample_dir", "path to the directory of incoming user voice samples")
	flag.StringVar(&voiceModelDir, "voicemodeldir", "/tmp/voice_model_dir", "path to the directory of constructed user voice models")
	flag.StringVar(&voiceTempModelDir, "voicetempmodeldir", "/tmp/voice_temp_model_dir", "path to the directory of temporary user voice models used during TTS")
	flag.StringVar(&voiceOutputDir, "voiceoutputdir", "/tmp/voice_output_dir", "path to the directory of TTS output files")

	flag.StringVar(&azBlobConnString, "azblobconnstr", `BlobEndpoint=https://reconn0store.blob.core.windows.net/;QueueEndpoint=https://reconn0store.queue.core.windows.net/;FileEndpoint=https://reconn0store.file.core.windows.net/;TableEndpoint=https://reconn0store.table.core.windows.net/;SharedAccessSignature=sv=2022-11-02&ss=bfqt&srt=sco&sp=rwdlacupiytfx&se=2040-10-23T15:58:46Z&st=2023-10-23T07:58:46Z&spr=https&sig=eb%2Fj8eNXyY3qfnyDhLX7wMft0sa13ICSf9wxQgWZ7WE%3D`, "azure storage connections tring")
	flag.StringVar(&azVoiceSampleContainer, "azvoicecontainer", "voice-sample", "azure storage voice sample container name")
	flag.StringVar(&azVoiceModelContainer, "azmodelcontainer", "voice-model", "azure storage voice model container name")
	flag.StringVar(&azVoiceOutputContainer, "azvoiceoutcontainer", "voice-output", "azure storage voice output container name")

	flag.StringVar(&azServiceBusConnString, "azsvcbusconnstr", `Endpoint=sb://reconn-gpu-task.servicebus.windows.net/;SharedAccessKeyName=go-client;SharedAccessKey=ZMnC8xwMCI6oM/hCzMKN/QZrFgbz1Wuzy+ASbCgFOec=;EntityPath=all-in-one`, "azure service bus connection string")
	flag.StringVar(&azServiceBusQueue, "azsvcbusqueue", `all-in-one`, "azure service bus queue name")

	flag.Parse()

	// Connect to DB.
	lowLevelDB, reconnDB, err := db.Connect(dbConf)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
		return
	}
	log.Printf("successfully connected to database %v:%v, stats: %+v", dbConf.Host, dbConf.Port, lowLevelDB.Stats())

	// Connect to Azure blob storage.
	blobClient, err := azblob.NewClientFromConnectionString(azBlobConnString, nil)
	if err != nil {
		log.Fatalf("failed to connect to azure blob storage: %v", err)
		return
	}
	blobProps, err := blobClient.ServiceClient().GetProperties(context.Background(), nil)
	if err != nil {
		log.Fatalf("failed to connect to azure blob storage: %v", err)
		return
	}
	log.Printf("successfully connected to azure storage (err? %v), %+#v", err, blobProps)

	// Connect to azure service bus.
	svcBusClient, err := azservicebus.NewClientFromConnectionString(azServiceBusConnString, nil)
	if err != nil {
		log.Fatalf("failed to connect to azure service bus: %v", err)
		return
	}
	if gpuWorkerMode {
		log.Printf("about to start GPU worker for service bus queue %q", azServiceBusQueue)
		svcBusReceiver, err := svcBusClient.NewReceiverForQueue(azServiceBusQueue, nil)
		if err != nil {
			log.Fatalf("failed to connect to azure service bus: %v", err)
			return
		}
		workerConf := &workersvc.Config{
			ServiceBusQueue:    azServiceBusQueue,
			ServiceBusClient:   svcBusClient,
			ServiceBusReceiver: svcBusReceiver,
		}
		startGPUWorker(workerConf)
	} else {
		log.Printf("about to start web service on port %d, connect to backend voice service at %q, debug mode? %v, using http basic auth? %v", port, voiceServiceAddr, httpDebugMode, basicAuthUser != "")
		svcBusSender, err := svcBusClient.NewSender(azServiceBusQueue, nil)
		if err != nil {
			log.Fatalf("failed to connect to azure service bus: %v", err)
			return
		}
		httpConf := &httpsvc.Config{
			DebugMode: httpDebugMode,

			VoiceServiceAddr: voiceServiceAddr,
			OpenAIKey:        openaiKey,

			BasicAuthUser:     basicAuthUser,
			BasicAuthPassword: basicAuthPassword,

			LowLevelDB: lowLevelDB,
			Database:   reconnDB,

			VoiceSampleDir:    voiceSampleDir,
			VoiceModelDir:     voiceModelDir,
			VoiceTempModelDir: voiceTempModelDir,
			VoiceOutputDir:    voiceOutputDir,

			BlobClient:           blobClient,
			VoiceSampleContainer: azVoiceSampleContainer,
			VoiceModelContainer:  azVoiceModelContainer,
			VoiceOutputContainer: azVoiceOutputContainer,

			ServiceBusQueue:  azServiceBusQueue,
			ServiceBusClient: svcBusClient,
			ServiceBusSender: svcBusSender,
		}
		startHTTPServer(httpConf, addr, port)
	}
}

func startHTTPServer(conf *httpsvc.Config, addr string, port int) {
	httpService, err := httpsvc.New(conf)
	if err != nil {
		log.Fatalf("failed to initialise http service: %v", err)
	}
	server := &http.Server{
		// The real-time voice service endpoint relays (mainly for development & testing) require a generous amount of timeout.
		ReadTimeout:       5 * time.Minute,
		ReadHeaderTimeout: 5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		MaxHeaderBytes:    1024 * 1024,
		Handler:           httpService.SetupRouter(),
		Addr:              net.JoinHostPort(addr, strconv.Itoa(port)),
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func startGPUWorker(conf *workersvc.Config) {
}
