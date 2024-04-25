package db_test

// REDIS MUST BE RUNNING

import (
	"context"
	"testing"

	"exp1/db"
)

func TestGetScheduleConfigs(t *testing.T) {
	ctx := context.Background()
	configs, err := db.GetScheduleConfigs(ctx)
	if err != nil {
		t.Fatalf("db.GetScheduleConfigs failed: %v", err)
	}
	if len(configs) == 0 {
		t.Fatalf("no schedule configs found")
	}
	for config, ids := range configs {
		t.Logf("config: %v, ids: %v", config, ids)
	}
}
