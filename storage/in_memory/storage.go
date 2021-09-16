package in_memory

import "time"
import "miniblog/storage/models"

type post struct {
	id string
	authorId string
	text string
	createdAt time.Time
}

func (p *post) Id() string {
	return p.id
}

func (p *post) AuthorId() string {
	return p.authorId
}

func (p *post) Text() string {
	return p.text
}

func (p *post) CreatedAt() time.Time {
	return p.createdAt
}

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
		if post.authorId == *userId && post.id > *page {
			appendPost := post
			postsToReturn = append(postsToReturn, &appendPost)
		}
		if len(postsToReturn) == size {
			return postsToReturn
		}
	}
	return postsToReturn
}
