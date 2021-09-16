package server

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"math/rand"
	"miniblog/storage/in_memory"
	"net/http"
	"os"
	"time"
	"miniblog/storage"
)

type CreatePostData struct {
	Key string `json:"text"`
}

func handleCreatePost(w http.ResponseWriter, r *http.Request) {

	_, err := w.Write([]byte("Hello from server"))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	w.Header().Set("Content-Type", "plain/text")
	//	fmt.Fprintln(os.Stderr, "PUT URL")
	//	var data PutRequestData
	//
	//	err := json.NewDecoder(r.Body).Decode(&data)
	//	if err != nil {
	//		http.Error(w, err.Error(), http.StatusBadRequest)
	//		return
	//	}
	//	fmt.Fprintln(os.Stderr, "PUT URL: " + data.Url)
	//
	//	newUrlKey := getRandomKey()
	//	h.storage[newUrlKey] = data.Url
	//	//  http://my.site.com/bdfhfd
	//
	//	response := PutResponseData{
	//		Key: newUrlKey,
	//	}
	//	rawResponse, _ := json.Marshal(response)
	//
	//	_, err = w.Write(rawResponse)
	//	if err != nil {
	//		http.Error(w, err.Error(), http.StatusBadRequest)
	//		return
	//	}
	//
	//	w.Header().Set("Content-Type", "application/json")
}

type HTTPHandler struct {
	storage storage.Storage
}


func main() {
	r := mux.NewRouter()

	handler := &HTTPHandler{ &in_memory.InMemoryStorage{} }

	r.HandleFunc("/api/v1/posts", handler.handleCreatePost).Methods("POST")

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Printf("Start serving on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}