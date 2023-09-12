package main

import (
	"flag"
	"log"
	"net"
	"strconv"

	"github.com/re-connect-ai/reconn/httpsvc"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 8080, "http server port")
	flag.Parse()

	log.Printf("about to start on port %d", port)

	svc := new(httpsvc.HttpService)
	r := svc.SetupRouter()
	if err := r.Run(net.JoinHostPort("0.0.0.0", strconv.Itoa(port))); err != nil {
		log.Fatal(err)
	}
}
