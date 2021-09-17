package storage

import "miniblog/storage/models"

type Storage interface {
	AddPost(userId *string, text *string) []byte
	GetPostById(id *string) models.Post
	GetPostsByUserId(userId *string, page *string, size int) []models.Post
}
