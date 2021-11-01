package persistent_cached

import (
	"context"
	"github.com/go-redis/redis/v8"
	"miniblog/storage"
	"miniblog/storage/models"
)

func CreatePersistentStorageCachedWithRedis(persistentStorage storage.Storage, redisUrl string) storage.Storage {
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisUrl,
	})
	return &PersistentStorageWithCache{
		client:            redisClient,
		persistentStorage: persistentStorage,
	}
}

type PersistentStorageWithCache struct {
	client            *redis.Client
	persistentStorage storage.Storage
}

func (s *PersistentStorageWithCache) AddPost(ctx context.Context, userId string, text string) (models.Post, error) {
	return s.persistentStorage.AddPost(ctx, userId, text)
}
func (s *PersistentStorageWithCache) GetPost(ctx context.Context, id string) (models.Post, error) {
	return s.persistentStorage.GetPost(ctx, id)
}
func (s *PersistentStorageWithCache) GetPostsByUserId(ctx context.Context, userId *string, page *string, size int) ([]models.Post, *string, error) {
	return s.persistentStorage.GetPostsByUserId(ctx, userId, page, size)
}
