package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/lmittmann/tint"
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
	TypeNotificationEmail = "notification:email"
	TypeNotificationSMS   = "notification:sms"
	TypeNotificationPush  = "notification:push"
)

type NotificationEmail struct {
	Recipient string
	Subject   string
	Body      string
}

type NotificationSMS struct {
	Recipient string
	Body      string
}

type NotificationPush struct {
	Recipient string
	Body      string
}

// Task builders

func BuildNotificationEmail(recipient, subject, body string) (*asynq.Task, error) {
	payload, err := json.Marshal(NotificationEmail{Recipient: recipient, Subject: subject, Body: body})
	if err != nil {
		return nil, fmt.Errorf("json.Marshal failed: %v", err)
	}
	return asynq.NewTask(TypeNotificationEmail, payload), nil
}

func BuildNotificationSMS(recipient, body string) (*asynq.Task, error) {
	payload, err := json.Marshal(NotificationSMS{Recipient: recipient, Body: body})
	if err != nil {
		return nil, fmt.Errorf("json.Marshal failed: %v", err)
	}
	return asynq.NewTask(TypeNotificationSMS, payload), nil
}

func BuildNotificationPush(recipient, body string) (*asynq.Task, error) {
	payload, err := json.Marshal(NotificationPush{Recipient: recipient, Body: body})
	if err != nil {
		return nil, fmt.Errorf("json.Marshal failed: %v", err)
	}
	return asynq.NewTask(TypeNotificationPush, payload, asynq.MaxRetry(5), asynq.Timeout(20*time.Minute)), nil
}

// Handlers
// Must implement asynq.Handler interface.
// type HandlerFunc func(context.Context, *Task) error
// func (fn HandlerFunc) ProcessTask(ctx context.Context, task *Task) error
type ProcessNotificationEmail struct {
	Sender string
	Log    *slog.Logger
}

func NewProcessNotificationEmail(log *slog.Logger) *ProcessNotificationEmail {
	return &ProcessNotificationEmail{
		Sender: "experiments@asynq",
		Log:    log.With(slog.String("sender", "experiments@asynq")),
	}
}

func (p *ProcessNotificationEmail) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var n NotificationEmail
	if err := json.Unmarshal(t.Payload(), &n); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	// Compose and send email
	p.Log.Info("📨 Sending Email", slog.String("sender", p.Sender), slog.String("recipient", n.Recipient), slog.String("subject", n.Subject), slog.String("body", n.Body))
	return nil
}

func HandleNotificationSMS(ctx context.Context, t *asynq.Task) error {
	var n NotificationSMS
	if err := json.Unmarshal(t.Payload(), &n); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	// Send SMS
	// TODO: implement
	fmt.Println("📟 TODO: Send SMS to", n.Recipient, "🔧")
	return nil
}

func HandleNotificationPush(ctx context.Context, t *asynq.Task) error {
	var n NotificationPush
	if err := json.Unmarshal(t.Payload(), &n); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	// Send push notification
	// TODO: implement
	fmt.Println("📱 TODO: Send push notification to", n.Recipient, "🔧")
	return nil
}
