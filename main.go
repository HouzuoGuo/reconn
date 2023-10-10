package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/re-connect-ai/reconn/httpsvc"
)

func main() {
	var port int
	var debugMode bool
	var addr, voiceServiceAddr, openaiKey, basicAuthUser, basicAuthPassword, voiceModelDir string
	var dbAddress, dbUser, dbPassword string
	flag.BoolVar(&debugMode, "debug", false, "start http server in debug mode")
	flag.IntVar(&port, "port", 8080, "http server listener port")
	flag.StringVar(&addr, "addr", "0.0.0.0", "http server listener address")
	flag.StringVar(&basicAuthUser, "authuser", "reconn", "http basic auth user name")
	flag.StringVar(&basicAuthPassword, "authpass", "reconnreconn", "http basic auth user name")
	flag.StringVar(&voiceServiceAddr, "voicesvcaddr", "localhost:8081", "voice service address (host:port)")
	flag.StringVar(&openaiKey, "openaikey", "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", "openai API secret key")
	flag.StringVar(&voiceModelDir, "voicemodeldir", "/tmp/voice_model_dir", "the directory of constructed user voice models used by the voice service")

	flag.StringVar(&dbAddress, "dbaddress", "reconn-user-db.postgres.database.azure.com", "postgresql database host name")
	flag.StringVar(&dbUser, "dbuser", "reconnadmin", "postgresql database user name")
	flag.StringVar(&dbPassword, "dbpassword", "BOInscINOnioVc2RK", "postgresql database password")
	flag.Parse()

	log.Printf("about to start web service on port %d, connect to backend voice service at %q, debug mode? %v, using http basic auth? %v", port, voiceServiceAddr, debugMode, basicAuthUser != "")
	httpService, err := httpsvc.New(httpsvc.Config{
		DebugMode:         debugMode,
		VoiceServiceAddr:  voiceServiceAddr,
		OpenAIKey:         openaiKey,
		BasicAuthUser:     basicAuthUser,
		BasicAuthPassword: basicAuthPassword,
		VoiceModelDir:     voiceModelDir,
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
