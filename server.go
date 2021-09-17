package main

import (
	"github.com/gorilla/mux"
	"log"
	"miniblog/handlers"
	"miniblog/storage/in_memory"
	"net/http"
	"time"
)

func main() {
	r := mux.NewRouter()

	handler := &handlers.HTTPHandler{in_memory.CreateInMemoryStorage()}

	r.HandleFunc("/api/v1/posts", handler.HandleCreatePost).Methods("POST")

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Printf("Start serving on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
