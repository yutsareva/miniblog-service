package persistent

import (
	"context"
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
)

const (
	PAGE_SIZE int = 100
)

func addSubscription(userId, subscriber string) (int, error) {
	addedPostCount := 0
	mongo := GetMongoStorageWithoutBroker()
	var page *string
	page = nil

	for true {
		posts, maybePage, err := mongo.GetPostsByUserId(context.Background(), &userId, page, PAGE_SIZE)
		if err != nil {
			log.Printf("Failed to process subscription: %s; %s -> %s", err.Error(), subscriber, userId)
			return 0, err
		}

		err = mongo.UpdateFeedNewSubscription(context.Background(), subscriber, posts)
		if err != nil {
			log.Printf("Failed to process subscription: %s; %s -> %s", err.Error(), subscriber, userId)
			return 0, err
		}
		log.Printf("Added %d posts to feed from user %s to subscriber %s", len(posts), userId, subscriber)

		addedPostCount += len(posts)
		if maybePage != nil {
			page = maybePage
		} else {
			break
		}
	}
	log.Printf("Added %d feed items for user %s", addedPostCount, subscriber)
	return addedPostCount, nil
}

func addPost(postId string, authorId string) (int, error) {
	mongo := GetMongoStorageWithoutBroker()

	subscribers, err := mongo.GetSubscribers(context.Background(), authorId)
	log.Printf("Got %d subscribers for post %s: %s", len(subscribers), postId, subscribers)
	if err != nil {
		log.Printf("Failed to process adding post %s to feed: %s", postId, err.Error())
		return 0, err
	}

	addedFeedItems, err := mongo.UpdateFeedNewPost(context.Background(), postId, subscribers)
	if err != nil {
		log.Printf("Failed to process adding post %s to feed: %s", postId, err.Error())
		return 0, err
	}

	log.Printf("Added %d feed items from author %s", addedFeedItems, authorId)
	return addedFeedItems, nil
}

func patchPost(postId string) (int, error) {
	mongo := GetMongoStorageWithoutBroker()

	addedFeedItems, err := mongo.UpdateFeedPatchPost(context.Background(), postId)
	if err != nil {
		log.Printf("Failed to process adding post %s to feed: %s", postId, err.Error())
		return 0, err
	}

	log.Printf("Updated %d feed items", addedFeedItems)
	return addedFeedItems, nil
}

func CreateWorker(redisUrl string) error {
	consumerTag := "machinery_worker"

	broker, err := startBroker(redisUrl)
	if err != nil {
		return err
	}

	worker := broker.NewWorker(consumerTag, 0)

	errorhandler := func(err error) {
		log.Printf("Something went wrong: %s", err)
	}

	worker.SetErrorHandler(errorhandler)

	return worker.Launch()
}

func startBroker(brokerUrl string) (*machinery.Server, error) {
	cnf := &config.Config{
		DefaultQueue:    "machinery_tasks",
		ResultsExpireIn: 3600,
		Broker:          brokerUrl, // "redis://localhost:6379"
		ResultBackend:   brokerUrl,
		Redis: &config.RedisConfig{
			MaxIdle:                3,
			IdleTimeout:            240,
			ReadTimeout:            15,
			WriteTimeout:           15,
			ConnectTimeout:         15,
			NormalTasksPollPeriod:  1000,
			DelayedTasksPollPeriod: 500,
		},
	}
	server, err := machinery.NewServer(cnf)
	if err != nil {
		return nil, err
	}

	// Register tasks
	tasks := map[string]interface{}{
		"addSubscription": addSubscription,
		"addPost":         addPost,
		"patchPost":       patchPost,
	}
	return server, server.RegisterTasks(tasks)
}

func createAddSubscriptionTask(userId, subscriber string) tasks.Signature {
	task := tasks.Signature{
		Name: "addSubscription",
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: userId,
			},
			{
				Type:  "string",
				Value: subscriber,
			},
		},
	}
	return task
}

func createAddPostTask(postId primitive.ObjectID, authorId string) tasks.Signature {
	task := tasks.Signature{
		Name: "addPost",
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: postId.Hex(),
			},
			{
				Type:  "string",
				Value: authorId,
			},
		},
	}
	return task
}

func createPatchPostTask(postId primitive.ObjectID) tasks.Signature {
	task := tasks.Signature{
		Name: "patchPost",
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: postId.Hex(),
			},
		},
	}
	return task
}
