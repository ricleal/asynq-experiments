package main

import (
	"errors"
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

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			// Specify how many concurrent workers to use
			Concurrency: 10,
			// Optionally specify multiple queues with different priority.
			Queues: map[string]int{
				"aws":  5,
				"cron": 5,
			},
			LogLevel: asynq.WarnLevel,
			//
			// If error is due to rate limit, don't count the error as a failure.
			IsFailure:      func(err error) bool { return !tasks.IsRateLimitError(err) },
			RetryDelayFunc: retryDelay,
		},
	)

	// mux maps a type to a handler
	mux := asynq.NewServeMux()
	mux.Handle(tasks.TypeEventStart, tasks.NewProcessStartEvent(log, client))
	mux.Handle(tasks.TypeEventStop, tasks.NewProcessStopEvent(log, client))
	mux.Handle(tasks.TypeEventAWS, tasks.NewProcessEventAWS(log))

	// Run server
	log.Info("starting server", slog.String("addr", redisAddr))
	if err := srv.Run(mux); err != nil {
		log.Error("could not run server", tint.Err(err))
		os.Exit(1)
	}
}

func retryDelay(n int, err error, task *asynq.Task) time.Duration {
	var ratelimitErr *tasks.RateLimitError
	if errors.As(err, &ratelimitErr) {
		return ratelimitErr.RetryIn
	}
	return asynq.DefaultRetryDelayFunc(n, err, task)
}
