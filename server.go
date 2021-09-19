package main

import (
	"github.com/gorilla/mux"
	"log"
	"miniblog/handlers"
	"miniblog/storage/in_memory"
	"net/http"
	"os"
	"time"
)

func CreateServer() *http.Server {
	r := mux.NewRouter()

	handler := &handlers.HTTPHandler{in_memory.CreateInMemoryStorage()}

	r.HandleFunc("/api/v1/posts", handler.HandleCreatePost).Methods("POST")
	r.HandleFunc("/api/v1/posts/{postId}", handler.HandleGetPost).Methods("GET")
	r.HandleFunc("/api/v1/users/{userId}/posts", handler.HandleGetPosts).Methods("GET")

	port, found := os.LookupEnv("SERVER_PORT")
	if !found {
		port = "8080"
	}

	return &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}

func main() {
	srv := CreateServer()
	log.Printf("Start serving on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
