package in_memory

import (
	"github.com/google/uuid"
	"miniblog/storage"
	"miniblog/storage/models"
	"sync"
	"time"
	//"log"
)

type InMemoryStorage struct {
	mut           sync.RWMutex
	posts         map[string]models.Post
	postIdsByUser map[string][]string
}


func (s *InMemoryStorage) GetPostsByUserId(userId *string, page *string, size int) ([]models.Post, *string) {
	postIds, found := s.postIdsByUser[*userId]
	var posts []models.Post
	if !found {
		if page != nil {
			return nil, nil
		}
		return posts, nil
	}
	postCount := len(postIds)
	if page == nil {
		first := 0
		if  postCount - size > 0 {
			first = postCount - size
		}

		for idx, _ := range postIds[first:] {
			i := postCount - idx - 1
			posts = append(posts, s.posts[postIds[i]])
		}
		return posts, &postIds[first]
	}

	var nextAfterLast *int
	for i := postCount-1; i >= 0; i-- {
		if postIds[i] == *page {
			nextAfterLast = new(int)
			*nextAfterLast = i
			break
		}
	}
	if nextAfterLast != nil {
		first := 0
		if  *nextAfterLast - size > 0 {
			first = *nextAfterLast - size
		}
		for idx, _ := range postIds[first: *nextAfterLast] {
			i := *nextAfterLast - idx - 1
			posts = append(posts, s.posts[postIds[i]])
		}
		return posts, &postIds[first]
	}
	return nil, nil
}

func (s *InMemoryStorage) AddPost(userId *string, text *string) models.Post {
	id := uuid.New().String()
	createdAt := time.Now().UTC().Format(time.RFC3339)
	p := models.Post{id, *userId, *text, createdAt}
	s.posts[p.Id] = p
	s.postIdsByUser[p.AuthorId] = append(s.postIdsByUser[p.AuthorId], p.Id)
	return p
}

func (s *InMemoryStorage) GetPost(postId *string) *models.Post {
	post, found := s.posts[*postId]
	if !found {
		return nil
	}
	return &post
}

func CreateInMemoryStorage() storage.Storage {
	return &InMemoryStorage{
		//make(sync.RWMutex),
		posts: make(map[string]models.Post),
		postIdsByUser: make(map[string][]string),
	}
}
