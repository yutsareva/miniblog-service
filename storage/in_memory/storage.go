package in_memory

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"miniblog/storage"
	"miniblog/storage/models"
	"sync"
	"time"
)

type Post struct {
	Id             string `json:"id"`
	AuthorId       string `json:"authorId"`
	Text           string `json:"text"`
	CreatedAt      string `json:"createdAt"`
	LastModifiedAt string `json:"lastModifiedAt"`
}

func (p *Post) GetId() string {
	return p.Id
}

func (p *Post) GetVersion() int64 {
	return 0
}

type InMemoryStorage struct {
	mut           sync.RWMutex
	posts         map[string]Post
	postIdsByUser map[string][]string
}

func (s *InMemoryStorage) PatchPost(
	ctx context.Context,
	postId string,
	userId string,
	text string,
) (models.Post, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	post, found := s.posts[postId]
	if !found {
		return nil, fmt.Errorf("post %s not found: %w", postId, storage.NotFoundError)
	}
	if post.AuthorId != userId {
		return nil, fmt.Errorf("post %s is owned by another user: %w", postId, storage.Forbidden)
	}
	post.Text = text
	post.AuthorId = userId
	post.LastModifiedAt = time.Now().UTC().Format(time.RFC3339)
	s.posts[postId] = post
	return &post, nil
}

func (s *InMemoryStorage) GetPostsByUserId(
	ctx context.Context, userId *string, page *string, size int) ([]models.Post, *string, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	postIds, found := s.postIdsByUser[*userId]
	posts := make([]models.Post, 0)
	if !found {
		if page != nil {
			return nil, nil, fmt.Errorf("provided page for non-existent user", storage.ClientError)
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
			post := s.posts[postIds[i]]
			posts = append(posts, &post)
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
	if last == nil {
		return nil, nil, fmt.Errorf("page not found", storage.ClientError)
	}
	first := 0
	if *last-size+1 >= 0 {
		first = *last - size + 1
	}
	for idx := range postIds[first : *last+1] {
		i := *last - idx
		post := s.posts[postIds[i]]
		posts = append(posts, &post)
	}
	if first == 0 {
		return posts, nil, nil
	}
	return posts, &postIds[first-1], nil

}

func (s *InMemoryStorage) AddPost(ctx context.Context, userId, text string) (models.Post, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	id := uuid.New().String()
	createdAt := time.Now().UTC().Format(time.RFC3339)
	p := Post{
		Id:             id,
		AuthorId:       userId,
		Text:           text,
		CreatedAt:      createdAt,
		LastModifiedAt: createdAt,
	}
	s.posts[p.Id] = p
	s.postIdsByUser[p.AuthorId] = append(s.postIdsByUser[p.AuthorId], p.Id)
	return &p, nil
}

func (s *InMemoryStorage) GetPost(ctx context.Context, postId string) (models.Post, error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	post, found := s.posts[postId]
	if !found {
		return nil, fmt.Errorf("post %s not found: %w", postId, storage.NotFoundError)
	}
	return &post, nil
}

func CreateInMemoryStorage() storage.Storage {
	return &InMemoryStorage{
		posts:         make(map[string]Post),
		postIdsByUser: make(map[string][]string),
	}
}
