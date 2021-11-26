package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"miniblog/storage"
	"net/http"
)

type Subscribers struct {
	Users    []string `json:"users"`
}

func (h *HTTPHandler) HandleGetSubscribers(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("System-Design-User-Id")
	if userId == "" {
		http.Error(w, "Invalid user token", http.StatusUnauthorized)
		return
	}
	subscriptions, err := h.Storage.GetSubscribers(r.Context(), userId)
	if err != nil {
		if errors.As(err, &storage.ClientError) {
			log.Printf("Client error while getting subscriptions for user: %s", err.Error())
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		log.Printf("Failed to get subscriptions for user: %s", err.Error())
		http.Error(w, INTERNAL_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	subscriptionsResponse := Subscriptions{
		subscriptions,
	}

	w.Header().Set("Content-Type", "application/json")
	rawResponse, err := json.Marshal(subscriptionsResponse)
	if err != nil {
		log.Printf("Failed to dump subscriptions to json: %s", err.Error())
		http.Error(w, INTERNAL_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}
	w.Write(rawResponse)
}

