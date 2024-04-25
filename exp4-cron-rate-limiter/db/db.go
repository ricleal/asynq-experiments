package db

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Redis will have keys and values
// schedule:<task_type> -> <cronspec>
// eg: schedule:notification:email -> "*/5 * * * *"
// eg: schedule:notification:push -> "@every 30s"

// Run populate first time
// redis-cli < data.redis

type ScheduleConfig struct {
	CronSpec string
	TaskType string
}

const redisAddr = "127.0.0.1:6379"

var (
	rdb  *redis.Client
	once sync.Once
)

func GetScheduleConfigs(ctx context.Context) (map[ScheduleConfig][]string, error) {
	once.Do(func() {
		rdb = redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: "", // no password set
			DB:       0,  // use default DB
		})
	})

	keys, err := rdb.Keys(ctx, "schedule:*").Result()
	if err != nil {
		return nil, fmt.Errorf("rdb.Keys failed: %v", err)
	}

	// map indexed by a config to a list of ids
	configs := make(map[ScheduleConfig][]string)
	for _, key := range keys {
		value, err := rdb.Get(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("rdb.Get failed: %v", err)
		}
		parts := strings.Split(key, ":")
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid key: %s", key)
		}
		taskType := strings.Join(parts[1:len(parts)-1], ":")
		id := parts[len(parts)-1]
		conf := ScheduleConfig{CronSpec: value, TaskType: taskType}
		// if the config is not in the map, add it
		if _, ok := configs[conf]; !ok {
			configs[conf] = []string{id}
			continue
		}
		// if the config is in the map, append the id
		configs[conf] = append(configs[conf], id)
	}
	return configs, nil
}
