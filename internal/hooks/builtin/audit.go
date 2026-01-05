package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/keanuharrell/a9s/internal/core"
)

// =============================================================================
// Audit Hook
// =============================================================================

// AuditHook logs security-relevant events to a dedicated audit log.
// It captures actions that modify resources, security checks, and access patterns.
type AuditHook struct {
	name     string
	mu       sync.Mutex
	file     *os.File
	filePath string
	enabled  bool

	// Filters
	eventTypes    []core.EventType
	includeSource []string // Only audit these sources (empty = all)
	excludeSource []string // Exclude these sources

	// Rotation
	maxSize    int64 // Max file size in bytes before rotation
	maxBackups int   // Number of backup files to keep
}

// AuditOption configures the audit hook.
type AuditOption func(*AuditHook)

// WithAuditFile sets the audit log file path.
func WithAuditFile(path string) AuditOption {
	return func(h *AuditHook) {
		h.filePath = path
	}
}

// WithAuditEventTypes sets which event types to audit.
func WithAuditEventTypes(types []core.EventType) AuditOption {
	return func(h *AuditHook) {
		h.eventTypes = types
	}
}

// WithAuditIncludeSources limits auditing to specific sources.
func WithAuditIncludeSources(sources []string) AuditOption {
	return func(h *AuditHook) {
		h.includeSource = sources
	}
}

// WithAuditExcludeSources excludes specific sources from auditing.
func WithAuditExcludeSources(sources []string) AuditOption {
	return func(h *AuditHook) {
		h.excludeSource = sources
	}
}

// WithAuditRotation configures log rotation.
func WithAuditRotation(maxSize int64, maxBackups int) AuditOption {
	return func(h *AuditHook) {
		h.maxSize = maxSize
		h.maxBackups = maxBackups
	}
}

// NewAuditHook creates a new audit hook.
func NewAuditHook(enabled bool, opts ...AuditOption) *AuditHook {
	h := &AuditHook{
		name:       "audit",
		enabled:    enabled,
		filePath:   defaultAuditPath(),
		maxSize:    10 * 1024 * 1024, // 10MB default
		maxBackups: 5,
		eventTypes: []core.EventType{
			// Action events (most important for audit)
			core.EventActionStarted,
			core.EventActionExecuted,
			core.EventActionFailed,

			// Resource changes
			core.EventResourceCreated,
			core.EventResourceUpdated,
			core.EventResourceDeleted,

			// Security-relevant
			core.EventServiceHealthCheck,
			core.EventError,
		},
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// defaultAuditPath returns the default audit log path.
func defaultAuditPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/a9s-audit.log"
	}
	return filepath.Join(home, ".config", "a9s", "audit.log")
}

// =============================================================================
// Hook Interface Implementation
// =============================================================================

// Name returns the hook name.
func (h *AuditHook) Name() string {
	return h.name
}

// EventTypes returns the event types this hook handles.
func (h *AuditHook) EventTypes() []core.EventType {
	return h.eventTypes
}

// Priority returns the execution priority.
func (h *AuditHook) Priority() int {
	return 90 // High priority - audit should run early
}

// Handle writes the event to the audit log.
func (h *AuditHook) Handle(_ context.Context, event core.Event) error {
	if !h.enabled {
		return nil
	}

	// Check source filters
	if !h.shouldAudit(event.Source()) {
		return nil
	}

	// Ensure file is open
	if err := h.ensureOpen(); err != nil {
		return fmt.Errorf("audit: failed to open log: %w", err)
	}

	// Create audit record
	record := h.createRecord(event)

	// Write to file
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("audit: failed to marshal record: %w", err)
	}

	if _, err := h.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("audit: failed to write record: %w", err)
	}

	// Check for rotation
	_ = h.checkRotation()

	return nil
}

// =============================================================================
// Audit Record
// =============================================================================

