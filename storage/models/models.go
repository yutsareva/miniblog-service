package models

import (
	"encoding/json"
	"log"
)

type Post struct {
	Id        string    `json:"id"`
	AuthorId  string    `json:"authorId"`
	Text      string    `json:"text"`
	CreatedAt string `json:"createdAt"`
	// CreatedAt time.Time `json:"createdAt"`
}

func (p *Post) ToJson() []byte {
	j, err := json.Marshal(p)
	if err != nil {
		log.Fatalf("Failed to dump post to json: %s", err.Error())
	}
	return j
}
