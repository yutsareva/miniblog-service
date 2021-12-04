package persistent_cached

import (
	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
	"log"

	//"github.com/RichardKnop/machinery/v1/log"
	//"github.com/RichardKnop/machinery/v1/tasks"
)

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