// AuditRecord represents a single audit log entry.
type AuditRecord struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`
	Source    string    `json:"source"`
	Action    string    `json:"action,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	Success   *bool     `json:"success,omitempty"`
	Error     string    `json:"error,omitempty"`
	Details   any       `json:"details,omitempty"`
}

func (h *AuditHook) createRecord(event core.Event) AuditRecord {
	record := AuditRecord{
		Timestamp: event.Timestamp(),
		EventType: string(event.Type()),
		Source:    event.Source(),
	}

	// Extract relevant fields based on data type
	switch d := event.Data().(type) {
	case core.ActionEventData:
		record.Action = d.Action
		record.Resource = d.ResourceID
		if d.Result != nil {
			record.Success = &d.Result.Success
		}
		if d.Error != "" {
			record.Error = d.Error
		}
		if d.Params != nil {
			record.Details = d.Params
		}

	case core.ResourceEventData:
		record.Resource = d.ResourceID
		if d.Error != "" {
			record.Error = d.Error
		}
		if d.Count > 0 {
			record.Details = map[string]any{
				"resource_type": d.ResourceType,
				"count":         d.Count,
			}
		}

	case core.ServiceEventData:
		record.Source = d.ServiceName
		if d.Error != "" {
			record.Error = d.Error
		}
		record.Details = map[string]string{"status": d.Status}

	case map[string]string:
		if action, ok := d["action"]; ok {
			record.Action = action
		}
		if resource, ok := d["resource"]; ok {
			record.Resource = resource
		}
		if errStr, ok := d["error"]; ok {
			record.Error = errStr
		}
		record.Details = d

	case error:
		record.Error = d.Error()
	}

	return record
}

// =============================================================================
// File Management
// =============================================================================

func (h *AuditHook) ensureOpen() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.file != nil {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(h.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Open file
	f, err := os.OpenFile(h.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	h.file = f
	return nil
}

func (h *AuditHook) checkRotation() error {
	if h.file == nil || h.maxSize <= 0 {
		return nil
	}

	info, err := h.file.Stat()
	if err != nil {
		return err
	}

	if info.Size() < h.maxSize {
		return nil
	}

	return h.rotate()
}

func (h *AuditHook) rotate() error {
	// Close current file
	if h.file != nil {
		_ = h.file.Close()
		h.file = nil
	}

	// Rotate existing backups
	for i := h.maxBackups - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", h.filePath, i)
		newPath := fmt.Sprintf("%s.%d", h.filePath, i+1)
		_ = os.Rename(oldPath, newPath)
	}

	// Move current file to .1
	_ = os.Rename(h.filePath, h.filePath+".1")

	// Remove oldest backup if over limit
	if h.maxBackups > 0 {
		oldestPath := fmt.Sprintf("%s.%d", h.filePath, h.maxBackups+1)
		_ = os.Remove(oldestPath)
	}

	// Open new file
	return h.ensureOpen()
}

func (h *AuditHook) shouldAudit(source string) bool {
	// Check exclusions first
	for _, excluded := range h.excludeSource {
		if source == excluded {
			return false
		}
	}

	// If no inclusions specified, audit all
	if len(h.includeSource) == 0 {
		return true
	}

	// Check inclusions
	for _, included := range h.includeSource {
		if source == included {
			return true
		}
	}

	return false
}

// =============================================================================
// Lifecycle
// =============================================================================

// Close closes the audit log file.
func (h *AuditHook) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.file != nil {
		err := h.file.Close()
		h.file = nil
		return err
	}

	return nil
}

// SetEnabled enables or disables auditing.
func (h *AuditHook) SetEnabled(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = enabled
}

// IsEnabled returns whether auditing is enabled.
func (h *AuditHook) IsEnabled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.enabled
}

// FilePath returns the audit log file path.
func (h *AuditHook) FilePath() string {
	return h.filePath
}

// =============================================================================
// Interface Assertion
// =============================================================================

var _ core.Hook = (*AuditHook)(nil)
