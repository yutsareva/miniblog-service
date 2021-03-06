package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

type CreatePostRequestData struct {
	Text string `json:"text"`
}

func (h *HTTPHandler) HandleCreatePost(w http.ResponseWriter, r *http.Request) {
	var data CreatePostRequestData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userId := r.Header.Get("System-Design-User-Id")
	if userId == "" {
		http.Error(w, "Invalid user token", http.StatusUnauthorized)
		return
	}

	post, err := h.Storage.AddPost(r.Context(), userId, data.Text)
	if err != nil {
		log.Printf("Failed to add post: %s", err.Error())
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
