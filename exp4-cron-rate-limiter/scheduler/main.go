package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"exp1/db"
	"exp1/tasks"

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

	p.log.Debug("GetConfigs called", slog.Any("configs", configs))

	var periodicTaskConfig []*asynq.PeriodicTaskConfig
	for config, ids := range configs {
		var err error
		var task *asynq.Task
		switch config.TaskType {
		case tasks.TypeEventStart:
			task, err = tasks.BuildEventStart(ids)
			if err != nil {
				p.log.Error("could not create start task", tint.Err(err))
				continue
			}
		case tasks.TypeEventStop:
			task, err = tasks.BuildEventStop(ids)
			if err != nil {
				p.log.Error("could not create stop task", tint.Err(err))
				continue
			}
		default:
			p.log.Warn("unknown task type", slog.String("task_type", config.TaskType))
			continue
		}

		p.log.Info("adding task", slog.String("task_type", config.TaskType),
			slog.String("cron_spec", config.CronSpec), slog.Any("ids", ids))
		periodicTaskConfig = append(periodicTaskConfig, &asynq.PeriodicTaskConfig{
			Cronspec: config.CronSpec,
			Task:     task,
			Opts: []asynq.Option{
				asynq.Queue("cron"), // All cron tasks go to the "cron" queue (event start and stop)
			},
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
			PeriodicTaskConfigProvider: provider,         // struct that must implement the GetConfigs() method
			SyncInterval:               10 * time.Second, // how often the GetConfigs() should be called
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
