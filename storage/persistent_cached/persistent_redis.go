package persistent_cached

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"miniblog/storage"
	"miniblog/storage/models"
	"miniblog/storage/persistent"
	"strconv"
)

// TODO: current implementation depends on time synchronization on different replicas

// KEYS[1] - key
// KEYS[2] - version
// KEYS[3] - value
var UPDATE_SCRIPT_STR = `
	local old_version = redis.call("hget", KEYS[1], "version")
	if (old_version == false) then
	  redis.call("hset", KEYS[1], "value", KEYS[3])
	  redis.call("hset", KEYS[1], "version", KEYS[2])
	  return 1
    end
    if (tonumber(old_version) < tonumber(KEYS[2])) then
	  redis.call("hset", KEYS[1], "value", KEYS[3])
	  redis.call("hset", KEYS[1], "version", KEYS[2])
	  return 1
    end
    return 0
`
var UPDATE_SCRIPT = redis.NewScript(UPDATE_SCRIPT_STR)

func updateCache(ctx context.Context, client *redis.Client, post models.Post) {
	j, err := json.Marshal(post)
	if err != nil {
		fmt.Println("Failed to dump to json:", err)
		return
	}
	_, err = UPDATE_SCRIPT.Run(
		ctx,
		client,
		[]string{post.GetId(), strconv.FormatInt(post.GetLastModifiedAt(), 10), string(j)},
		[]interface{}{},
	).Result()
	if err != nil {
		fmt.Println("Failed to update redis cache:", err)
		return
	}
	//fmt.Println("Cache update returned: ", updated)
}

func getFromCache(ctx context.Context, client *redis.Client, postId string) (models.Post, error) {
	val, err := client.Get(ctx, postId).Result()
	if err == nil {
		var p persistent.Post
		err = json.Unmarshal([]byte(val), &p)
		if err == nil {
			//fmt.Println("Got post from redis!")
			return &p, nil
		}
	}
	//fmt.Println("Failed to get post from redis:", err)
	return nil, err
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
		updateCache(ctx, s.client, post)
	}
	return post, err
}

func (s *PersistentStorageWithCache) AddPost(ctx context.Context, userId string, text string) (models.Post, error) {
	post, err := s.persistentStorage.AddPost(ctx, userId, text)
	if err == nil {
		updateCache(ctx, s.client, post)
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
		updateCache(ctx, s.client, post)
	}
	return post, err
}

func (s *PersistentStorageWithCache) GetPostsByUserId(ctx context.Context, userId *string, page *string, size int) ([]models.Post, *string, error) {
	return s.persistentStorage.GetPostsByUserId(ctx, userId, page, size)
}
