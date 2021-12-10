package handlers

import (
	"log"
	"net/http"
	"path"
)

func (h *HTTPHandler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	subscriberId := r.Header.Get("System-Design-User-Id")
	userId := path.Base(path.Dir(r.URL.Path))
	if subscriberId == "" || userId == "" {
		http.Error(w, "Invalid user token", http.StatusUnauthorized)
		return
	}
	if userId == subscriberId {
		http.Error(w, "You cannot subscribe yourself.", http.StatusBadRequest)
		return
	}

	err := h.Storage.Subscribe(r.Context(), userId, subscriberId)
	if err != nil {
		log.Printf("Failed to add post: %s", err.Error())
		http.Error(w, INTERNAL_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}
}
