package main

import (
	"os"

	"exp1/tasks"

	"github.com/hibiken/asynq"
	"github.com/lmittmann/tint"
)

const redisAddr = "127.0.0.1:6379"

func main() {
	log := tasks.Logger(os.Stderr, os.Getenv("LOG_LEVEL"))

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: 10,
			// Optionally specify multiple queues with different priority.
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	// mux maps a type to a handler
	mux := asynq.NewServeMux()
	mux.Handle(tasks.TypeNotificationEmail, tasks.NewProcessNotificationEmail(log))
	mux.HandleFunc(tasks.TypeNotificationSMS, tasks.HandleNotificationSMS)
	mux.HandleFunc(tasks.TypeNotificationPush, tasks.HandleNotificationPush)

	// Run server
	if err := srv.Run(mux); err != nil {
		log.Error("could not run server", tint.Err(err))
		os.Exit(1)
	}
}
