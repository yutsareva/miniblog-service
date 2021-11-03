package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"miniblog/storage"
	"net/http"
	"path"
)

type PatchPostRequestData struct {
	Text string `json:"text"`
}

func (h *HTTPHandler) HandlePatchPost(w http.ResponseWriter, r *http.Request) {
	postId := path.Base(r.URL.Path)
	var data CreatePostRequestData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Printf("Failed to decode post data while updating post: %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userId := r.Header.Get("System-Design-User-Id")
	if userId == "" {
		http.Error(w, "Invalid user token", http.StatusUnauthorized)
		return
	}

	post, err := h.Storage.PatchPost(r.Context(), postId, userId, data.Text)
	if err != nil {
		if errors.Is(err, storage.Forbidden) {
			log.Printf("Forbidden error while updating post: %s", err.Error())
			http.Error(w, "Post is owned by another user.", http.StatusForbidden)
			return
		}
		if errors.Is(err, storage.NotFoundError) {
			log.Printf("Not Found error while updating post: %s", err.Error())
			http.Error(w, "Post is owned by another user.", http.StatusForbidden)
			return
		}
		if errors.As(err, &storage.ClientError) {
			log.Printf("Client error while updating post: %s", err.Error())
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		log.Printf("Internal error while updating post: %s", err.Error())
		http.Error(w, INTERNAL_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}
	rawResponse, err := json.Marshal(post)
	if err != nil {
		log.Printf("Failed to dump posts by user to json: %s", err.Error())
		http.Error(w, INTERNAL_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(rawResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
