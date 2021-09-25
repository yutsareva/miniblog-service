package storage

import (
	"context"
	"errors"
	"fmt"
	"miniblog/storage/models"
)

var (
	InternalError  = errors.New("storage internal error")
	ClientError    = errors.New("storage client error")
	CollisionError = fmt.Errorf("%w.collision", ClientError)
	NotFoundError  = fmt.Errorf("%w.not_found", ClientError)
)

type Storage interface {
	AddPost(ctx context.Context, userId *string, text *string) (models.Post, error)
	GetPost(ctx context.Context, id *string) (models.Post, error)
	GetPostsByUserId(ctx context.Context, userId *string, page *string, size int) ([]models.Post, *string, error)
}
