package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"miniblog/storage"
	"net/http"
	"path"
)

func (h *HTTPHandler) HandleGetPost(w http.ResponseWriter, r *http.Request) {
	postId := path.Base(r.URL.Path)
	post, err := h.Storage.GetPost(r.Context(), postId)
	if err != nil {
		if errors.As(err, &storage.NotFoundError) {
			http.Error(w, "Post was not found. Please check post id.", http.StatusNotFound)
			return
		}
		if errors.As(err, &storage.ClientError) {
			log.Printf("Client error while getting posts for author: %s", err.Error())
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		log.Printf("Failed to get posts for author: %s", err.Error())
		http.Error(w, INTERNAL_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	rawResponse, err := json.Marshal(post)
	if err != nil {
		log.Printf("Failed to dump post to json: %s", err.Error())
		http.Error(w, INTERNAL_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}
	w.Write(rawResponse)
}
