package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/HouzuoGuo/reconn-voice-clone/db"
	"github.com/HouzuoGuo/reconn-voice-clone/httpsvc"
	"github.com/HouzuoGuo/reconn-voice-clone/workersvc"
)

func main() {
	var httpDebugMode, gpuWorkerMode bool

	var port int
	var addr string
	var tlsCert, tlsKey string

	var basicAuthUser, basicAuthPassword string
	var voiceServiceAddr, openaiKey string
	var dbConf db.Config
	var voiceSampleDir, voiceModelDir, voiceTempModelDir, voiceOutputDir string

	var azBlobConnString, azVoiceSampleContainer, azVoiceModelContainer, azVoiceOutputContainer string
	var azServiceBusConnString, azServiceBusQueue string

	flag.BoolVar(&httpDebugMode, "debug", false, "start http server in debug mode")
	flag.BoolVar(&gpuWorkerMode, "gpuworker", false, "start as GPU worker instead of an http server")

	flag.IntVar(&port, "port", 8080, "web server listener port")
	flag.StringVar(&addr, "addr", "0.0.0.0", "http server listener address")
	flag.StringVar(&basicAuthUser, "authuser", "", "http basic auth user name")
	flag.StringVar(&basicAuthPassword, "authpass", "", "http basic auth user name")

	flag.StringVar(&tlsCert, "tlscert", "", "tls certificate file path")
	flag.StringVar(&tlsKey, "tlskey", "", "tls certificate key path")

	flag.StringVar(&voiceServiceAddr, "voicesvcaddr", "localhost:8081", "voice service address (host:port)")
	flag.StringVar(&openaiKey, "openaikey", "", "openai API secret key")

	flag.StringVar(&dbConf.Host, "dbhost", "", "postgresql database host name")
	flag.IntVar(&dbConf.Port, "dbport", 5432, "postgresql database port")
	flag.StringVar(&dbConf.User, "dbuser", "", "postgresql database user name")
	flag.StringVar(&dbConf.Password, "dbpassword", "", "postgresql database password")
	flag.StringVar(&dbConf.Database, "dbname", "", "postgresql database password")

	flag.StringVar(&voiceSampleDir, "voicesampledir", "/tmp/voice_sample_dir", "path to the directory of incoming user voice samples")
	flag.StringVar(&voiceModelDir, "voicemodeldir", "/tmp/voice_model_dir", "path to the directory of constructed user voice models")
	flag.StringVar(&voiceTempModelDir, "voicetempmodeldir", "/tmp/voice_temp_model_dir", "path to the directory of temporary user voice models used during TTS")
	flag.StringVar(&voiceOutputDir, "voiceoutputdir", "/tmp/voice_output_dir", "path to the directory of TTS output files")

	flag.StringVar(&azBlobConnString, "azblobconnstr", ``, "azure storage connections tring")
	flag.StringVar(&azVoiceSampleContainer, "azvoicecontainer", "voice-sample", "azure storage voice sample container name")
	flag.StringVar(&azVoiceModelContainer, "azmodelcontainer", "voice-model", "azure storage voice model container name")
	flag.StringVar(&azVoiceOutputContainer, "azvoiceoutcontainer", "voice-output", "azure storage voice output container name")

	flag.StringVar(&azServiceBusConnString, "azsvcbusconnstr", ``, "azure service bus connection string")
	flag.StringVar(&azServiceBusQueue, "azsvcbusqueue", ``, "azure service bus queue name")

	flag.Parse()

	if gpuWorkerMode {
		log.Printf("about to start GPU worker for service bus queue %q", azServiceBusQueue)
		workerConf := &workersvc.Config{
			VoiceServiceAddr: voiceServiceAddr,

			Database: dbConf,

			BlobConnectionString: azBlobConnString,
			ServiceBusQueue:      azServiceBusQueue,
			ServiceBusConnection: azServiceBusConnString,

			VoiceSampleDir:       voiceSampleDir,
			VoiceModelDir:        voiceModelDir,
			VoiceTempModelDir:    voiceTempModelDir,
			VoiceOutputDir:       voiceOutputDir,
			VoiceSampleContainer: azVoiceSampleContainer,
			VoiceModelContainer:  azVoiceModelContainer,
			VoiceOutputContainer: azVoiceOutputContainer,
		}
		startGPUWorker(workerConf)
	} else {
		log.Printf("about to start web service on port %d, connect to backend voice service at %q, debug mode? %v, using http basic auth? %v", port, voiceServiceAddr, httpDebugMode, basicAuthUser != "")
		httpConf := &httpsvc.Config{
			DebugMode:        httpDebugMode,
			VoiceServiceAddr: voiceServiceAddr,
			OpenAIKey:        openaiKey,

			BasicAuthUser:     basicAuthUser,
			BasicAuthPassword: basicAuthPassword,

			Database: dbConf,

			VoiceSampleDir:       voiceSampleDir,
			VoiceModelDir:        voiceModelDir,
			VoiceTempModelDir:    voiceTempModelDir,
			VoiceOutputDir:       voiceOutputDir,
			VoiceSampleContainer: azVoiceSampleContainer,
			VoiceModelContainer:  azVoiceModelContainer,
			VoiceOutputContainer: azVoiceOutputContainer,

			BlobConnectionString: azBlobConnString,
			ServiceBusQueue:      azServiceBusQueue,
			ServiceBusConnection: azServiceBusConnString,
		}
		startHTTPServer(httpConf, addr, port, tlsCert, tlsKey)
	}
}

func startHTTPServer(conf *httpsvc.Config, addr string, port int, tlsCert, tlsKey string) {
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
	if tlsCert == "" {
		log.Fatal(server.ListenAndServe())
	} else {
		log.Fatal(server.ListenAndServeTLS(tlsCert, tlsKey))
	}
}

func startGPUWorker(conf *workersvc.Config) {
	worker, err := workersvc.New(conf)
	if err != nil {
		log.Fatalf("failed to initialise GPU worker service: %v", err)
	}
	log.Fatalf("GPU worker exited: %v", worker.Run())
}
