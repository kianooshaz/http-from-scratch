package main

import (
	"log"
	"net/http"

	"github.com/kianooshaz/http-from-scratch/http0.9/server"
)

func main() {
	addr := "127.0.0.1:9000"
	s := server.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World!"))
		}),
	}
	log.Printf("Listening on %s", addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
