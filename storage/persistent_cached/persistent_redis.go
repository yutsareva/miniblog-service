package persistent_cached

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"miniblog/storage"
	"miniblog/storage/models"
	"miniblog/storage/persistent"
	"time"
)

func saveToCache(ctx context.Context, client *redis.Client, post models.Post) {
	j, err := json.Marshal(post)
	if err == nil {
		err = client.Set(ctx, post.GetId(), j, time.Hour).Err()
		if err != nil {
			fmt.Println("Failed to save post to redis: ", err)
		}
	}
}

func getFromCache(ctx context.Context, client *redis.Client, postId string) (models.Post, error) {
	val, err := client.Get(ctx, postId).Result()
	if err == nil {
		var p persistent.Post
		err = json.Unmarshal([]byte(val), &p)
		if err == nil {
			return &p, nil
		}
	}
	fmt.Println("Failed to get post from redis: ", err)
	return nil, err
}

func removeFromCache(ctx context.Context, client *redis.Client, postId string) {
	err := client.Del(ctx, postId).Err()
	if err != nil {
		fmt.Println("Failed to remove post from redis: ", err.Error())
	}
}

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

func (s *PersistentStorageWithCache) PatchPost(ctx context.Context, id string, userId string, text string) (models.Post, error) {
	post, err := s.persistentStorage.PatchPost(ctx, id, userId, text)
	if err == nil {
		removeFromCache(ctx, s.client, post.GetId())
	}
	return post, err
}

func (s *PersistentStorageWithCache) AddPost(ctx context.Context, userId string, text string) (models.Post, error) {
	post, err := s.persistentStorage.AddPost(ctx, userId, text)
	if err == nil {
		saveToCache(ctx, s.client, post)
	}
	return post, err
}

func (s *PersistentStorageWithCache) GetPost(ctx context.Context, postId string) (models.Post, error) {
	p, err := getFromCache(ctx, s.client, postId)
	if err == nil {
		return p, nil
	}
	post, err := s.persistentStorage.GetPost(ctx, postId)
	if err == nil {
		saveToCache(ctx, s.client, post)
	}
	return post, err
}

func (s *PersistentStorageWithCache) GetPostsByUserId(ctx context.Context, userId *string, page *string, size int) ([]models.Post, *string, error) {
	return s.persistentStorage.GetPostsByUserId(ctx, userId, page, size)
}
