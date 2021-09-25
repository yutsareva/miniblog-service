package handlers

import (
	"encoding/json"
	"log"
	"miniblog/storage/models"
	"net/http"
	"path"
	"strconv"
)

var DEFAULT_PAGE_SIZE = 10

type PostByUserIdResponse struct {
	Posts    []models.Post `json:"posts,omitempty"`
	NextPage *string       `json:"nextPage,omitempty"`
}

func (p *PostByUserIdResponse) ToJson() []byte {
	j, err := json.Marshal(p)
	if err != nil {
		log.Fatalf("Failed to dump posts by user to json: %s", err.Error())
	}
	return j
}

func (h *HTTPHandler) HandleGetPosts(w http.ResponseWriter, r *http.Request) {
	userId := path.Base(path.Dir(r.URL.Path))

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

	posts, nextPage, _ := h.Storage.GetPostsByUserId(r.Context(), &userId, page, size)

	if posts == nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	postsResponse := PostByUserIdResponse{
		posts,
		nextPage,
	}

	w.Header().Set("Content-Type", "application/json")
	rawResponse := postsResponse.ToJson()
	w.Write(rawResponse)
}
