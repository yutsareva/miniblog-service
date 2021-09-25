package handlers

import (
	"net/http"
	"path"
)

func (h *HTTPHandler) HandleGetPost(w http.ResponseWriter, r *http.Request) {
	postId := path.Base(r.URL.Path)
	maybePost, _ := h.Storage.GetPost(r.Context(), &postId)
	if maybePost == nil {
		http.Error(w, "Post not found.", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	rawResponse := maybePost.ToJson()
	w.Write(rawResponse)
}
