package main

import (
	"github.com/gorilla/mux"
	"log"
	"miniblog/handlers"
	"miniblog/storage/in_memory"
	"net/http"
	"time"
)

func CreateServer() *http.Server {
	r := mux.NewRouter()

	handler := &handlers.HTTPHandler{in_memory.CreateInMemoryStorage()}

	r.HandleFunc("/api/v1/posts", handler.HandleCreatePost).Methods("POST")

	return &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}

func main() {
	srv := CreateServer()
	log.Printf("Start serving on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
