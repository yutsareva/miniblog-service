package main

import (
	"github.com/gorilla/mux"
	_ "github.com/motemen/go-loghttp/global"
	"log"
	"miniblog/handlers"
	"miniblog/storage"
	"miniblog/storage/in_memory"
	"miniblog/storage/persistent"
	"miniblog/storage/persistent_cached"
	"miniblog/utils"
	"net/http"
	"os"
	"time"
)

type StorageMode string

const (
	InMemory       StorageMode = "inmemory"
	Mongo                      = "mongo"
	MongoWithCache             = "cached"
)

type AppMode string

const (
	ServerMode AppMode = "SERVER"
	WorkerMode         = "WORKER"
)

func CreateServer() *http.Server {
	r := mux.NewRouter()

	port := utils.GetEnvVarWithDefault("SERVER_PORT", "8080")
	storageMode := utils.GetEnvVarWithDefault("STORAGE_MODE", "mongo")

	var storage storage.Storage
	if StorageMode(storageMode) == InMemory {
		storage = in_memory.CreateInMemoryStorage()
	} else {
		mongoUrl := utils.GetEnvVar("MONGO_URL")
		mongoDbName := utils.GetEnvVar("MONGO_DBNAME")
		brokerUrl := "redis://" + utils.GetEnvVar("REDIS_URL")
		if StorageMode(storageMode) == Mongo {
			storage = persistent.CreateMongoStorageWithBroker(mongoUrl, mongoDbName, brokerUrl)
		} else if StorageMode(storageMode) == MongoWithCache {
			cacheUrl := utils.GetEnvVar("REDIS_CACHE_URL")
			persistentStorage := persistent.CreateMongoStorageWithBroker(mongoUrl, mongoDbName, brokerUrl)
			storage = persistent_cached.CreatePersistentStorageCachedWithRedis(persistentStorage, cacheUrl)
		} else {
			panic("Invalid 'STORAGE_MODE'")
		}
	}

	handler := &handlers.HTTPHandler{Storage: storage}

	r.HandleFunc("/maintenance/ping", handler.HealthCheck).Methods("GET")
	r.HandleFunc("/api/v1/posts", handler.HandleCreatePost).Methods("POST")
	r.HandleFunc("/api/v1/posts/{postId}", handler.HandleGetPost).Methods("GET")
	r.HandleFunc("/api/v1/users/{userId}/posts", handler.HandleGetPosts).Methods("GET")
	r.HandleFunc("/api/v1/posts/{postId}", handler.HandlePatchPost).Methods("PATCH")
	r.HandleFunc("/api/v1/users/{userId}/subscribe", handler.HandleSubscribe).Methods("POST")
	r.HandleFunc("/api/v1/subscriptions", handler.HandleGetSubscriptions).Methods("GET")
	r.HandleFunc("/api/v1/subscribers", handler.HandleGetSubscribers).Methods("GET")
	r.HandleFunc("/api/v1/feed", handler.HandleFeed).Methods("GET")

	return &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}

func main() {
	appMode, found := os.LookupEnv("APP_MODE")
	if !found {
		panic("'APP_MODE' not specified")
	}
	switch AppMode(appMode) {
	case ServerMode:
		srv := CreateServer()
		log.Printf("Start serving on %s", srv.Addr)
		log.Fatal(srv.ListenAndServe())
	case WorkerMode:
		brokerUrl := "redis://" + utils.GetEnvVar("REDIS_URL")
		if err := persistent.CreateWorker(brokerUrl); err != nil {
			panic("Failed to start worker: " + err.Error())
		}
	default:
		panic("Invalid 'APP_MODE'")
	}
}
