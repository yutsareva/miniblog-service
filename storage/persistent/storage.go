package persistent

import (
	//"github.com/google/uuid"
	"miniblog/storage"
	"miniblog/storage/models"
	//"time"
)

type MongoStorage struct {
}

func (s *MongoStorage) GetPostsByUserId(userId *string, page *string, size int) ([]models.Post, *string) {
	panic("not implemented")
}

func (s *MongoStorage) AddPost(userId *string, text *string) models.Post {
	panic("not implemented")
}

func (s *MongoStorage) GetPost(postId *string) *models.Post {
	panic("not implemented")
}

func CreateMongoStorage(mongoUrl, mongoDbName string) storage.Storage {
	return &MongoStorage{}
}
