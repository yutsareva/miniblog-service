package storage

import "miniblog/storage/models"

type Storage interface {
	GetPostById(id *string) models.Post
	GetPostsByUserId(userId *string, page *string, size int) []models.Post
}


