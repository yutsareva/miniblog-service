package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"miniblog/storage"
	"miniblog/storage/models"
	"net/http"
	"strconv"
)


type FeedResponse struct {
	Posts    []models.Post `json:"posts,omitempty"`
	NextPage *string       `json:"nextPage,omitempty"`
}

func (h *HTTPHandler) HandleFeed(w http.ResponseWriter, r *http.Request) {
	userId := r.Header.Get("System-Design-User-Id")

	cgiPage, found := r.URL.Query()["page"]
	var page *string = nil
	if found {
		page = &cgiPage[0]
	}

	cgiSize, found := r.URL.Query()["size"]
	size := DEFAULT_PAGE_SIZE
	if found {
		var err error
		size, err = strconv.Atoi(cgiSize[0])
		if err != nil || size < 1 || size > 100 {
			http.Error(w, "Invalid size", http.StatusBadRequest)
			return
		}
	}

	posts, nextPage, err := h.Storage.Feed(r.Context(), &userId, page, size)
	if err != nil {
		if errors.As(err, &storage.ClientError) {
			log.Printf("Client error while getting posts for author: %s", err.Error())
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		log.Printf("Failed to get posts for author: %s", err.Error())
		http.Error(w, INTERNAL_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}

	postsResponse := PostByUserIdResponse{
		posts,
		nextPage,
	}

	w.Header().Set("Content-Type", "application/json")
	rawResponse, err := json.Marshal(postsResponse)
	if err != nil {
		log.Printf("Failed to dump posts by user to json: %s", err.Error())
		http.Error(w, INTERNAL_ERROR_MESSAGE, http.StatusInternalServerError)
		return
	}
	w.Write(rawResponse)
}
