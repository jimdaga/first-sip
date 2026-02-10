package main

import (
	"log"
	"net/http"

	"github.com/jimdaga/first-sip/internal/health"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health.Handler)

	log.Println("starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
