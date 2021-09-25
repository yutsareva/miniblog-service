package handlers

import (
	"encoding/json"
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
	post, _ := h.Storage.AddPost(r.Context(), &userId, &data.Text)
	rawResponse := post.ToJson()

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(rawResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
