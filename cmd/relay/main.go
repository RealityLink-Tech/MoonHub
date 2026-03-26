// cloud/relay/main.go
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/RealityLink-Tech/MoonHub/cloud/relay"
)

func main() {
	addr := flag.String("addr", ":8081", "listen address")
	directoryURL := flag.String("directory-url", "http://localhost:8080", "directory service URL")
	flag.Parse()

	auth := relay.NewAuthenticator(*directoryURL)
	bridge := relay.NewBridge()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			http.Error(w, "not a websocket request", http.StatusBadRequest)
			return
		}

		challenge := relay.GenerateChallenge()
		agentID, err := auth.Verify(r.Header.Get("Authorization"), challenge)
		if err != nil {
			http.Error(w, "authentication failed", http.StatusUnauthorized)
			return
		}

		r.Header.Set("X-Agent-Id", agentID)
		r.Header.Set("X-Challenge", challenge)
		bridge.ServeHTTP(w, r)
	})

	go bridge.Run()

	log.Printf("relay service listening on %s", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
