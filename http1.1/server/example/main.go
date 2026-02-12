package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/kianooshaz/http-from-scratch/http1.1/server"
)

func main() {
	addr := "127.0.0.1:9000"
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(".")))
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		b, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		w.Write(b)
	})
	mux.HandleFunc("/echo/chunked", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		io.Copy(w, r.Body)
	})
	mux.HandleFunc("/status/{status}", func(w http.ResponseWriter, r *http.Request) {
		status, err := strconv.ParseInt(r.PathValue("status"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, fmt.Sprintf("error: %s", err))
			return
		}
		w.WriteHeader(int(status))
	})
	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "application/json")
		json.NewEncoder(w).Encode(r.Header)
	})
	mux.HandleFunc("/nothing", func(w http.ResponseWriter, r *http.Request) {})
	s := server.Server{
		Addr:    addr,
		Handler: mux,
	}
	log.Printf("Starting web server: http://%s", addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
