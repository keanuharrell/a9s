// Package builtin provides built-in hooks for a9s.
package builtin

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/keanuharrell/a9s/internal/core"
)

// =============================================================================
// Logging Hook
// =============================================================================

// LoggingHook logs events to a writer.
type LoggingHook struct {
	name       string
	logger     *log.Logger
	level      LogLevel
	eventTypes []core.EventType
	format     LogFormat
}

// LogLevel defines the minimum level to log.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// LogFormat defines the output format.
type LogFormat int

const (
	LogFormatText LogFormat = iota
	LogFormatJSON
)

// LoggingOption configures the logging hook.
type LoggingOption func(*LoggingHook)

// WithLogLevel sets the minimum log level.
func WithLogLevel(level LogLevel) LoggingOption {
	return func(h *LoggingHook) {
		h.level = level
	}
}

// WithLogFormat sets the output format.
func WithLogFormat(format LogFormat) LoggingOption {
	return func(h *LoggingHook) {
		h.format = format
	}
}

// WithLogWriter sets the output writer.
func WithLogWriter(w io.Writer) LoggingOption {
	return func(h *LoggingHook) {
		h.logger = log.New(w, "", 0)
	}
}

// WithLogEventTypes sets which event types to log.
func WithLogEventTypes(types []core.EventType) LoggingOption {
	return func(h *LoggingHook) {
		h.eventTypes = types
	}
}

// NewLoggingHook creates a new logging hook.
func NewLoggingHook(opts ...LoggingOption) *LoggingHook {
	h := &LoggingHook{
		name:   "logging",
		logger: log.New(os.Stdout, "", 0),
		level:  LogLevelInfo,
		format: LogFormatText,
		eventTypes: []core.EventType{
			core.EventServiceRegistered,
			core.EventServiceUnregistered,
			core.EventResourceListed,
			core.EventActionStarted,
			core.EventActionExecuted,
			core.EventActionFailed,
			core.EventPluginLoaded,
			core.EventPluginUnloaded,
			core.EventError,
			core.EventWarning,
		},
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// =============================================================================
// Hook Interface Implementation
// =============================================================================

// Name returns the hook name.
func (h *LoggingHook) Name() string {
	return h.name
}

// EventTypes returns the event types this hook handles.
func (h *LoggingHook) EventTypes() []core.EventType {
	return h.eventTypes
}

// Priority returns the execution priority.
func (h *LoggingHook) Priority() int {
	return 100 // High priority - logging should run first
}

// Handle logs the event.
func (h *LoggingHook) Handle(_ context.Context, event core.Event) error {
	level := h.eventLevel(event.Type())
	if level < h.level {
		return nil
	}

	if h.format == LogFormatJSON {
		h.logJSON(event, level)
	} else {
		h.logText(event, level)
	}

	return nil
}

// =============================================================================
// Logging Implementation
// =============================================================================

func (h *LoggingHook) eventLevel(eventType core.EventType) LogLevel {
	switch eventType {
	case core.EventError, core.EventActionFailed, core.EventPluginError:
		return LogLevelError
	case core.EventWarning:
		return LogLevelWarn
	case core.EventServiceRegistered, core.EventServiceUnregistered,
		core.EventActionStarted, core.EventActionExecuted,
		core.EventPluginLoaded, core.EventPluginUnloaded:
		return LogLevelInfo
	default:
		return LogLevelDebug
	}
}

func (h *LoggingHook) logText(event core.Event, level LogLevel) {
	levelStr := h.levelString(level)
	timestamp := event.Timestamp().Format("15:04:05")

	// Format data based on type
	dataStr := h.formatData(event.Data())

	h.logger.Printf("[%s] %s [%s] %s: %s",
		timestamp,
		levelStr,
		event.Type(),
		event.Source(),
		dataStr,
	)
}

func (h *LoggingHook) logJSON(event core.Event, level LogLevel) {
	// Simple JSON output
	h.logger.Printf(`{"timestamp":"%s","level":"%s","event":"%s","source":"%s","data":%s}`,
		event.Timestamp().Format(time.RFC3339),
		h.levelString(level),
		event.Type(),
		event.Source(),
		h.formatDataJSON(event.Data()),
	)
}

func (h *LoggingHook) levelString(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func (h *LoggingHook) formatData(data any) string {
	if data == nil {
		return ""
	}

	switch d := data.(type) {
	case core.ResourceEventData:
		if d.Error != "" {
			return fmt.Sprintf("error=%s", d.Error)
		}
		if d.Count > 0 {
			return fmt.Sprintf("type=%s count=%d", d.ResourceType, d.Count)
		}
		return fmt.Sprintf("resource=%s type=%s", d.ResourceID, d.ResourceType)

	case core.ActionEventData:
		if d.Error != "" {
			return fmt.Sprintf("action=%s resource=%s error=%s", d.Action, d.ResourceID, d.Error)
		}
		if d.Result != nil {
			return fmt.Sprintf("action=%s resource=%s success=%t", d.Action, d.ResourceID, d.Result.Success)
		}
		return fmt.Sprintf("action=%s resource=%s", d.Action, d.ResourceID)

	case core.ServiceEventData:
		if d.Error != "" {
			return fmt.Sprintf("service=%s error=%s", d.ServiceName, d.Error)
		}
		return fmt.Sprintf("service=%s status=%s", d.ServiceName, d.Status)

	case map[string]string:
		parts := make([]string, 0, len(d))
		for k, v := range d {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
		return strings.Join(parts, " ")

	case string:
		return d

	case error:
		return d.Error()

	default:
		return fmt.Sprintf("%v", d)
	}
}

func (h *LoggingHook) formatDataJSON(data any) string {
	if data == nil {
		return "null"
	}

	switch d := data.(type) {
	case core.ResourceEventData:
		return fmt.Sprintf(`{"resource_id":"%s","resource_type":"%s","count":%d,"error":"%s"}`,
			d.ResourceID, d.ResourceType, d.Count, d.Error)

	case core.ActionEventData:
		success := false
		if d.Result != nil {
			success = d.Result.Success
		}
		return fmt.Sprintf(`{"action":"%s","resource_id":"%s","success":%t,"error":"%s"}`,
			d.Action, d.ResourceID, success, d.Error)

	case core.ServiceEventData:
		return fmt.Sprintf(`{"service":"%s","status":"%s","error":"%s"}`,
			d.ServiceName, d.Status, d.Error)

	case string:
		return fmt.Sprintf(`"%s"`, d)

	case error:
		return fmt.Sprintf(`{"error":"%s"}`, d.Error())

	default:
		return fmt.Sprintf(`"%v"`, d)
	}
}

// =============================================================================
// Interface Assertion
// =============================================================================

var _ core.Hook = (*LoggingHook)(nil)
