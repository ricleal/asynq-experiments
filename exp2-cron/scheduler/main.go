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

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	scheduler := asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: redisAddr},
		&asynq.SchedulerOpts{
			Location: loc,
		},
	)

	// Email
	task, err := tasks.BuildNotificationEmail("5E8pR@example.com", "Hello!", "How are you?")
	if err != nil {
		log.Error("could not create task", tint.Err(err))
		os.Exit(1)
	}

	// At every minute
	// entryID, err := scheduler.Register("* * * * *", task)
	entryID, err := scheduler.Register("@every 5s", task)
	if err != nil {
		log.Error("could not register task", tint.Err(err))
	}
	log.Info("registered an entry", slog.String("id", entryID))

	if err := scheduler.Run(); err != nil {
		log.Error("could not run scheduler", tint.Err(err))
	}
}
