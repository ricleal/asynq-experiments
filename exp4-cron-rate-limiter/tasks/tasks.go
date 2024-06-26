package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/lmittmann/tint"
	"golang.org/x/time/rate"
)

func Logger(w io.Writer, levelAsString string) *slog.Logger {
	var level slog.Level

	switch strings.ToLower(levelAsString) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "Error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logger := slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      level,
			TimeFormat: time.TimeOnly,
		}),
	)

	return logger
}

// A list of task types.
const (
	TypeEventStart = "event:start"
	TypeEventStop  = "event:stop"
	TypeEventAWS   = "event:aws"
)

type EventStart struct {
	EventUUID uuid.UUID
	IDs       []string
}

type EventStop struct {
	EventUUID uuid.UUID
	IDs       []string
}

type EventAWS struct {
	ARN string
}

// Task builders

func BuildEventStart(ids []string) (*asynq.Task, error) {
	payload, err := json.Marshal(EventStart{
		EventUUID: uuid.New(),
		IDs:       ids,
	})
	if err != nil {
		return nil, fmt.Errorf("json.Marshal failed: %v", err)
	}
	return asynq.NewTask(TypeEventStart, payload), nil
}

func BuildEventStop(ids []string) (*asynq.Task, error) {
	payload, err := json.Marshal(EventStart{
		EventUUID: uuid.New(),
		IDs:       ids,
	})
	if err != nil {
		return nil, fmt.Errorf("json.Marshal failed: %v", err)
	}
	return asynq.NewTask(TypeEventStop, payload), nil
}

func BuildEventAWS(arn string) (*asynq.Task, error) {
	payload, err := json.Marshal(EventAWS{ARN: arn})
	if err != nil {
		return nil, fmt.Errorf("json.Marshal failed: %v", err)
	}
	return asynq.NewTask(TypeEventAWS, payload), nil
}

// Handlers
type ProcessStartEvent struct {
	Log    *slog.Logger
	client *asynq.Client
}

func NewProcessStartEvent(log *slog.Logger, client *asynq.Client) *ProcessStartEvent {
	return &ProcessStartEvent{
		client: client,
		Log:    log.With(slog.String("event_type", TypeEventStart)),
	}
}

func (p *ProcessStartEvent) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var e EventStart
	if err := json.Unmarshal(t.Payload(), &e); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	p.Log.Info("✅ Enqueueing AWS start event", slog.String("event_uuid", e.EventUUID.String()),
		slog.Any("ids", e.IDs))

	for _, id := range e.IDs {
		// Enqueue AWS event
		task, err := BuildEventAWS("arn:aws:sns:us-east-1:123456789012:start-event/" + id)
		if err != nil {
			return fmt.Errorf("BuildEventAWS failed: %v", err)
		}
		info, err := p.client.Enqueue(task, asynq.Queue("aws"))
		if err != nil {
			p.Log.Error("could not enqueue task", tint.Err(err))
			os.Exit(1)
		}
		p.Log.Info("enqueued task", slog.String("id", info.ID), slog.String("queue", info.Queue), slog.Any("state", info.State),
			slog.String("task_type", task.Type()))
	}
	return nil
}

type ProcessStopEvent struct {
	Log    *slog.Logger
	client *asynq.Client
}

func NewProcessStopEvent(log *slog.Logger, client *asynq.Client) *ProcessStopEvent {
	return &ProcessStopEvent{
		client: client,
		Log:    log.With(slog.String("event_type", TypeEventStop)),
	}
}

func (p *ProcessStopEvent) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var e EventStop
	if err := json.Unmarshal(t.Payload(), &e); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	p.Log.Info("🚫 Enqueueing AWS stop event", slog.String("event_uuid", e.EventUUID.String()),
		slog.Any("ids", e.IDs))

	for _, id := range e.IDs {
		// Enqueue AWS event
		task, err := BuildEventAWS("arn:aws:sns:us-east-1:123456789012:stop-event/" + id)
		if err != nil {
			return fmt.Errorf("BuildEventAWS failed: %v", err)
		}
		info, err := p.client.Enqueue(task, asynq.Queue("aws"))
		if err != nil {
			p.Log.Error("could not enqueue task", tint.Err(err))
			os.Exit(1)
		}
		p.Log.Info("enqueued task", slog.String("id", info.ID), slog.String("queue", info.Queue), slog.Any("state", info.State),
			slog.String("task_type", task.Type()))
	}
	return nil
}

type ProcessEventAWS struct {
	Log     *slog.Logger
	limiter *rate.Limiter
}

func NewProcessEventAWS(log *slog.Logger) *ProcessEventAWS {
	return &ProcessEventAWS{
		Log: log.With(slog.String("event_type", TypeEventAWS)),
		// Rate is 5 events/sec and permits burst of at most 10 events.
		limiter: rate.NewLimiter(5, 10),
	}
}

func (p *ProcessEventAWS) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var e EventAWS
	if err := json.Unmarshal(t.Payload(), &e); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	if !p.limiter.Allow() {
		p.Log.Warn("❗rate limited", slog.String("arn", e.ARN))
		return &RateLimitError{
			RetryIn: time.Duration(rand.Intn(3)) * time.Second,
		}
	}

	p.Log.Info("🚀 Processing Event AWS", slog.String("arn", e.ARN))
	return nil
}

type RateLimitError struct {
	RetryIn time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited (retry in  %v)", e.RetryIn)
}

func IsRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}
