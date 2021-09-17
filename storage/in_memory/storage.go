package in_memory

import (
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"miniblog/storage"
	"miniblog/storage/models"
	"time"
)

type post struct {
	Id        string    `json:"id"`
	AuthorId  string    `json:"author_id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

//func (p *post) Id() string {
//	return p.Id
//}
//
//func (p *post) AuthorId() string {
//	return p.authorId
//}
//
//func (p *post) Text() string {
//	return p.text
//}
//
//func (p *post) CreatedAt() time.Time {
//	return p.createdAt
//}

type InMemoryStorage struct {
	posts map[string]post
}

func (s *InMemoryStorage) GetPostById(id *string) models.Post {
	ret := s.posts[*id]
	return &ret
}

func (s *InMemoryStorage) GetPostsByUserId(userId *string, page *string, size int) []models.Post {
	postsToReturn := make([]models.Post, 0)
	for _, post := range s.posts {
		if post.AuthorId == *userId && post.Id > *page {
			appendPost := post
			postsToReturn = append(postsToReturn, &appendPost)
		}
		if len(postsToReturn) == size {
			return postsToReturn
		}
	}
	return postsToReturn
}

func (s *InMemoryStorage) AddPost(userId *string, text *string) []byte {
	id := uuid.New().String()
	createdAt := time.Now()
	p := post{id, *userId, *text, createdAt}
	s.posts[p.Id] = p
	j, err := json.Marshal(p)
	if err != nil {
		log.Fatalf("Failed to dump post to json: %s", err.Error())
	}
	return j
}

func CreateInMemoryStorage() storage.Storage {
	return &InMemoryStorage{make(map[string]post)}
}
