package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/kianooshaz/http-from-scratch/http1/server"
)

func main() {
	addr := "127.0.0.1:9000"
	mux := http.NewServeMux()

	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		json.NewEncoder(w).Encode(r.Header)
	})

	s := server.Server{
		Addr:    addr,
		Handler: mux,
	}
	log.Printf("Starting web server: http://%s", addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
