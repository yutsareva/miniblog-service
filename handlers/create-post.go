package handlers

import (
	"encoding/json"
	"net/http"
)

type CreatePostRequestData struct {
	Text string `json:"text"`
}

func (h *HTTPHandler) HandleCreatePost(w http.ResponseWriter, r *http.Request) {

	//_, err := w.Write([]byte("Hello from server"))
	//if err != nil {
	//	fmt.Println(err.Error())
	//	return
	//}
	//w.Header().Set("Content-Type", "plain/text")
	//	fmt.Fprintln(os.Stderr, "PUT URL")
	var data CreatePostRequestData
	//
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//	fmt.Fprintln(os.Stderr, "PUT URL: " + data.Url)
	//
	//	newUrlKey := getRandomKey()
	userId := r.Header.Get("System-Design-User-Id")
	rawResponse := h.Storage.AddPost(&userId, &data.Text)
	//h.storage[id] = data.Url
	//fmt.Println(id.String())
	//	//  http://my.site.com/bdfhfd
	//
	//	response := PutResponseData{
	//		Key: newUrlKey,
	//	}
	//
	_, err = w.Write(rawResponse)
	//	if err != nil {
	//		http.Error(w, err.Error(), http.StatusBadRequest)
	//		return
	//	}
	//
	w.Header().Set("Content-Type", "application/json")
}
