package main

import (
	"log/slog"
	"os"
	"time"

	"exp1/tasks"

	"github.com/hibiken/asynq"
	"github.com/lmittmann/tint"
)

const redisAddr = "127.0.0.1:6379"

func main() {
	log := tasks.Logger(os.Stderr, os.Getenv("LOG_LEVEL"))

	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	defer client.Close()

	// Email
	// Enqueue task to be processed immediately
	task, err := tasks.BuildNotificationEmail("5E8pR@example.com", "Hello!", "How are you?")
	if err != nil {
		log.Error("could not create task", tint.Err(err))
		os.Exit(1)
	}
	info, err := client.Enqueue(task)
	if err != nil {
		log.Error("could not enqueue task", tint.Err(err))
		os.Exit(1)
	}
	log.Info("enqueued task", slog.String("id", info.ID), slog.String("queue", info.Queue), slog.Any("state", info.State))

	// Schedule the same task to be processed in the future
	info, err = client.Enqueue(task, asynq.ProcessIn(time.Minute))
	if err != nil {
		log.Error("could not enqueue task", tint.Err(err))
		os.Exit(1)
	}
	log.Info("enqueued task", slog.String("id", info.ID), slog.String("queue", info.Queue), slog.Any("state", info.State))

	// SMS
	// Set other options to tune task processing behavior.
	// Options include MaxRetry, Queue, Timeout, Deadline, Unique etc.
	task, err = tasks.BuildNotificationSMS("0123456789", "How are you?")
	if err != nil {
		log.Error("could not create task", tint.Err(err))
		os.Exit(1)
	}
	info, err = client.Enqueue(task, asynq.MaxRetry(10), asynq.Timeout(3*time.Minute))
	if err != nil {
		log.Error("could not enqueue task", tint.Err(err))
		os.Exit(1)
	}
	log.Info("enqueued task", slog.String("id", info.ID), slog.String("queue", info.Queue), slog.Any("state", info.State))

	// Push
	task, err = tasks.BuildNotificationPush("0123456789", "How are you?")
	if err != nil {
		log.Error("could not create task", tint.Err(err))
		os.Exit(1)
	}
	info, err = client.Enqueue(
		task, asynq.Queue("critical"),
		asynq.Deadline(time.Now().Add(30*time.Minute)),
		asynq.Retention(24*time.Hour),
	)
	if err != nil {
		log.Error("could not enqueue task", tint.Err(err))
		os.Exit(1)
	}
	log.Info("enqueued task", slog.String("id", info.ID), slog.String("queue", info.Queue), slog.Any("state", info.State))
}
