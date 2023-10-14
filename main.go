package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/re-connect-ai/reconn/db"
	"github.com/re-connect-ai/reconn/httpsvc"
)

func main() {
	var port int
	var debugMode bool
	var addr, voiceServiceAddr, openaiKey, basicAuthUser, basicAuthPassword, voiceModelDir string
	var dbConf db.Config
	flag.BoolVar(&debugMode, "debug", false, "start http server in debug mode")
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

	// The voice model directory is used by a developer-exlcusive endpoint. Not for use in production.
	flag.StringVar(&voiceModelDir, "voicemodeldir", "/tmp/voice_model_dir", "the directory of constructed user voice models used by the voice service")
	flag.Parse()

	log.Printf("about to start web service on port %d, connect to backend voice service at %q, debug mode? %v, using http basic auth? %v", port, voiceServiceAddr, debugMode, basicAuthUser != "")

	lowLevelDB, reconnDB, err := db.Connect(dbConf)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	httpService, err := httpsvc.New(httpsvc.Config{
		DebugMode:         debugMode,
		VoiceServiceAddr:  voiceServiceAddr,
		OpenAIKey:         openaiKey,
		BasicAuthUser:     basicAuthUser,
		BasicAuthPassword: basicAuthPassword,
		VoiceModelDir:     voiceModelDir,
		LowLevelDB:        lowLevelDB,
		Database:          reconnDB,
	})
	if err != nil {
		log.Fatalf("failed to initialise http service: %v", err)
	}
	server := &http.Server{
		// The real-time voice service endpoint relays (mainly for development & testing) require a generous amount of timeout.
		ReadTimeout:       3 * time.Minute,
		ReadHeaderTimeout: 3 * time.Minute,
		WriteTimeout:      3 * time.Minute,
		MaxHeaderBytes:    1024 * 1024,
		Handler:           httpService.SetupRouter(),
		Addr:              net.JoinHostPort(addr, strconv.Itoa(port)),
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
