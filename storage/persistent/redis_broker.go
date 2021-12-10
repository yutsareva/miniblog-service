package persistent

import (
	"context"
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
	//"go.mongodb.org/mongo-driver/mongo"
	//"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	//"sync"
	//"github.com/RichardKnop/machinery/v1/log"
	//"github.com/RichardKnop/machinery/v1/tasks"
)

const (
	PAGE_SIZE int = 100
)

func addSubscription(userId, subscriber string) (int, error) {
	addedPostCount := 0
	log.Printf("addSubscription get storage")
	mongo := GetMongoStorageWithoutBroker()
	var page string
	log.Printf("Start")

	for true {
		log.Printf("GetPostsByUserId")
		posts, maybePage, err := mongo.GetPostsByUserId(context.Background(), &userId, &page, PAGE_SIZE)
		if err != nil {
			log.Printf("Failed to process subscription: %s; %s -> %s", err.Error(), subscriber, userId)
			return 0, err
		}
		log.Printf("Got %d posts", len(posts))

		err = mongo.UpdateFeed(context.Background(), userId, posts)
		if err != nil {
			log.Printf("Failed to process subscription: %s; %s -> %s", err.Error(), subscriber, userId)
			return 0, err
		}
		log.Printf("Added %d posts", len(posts))

		addedPostCount += len(posts)
		if maybePage != nil {
			page = *maybePage
		} else {
			break
		}
	}
	return addedPostCount, nil
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
