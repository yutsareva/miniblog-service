package persistent_cached

import (
	"context"
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
	//"go.mongodb.org/mongo-driver/mongo"
	//"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"miniblog/storage/persistent"
	//"sync"
	//"github.com/RichardKnop/machinery/v1/log"
	//"github.com/RichardKnop/machinery/v1/tasks"
)

const (
	PAGE_SIZE int = 100
)

func addSubscription(userId, subscriber string) (int, error) {
	addedPostCount := 0
	mongo := persistent.GetMongoStorage()
	page := ""

	for true {
		posts, maybePage, err := mongo.GetPostsByUserId(context.Background(), &userId, &page, PAGE_SIZE)
		if err != nil {
			log.Printf("Failed to process subscription: %s; %s -> %s", err.Error(), subscriber, userId)
			return 0, err
		}

		err = mongo.UpdateFeed(context.Background(), userId, posts)
		if err != nil {
			log.Printf("Failed to process subscription: %s; %s -> %s", err.Error(), subscriber, userId)
			return 0, err
		}

		addedPostCount += len(posts)
		if maybePage != nil {
			page = *maybePage
		} else {
			break
		}
	}
	return addedPostCount, nil
}

func startBroker(redisUrl string) (*machinery.Server, error) {
	cnf := &config.Config{
		DefaultQueue:    "machinery_tasks",
		ResultsExpireIn: 3600,
		Broker:          redisUrl, // "redis://localhost:6379"
		ResultBackend:   redisUrl,
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
	}
	return server, server.RegisterTasks(tasks)
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
