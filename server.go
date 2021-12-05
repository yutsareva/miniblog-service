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
	ServerMode AppMode = "server"
	WorkerMode         = "worker"
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
	} else {
		mongoUrl, found := os.LookupEnv("MONGO_URL")
		if !found {
			panic("'MONGO_URL' not specified")
		}
		mongoDbName, found := os.LookupEnv("MONGO_DBNAME")
		if !found {
			panic("'MONGO_DBNAME' not specified")
		}
		if StorageMode(storageMode) == Mongo {
			storage = persistent.CreateMongoStorage(mongoUrl, mongoDbName)
		} else if StorageMode(storageMode) == MongoWithCache {
			redisUrl, found := os.LookupEnv("REDIS_URL")
			if !found {
				panic("'REDIS_URL' was not specified for 'cached' STORAGE_MODE")
			}
			persistentStorage := persistent.CreateMongoStorage(mongoUrl, mongoDbName)
			storage = persistent_cached.CreatePersistentStorageCachedWithRedis(persistentStorage, redisUrl)

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
		redisUrl, found := os.LookupEnv("REDIS_URL")
		if !found {
			panic("'REDIS_URL' was not specified for 'cached' STORAGE_MODE")
		}
		if err := persistent_cached.CreateWorker(redisUrl); err != nil {
			panic("Failed to start worker: " + err.Error())
		}
	default:
		panic("Invalid 'APP_MODE'")
	}
}
