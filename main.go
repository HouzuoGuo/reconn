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
	var voiceServiceAddr, openaiKey string
	flag.IntVar(&port, "port", 8080, "http server port")
	flag.StringVar(&voiceServiceAddr, "voicesvcaddr", "localhost:8081", "voice service address (host:port)")
	flag.StringVar(&openaiKey, "openaikey", "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", "openai API secret key")
	flag.Parse()

	log.Printf("about to start on port %d", port)
	httpService, err := httpsvc.New(httpsvc.Config{
		VoiceServiceAddr: voiceServiceAddr,
		OpenAIKey:        openaiKey,
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
		Addr:              net.JoinHostPort("0.0.0.0", strconv.Itoa(port)),
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
