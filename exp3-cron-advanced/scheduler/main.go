package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"exp1/db"
	"exp1/tasks"

	"github.com/go-faker/faker/v4"
	"github.com/hibiken/asynq"
	"github.com/lmittmann/tint"
)

const redisAddr = "127.0.0.1:6379"

type PeriodicTasks struct {
	log *slog.Logger
}

func NewPeriodicTasks(log *slog.Logger) *PeriodicTasks {
	return &PeriodicTasks{
		log: log.With(slog.String("name", "periodic_tasks")),
	}
}

func (p *PeriodicTasks) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
	// TODO: propagate context
	ctx := context.Background()
	configs, err := db.GetScheduleConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.GetScheduleConfigs failed: %v", err)
	}

	var periodicTaskConfig []*asynq.PeriodicTaskConfig
	for _, config := range configs {
		var task *asynq.Task
		var err error
		switch config.TaskType {
		case tasks.TypeNotificationEmail:
			task, err = tasks.BuildNotificationEmail(faker.Email(), faker.Sentence(), faker.Paragraph())
		case tasks.TypeNotificationSMS:
			task, err = tasks.BuildNotificationSMS(faker.Phonenumber(), faker.Sentence())
		case tasks.TypeNotificationPush:
			task, err = tasks.BuildNotificationPush(faker.Phonenumber(), faker.Sentence())
		default:
			p.log.Warn("unknown task type", slog.String("task_type", config.TaskType))
			continue
		}
		if err != nil {
			p.log.Error("could not create task", tint.Err(err))
			continue
		}

		p.log.Info("adding task", slog.String("task_type", config.TaskType), slog.String("cron_spec", config.CronSpec))
		periodicTaskConfig = append(periodicTaskConfig, &asynq.PeriodicTaskConfig{
			Cronspec: config.CronSpec,
			Task:     task,
		})
	}

	return periodicTaskConfig, nil
}

func main() {
	log := tasks.Logger(os.Stderr, os.Getenv("LOG_LEVEL"))

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}

	provider := NewPeriodicTasks(log)

	manager, err := asynq.NewPeriodicTaskManager(
		asynq.PeriodicTaskManagerOpts{
			RedisConnOpt:               asynq.RedisClientOpt{Addr: "localhost:6379"},
			PeriodicTaskConfigProvider: provider,         // this provider object is the interface to your config source
			SyncInterval:               10 * time.Second, // this field specifies how often sync should happen
			SchedulerOpts: &asynq.SchedulerOpts{
				Location: loc,
				LogLevel: asynq.WarnLevel,
			},
		})
	if err != nil {
		log.Error("could not create manager", tint.Err(err))
		os.Exit(1)
	}

	log.Info("starting manager", slog.String("addr", "localhost:6379"))
	if err := manager.Run(); err != nil {
		log.Error("could not run manager", tint.Err(err))
	}
}
