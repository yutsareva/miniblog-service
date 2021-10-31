package main

import (
	"github.com/gorilla/mux"
	_ "github.com/motemen/go-loghttp/global"
	"log"
	"miniblog/handlers"
	"miniblog/storage"
	"miniblog/storage/in_memory"
	"miniblog/storage/persistent"
	"net/http"
	"os"
	"time"
)

type StorageMode string

const (
	InMemory StorageMode = "inmemory"
	Mongo                = "mongo"
)

func CreateServer() *http.Server {
	r := mux.NewRouter()

	port, found := os.LookupEnv("SERVER_PORT")
	if !found {
		port = "8080"
	}

	storageMode, found := os.LookupEnv("STORAGE_MODE")
	if !found {
		storageMode = "inmemory"
	}
	var storage storage.Storage
	if StorageMode(storageMode) == InMemory {
		storage = in_memory.CreateInMemoryStorage()
	} else if storageMode == Mongo {
		mongoUrl, found := os.LookupEnv("MONGO_URL")
		if !found {
			panic("'MONGO_URL' not specified")
		}
		mongoDbName, found := os.LookupEnv("MONGO_DBNAME")
		if !found {
			panic("'MONGO_DBNAME' not specified")
		}
		cached, found := os.LookupEnv("STORAGE_MODE")
		if !found || cached != "cached" {
			storage = persistent.CreateMongoStorage(mongoUrl, mongoDbName)
		} else {
			redisUrl, found := os.LookupEnv("REDIS_URL")
			if !found {
				panic("'REDIS_URL' was not provided for 'cached' STORAGE_MODE")
			}
			storage = persistent_cached.CreateMongoCachedWithRedis(mongoUrl, mongoDbName, redisUrl)
		}
	}
	handler := &handlers.HTTPHandler{Storage: storage}

	r.HandleFunc("/api/v1/posts", handler.HandleCreatePost).Methods("POST")
	r.HandleFunc("/api/v1/posts/{postId}", handler.HandleGetPost).Methods("GET")
	r.HandleFunc("/api/v1/users/{userId}/posts", handler.HandleGetPosts).Methods("GET")
	r.HandleFunc("/maintenance/ping", handler.HealthCheck).Methods("GET")

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
