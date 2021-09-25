package in_memory

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"miniblog/storage"
	"miniblog/storage/models"
	"sync"
	"time"
)

type Post struct {
	Id        string `json:"id"`
	AuthorId  string `json:"authorId"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

func (p Post) ToJson() []byte { // TODO why no pointer ??
	j, err := json.Marshal(p)
	if err != nil {
		log.Fatalf("Failed to dump post to json: %s", err.Error())
	}
	return j
}

type InMemoryStorage struct {
	mut           sync.RWMutex
	posts         map[string]Post
	postIdsByUser map[string][]string
}

func (s *InMemoryStorage) GetPostsByUserId(
	ctx context.Context, userId *string, page *string, size int) ([]models.Post, *string, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	postIds, found := s.postIdsByUser[*userId]
	posts := make([]models.Post, 0)
	if !found {
		if page != nil {
			return nil, nil, nil
		}
		return posts, nil, nil
	}
	postCount := len(postIds)
	if page == nil {
		first := 0
		if postCount-size > 0 {
			first = postCount - size
		}

		for idx := range postIds[first:] {
			i := postCount - idx - 1
			posts = append(posts, s.posts[postIds[i]])
		}
		if first == 0 {
			return posts, nil, nil
		}
		return posts, &postIds[first-1], nil
	}

	var last *int
	for i := postCount - 1; i >= 0; i-- {
		if postIds[i] == *page {
			last = new(int)
			*last = i
			break
		}
	}
	if last != nil {
		first := 0
		if *last-size+1 >= 0 {
			first = *last - size + 1
		}
		for idx := range postIds[first : *last+1] {
			i := *last - idx
			posts = append(posts, s.posts[postIds[i]])
		}
		if first == 0 {
			return posts, nil, nil
		}
		return posts, &postIds[first-1], nil
	}
	return nil, nil, nil
}

func (s *InMemoryStorage) AddPost(ctx context.Context, userId *string, text *string) (models.Post, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	id := uuid.New().String()
	createdAt := time.Now().UTC().Format(time.RFC3339)
	p := Post{
		Id:        id,
		AuthorId:  *userId,
		Text:      *text,
		CreatedAt: createdAt,
	}
	s.posts[p.Id] = p
	s.postIdsByUser[p.AuthorId] = append(s.postIdsByUser[p.AuthorId], p.Id)
	return &p, nil
}

func (s *InMemoryStorage) GetPost(ctx context.Context, postId *string) (models.Post, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	post, found := s.posts[*postId]
	if !found {
		return nil, nil
	}
	return &post, nil
}

func CreateInMemoryStorage() storage.Storage {
	return &InMemoryStorage{
		posts:         make(map[string]Post),
		postIdsByUser: make(map[string][]string),
	}
}
