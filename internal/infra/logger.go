package infra

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ctxKey string

const (
	RequestIDKey ctxKey = "request_id"
	UserIDKey    ctxKey = "user_id"
	ProjectKey   ctxKey = "project"
)

type AuditEntry struct {
	Timestamp  time.Time      `json:"timestamp"`
	Level      string         `json:"level"`
	RequestID  string         `json:"request_id,omitempty"`
	UserID     string         `json:"user_id,omitempty"`
	Project    string         `json:"project,omitempty"`
	Action     string         `json:"action"`
	Resource   string         `json:"resource,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
	Message    string         `json:"message,omitempty"`
	DurationMs int64          `json:"duration_ms,omitempty"`
	Success    bool           `json:"success"`
}

type Logger struct {
	slog      *slog.Logger
	level     slog.LevelVar
	logsDir   string
	auditMu   sync.Mutex
	auditFile *os.File
}

var (
	global     *Logger
	globalOnce sync.Once
)

func getLogsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ds-ac", "logs")
}

func NewLogger(logsDir string) *Logger {
	if logsDir == "" {
		logsDir = getLogsDir()
	}
	_ = os.MkdirAll(logsDir, 0755)

	var lvl slog.LevelVar
	lvl.Set(slog.LevelInfo)
	if envLvl := os.Getenv("DSAC_LOG_LEVEL"); envLvl != "" {
		switch envLvl {
		case "debug":
			lvl.Set(slog.LevelDebug)
		case "warn":
			lvl.Set(slog.LevelWarn)
		case "error":
			lvl.Set(slog.LevelError)
		}
	}

	logFilePath := filepath.Join(logsDir, "etoolcode.log")
	logFile, _ := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	var writer io.Writer = os.Stdout
	if logFile != nil {
		writer = io.MultiWriter(os.Stdout, logFile)
	}

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level:     &lvl,
		AddSource: true,
	})

	auditPath := filepath.Join(logsDir, "audit.jsonl")
	auditFile, _ := os.OpenFile(auditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	return &Logger{
		slog:      slog.New(handler),
		level:     lvl,
		logsDir:   logsDir,
		auditFile: auditFile,
	}
}

func InitGlobal() *Logger {
	globalOnce.Do(func() {
		global = NewLogger("")
	})
	return global
}

func L() *Logger {
	if global == nil {
		return InitGlobal()
	}
	return global
}

func (l *Logger) Info(msg string, args ...any)  { l.slog.Info(msg, args...) }
func (l *Logger) Debug(msg string, args ...any) { l.slog.Debug(msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.slog.Warn(msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.slog.Error(msg, args...) }

func (l *Logger) InfoCtx(ctx context.Context, msg string, args ...any) {
	l.slog.InfoContext(ctx, msg, l.enrichCtx(ctx, args)...)
}
func (l *Logger) ErrorCtx(ctx context.Context, msg string, args ...any) {
	l.slog.ErrorContext(ctx, msg, l.enrichCtx(ctx, args)...)
}

func (l *Logger) enrichCtx(ctx context.Context, args []any) []any {
	if ctx == nil {
		return args
	}
	extra := []any{}
	if v := ctx.Value(RequestIDKey); v != nil {
		extra = append(extra, "request_id", v)
	}
	if v := ctx.Value(UserIDKey); v != nil {
		extra = append(extra, "user_id", v)
	}
	if v := ctx.Value(ProjectKey); v != nil {
		extra = append(extra, "project", v)
	}
	return append(extra, args...)
}

func (l *Logger) Audit(ctx context.Context, entry AuditEntry) {
	entry.Timestamp = time.Now()
	if entry.Level == "" {
		entry.Level = "audit"
	}
	if ctx != nil {
		if v := ctx.Value(RequestIDKey); v != nil {
			if s, ok := v.(string); ok {
				entry.RequestID = s
			}
		}
		if v := ctx.Value(UserIDKey); v != nil {
			if s, ok := v.(string); ok {
				entry.UserID = s
			}
		}
		if v := ctx.Value(ProjectKey); v != nil {
			if s, ok := v.(string); ok && entry.Project == "" {
				entry.Project = s
			}
		}
	}
	l.auditMu.Lock()
	defer l.auditMu.Unlock()
	if l.auditFile != nil {
		b, _ := json.Marshal(entry)
		_, _ = l.auditFile.Write(append(b, '\n'))
	}
	l.slog.Info("AUDIT", "action", entry.Action, "resource", entry.Resource, "success", entry.Success, "project", entry.Project, "duration_ms", entry.DurationMs)
}

func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDKey, id)
}
