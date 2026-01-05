package core

import "time"

// =============================================================================
// AWS Configuration Types
// =============================================================================

// AWSConfig holds AWS connection configuration.
type AWSConfig struct {
	Profile string        `yaml:"profile" json:"profile"`
	Region  string        `yaml:"region" json:"region"`
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
	Retry   RetryConfig   `yaml:"retry" json:"retry"`
}

// RetryConfig configures AWS API retry behavior.
type RetryConfig struct {
	MaxAttempts    int           `yaml:"max_attempts" json:"max_attempts"`
	InitialBackoff time.Duration `yaml:"initial_backoff" json:"initial_backoff"`
}

// =============================================================================
// Resource Types
// =============================================================================

// Resource represents any AWS resource with common attributes.
type Resource struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"` // e.g., "ec2:instance", "s3:bucket", "iam:role"
	Name      string            `json:"name"`
	ARN       string            `json:"arn,omitempty"`
	Region    string            `json:"region,omitempty"`
	State     string            `json:"state,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	Metadata  map[string]any    `json:"metadata,omitempty"`
	CreatedAt *time.Time        `json:"created_at,omitempty"`
	UpdatedAt *time.Time        `json:"updated_at,omitempty"`
}

// GetTag returns a tag value by key, with a default if not found.
func (r *Resource) GetTag(key, defaultValue string) string {
	if r.Tags == nil {
		return defaultValue
	}
	if v, ok := r.Tags[key]; ok {
		return v
	}
	return defaultValue
}

// GetMetadata returns a metadata value by key.
func (r *Resource) GetMetadata(key string) any {
	if r.Metadata == nil {
		return nil
	}
	return r.Metadata[key]
}

// GetMetadataString returns a metadata value as string.
func (r *Resource) GetMetadataString(key string) string {
	if v := r.GetMetadata(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ResourceSpec is the specification for creating or updating a resource.
type ResourceSpec struct {
	Type   string            `json:"type"`
	Name   string            `json:"name"`
	Region string            `json:"region,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
	Config map[string]any    `json:"config,omitempty"`
}

// ListOptions configures resource listing behavior.
type ListOptions struct {
	Filters    map[string]string `json:"filters,omitempty"`
	MaxResults int               `json:"max_results,omitempty"`
	NextToken  string            `json:"next_token,omitempty"`
	SortBy     string            `json:"sort_by,omitempty"`
	SortOrder  SortOrder         `json:"sort_order,omitempty"`
}

// SortOrder defines the sort direction.
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// ListResult contains the result of a list operation with pagination.
type ListResult struct {
	Resources  []Resource `json:"resources"`
	NextToken  string     `json:"next_token,omitempty"`
	TotalCount int        `json:"total_count,omitempty"`
}

// =============================================================================
// Progressive Loading Types
// =============================================================================

// UpdateType defines the type of resource update.
type UpdateType int

const (
	// UpdateTypeBatch indicates a batch of resources (initial load).
	UpdateTypeBatch UpdateType = iota
	// UpdateTypeSingle indicates a single resource update (enrichment).
	UpdateTypeSingle
)

// ResourceUpdate represents an update to resources during progressive loading.
type ResourceUpdate struct {
	Type      UpdateType // Batch or single update
	Resources []Resource // For batch updates
	Resource  *Resource  // For single updates
	Index     int        // Index of the resource being updated
}

// =============================================================================
// Action Types
// =============================================================================

// Action represents an executable operation on a resource.
type Action struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Icon        string            `json:"icon,omitempty"`
	Shortcut    string            `json:"shortcut,omitempty"`
	Parameters  []ActionParameter `json:"parameters,omitempty"`
	Dangerous   bool              `json:"dangerous"` // Requires confirmation
	Category    string            `json:"category,omitempty"`
}

// ActionParameter defines a parameter for an action.
type ActionParameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // string, int, bool, select, duration
	Required    bool     `json:"required"`
	Default     any      `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"` // For select type
	Description string   `json:"description,omitempty"`
	Validation  string   `json:"validation,omitempty"` // Regex pattern
}

// ActionResult contains the result of executing an action.
type ActionResult struct {
	Success   bool          `json:"success"`
	Message   string        `json:"message"`
	Data      any           `json:"data,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// NewActionResult creates a new ActionResult.
func NewActionResult(success bool, message string) *ActionResult {
	return &ActionResult{
		Success:   success,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// WithData adds data to the action result.
func (r *ActionResult) WithData(data any) *ActionResult {
	r.Data = data
	return r
}

// WithDuration sets the duration of the action.
func (r *ActionResult) WithDuration(d time.Duration) *ActionResult {
	r.Duration = d
	return r
}

// =============================================================================
// Event Types
// =============================================================================

// EventType defines the type of system event.
type EventType string

const (
	// Service events
	EventServiceRegistered   EventType = "service.registered"
	EventServiceUnregistered EventType = "service.unregistered"
	EventServiceHealthCheck  EventType = "service.health_check"

	// Resource events
	EventResourceListed  EventType = "resource.listed"
	EventResourceGet     EventType = "resource.get"
	EventResourceCreated EventType = "resource.created"
	EventResourceUpdated EventType = "resource.updated"
	EventResourceDeleted EventType = "resource.deleted"

	// Action events
	EventActionStarted  EventType = "action.started"
	EventActionExecuted EventType = "action.executed"
	EventActionFailed   EventType = "action.failed"

	// Plugin events
	EventPluginLoaded   EventType = "plugin.loaded"
	EventPluginUnloaded EventType = "plugin.unloaded"
	EventPluginError    EventType = "plugin.error"

	// Config events
	EventConfigChanged  EventType = "config.changed"
	EventConfigReloaded EventType = "config.reloaded"

	// TUI events
	EventViewChanged EventType = "view.changed"
	EventViewRefresh EventType = "view.refresh"

	// General events
	EventError   EventType = "error"
	EventWarning EventType = "warning"
	EventInfo    EventType = "info"
)

// BaseEvent is a basic implementation of the Event interface.
type BaseEvent struct {
	eventType EventType
	timestamp time.Time
	source    string
	data      any
}

// NewEvent creates a new BaseEvent.
func NewEvent(eventType EventType, source string, data any) *BaseEvent {
	return &BaseEvent{
		eventType: eventType,
		timestamp: time.Now(),
		source:    source,
		data:      data,
	}
}

// Type implements Event.Type.
func (e *BaseEvent) Type() EventType { return e.eventType }

// Timestamp implements Event.Timestamp.
func (e *BaseEvent) Timestamp() time.Time { return e.timestamp }

// Source implements Event.Source.
func (e *BaseEvent) Source() string { return e.source }

// Data implements Event.Data.
func (e *BaseEvent) Data() any { return e.data }

// =============================================================================
// Common Event Data Types
// =============================================================================

// ResourceEventData contains data for resource-related events.
type ResourceEventData struct {
	ResourceID   string `json:"resource_id,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	Count        int    `json:"count,omitempty"`
	Error        string `json:"error,omitempty"`
}

// ActionEventData contains data for action-related events.
type ActionEventData struct {
	Action     string         `json:"action"`
	ResourceID string         `json:"resource_id,omitempty"`
	Params     map[string]any `json:"params,omitempty"`
	Result     *ActionResult  `json:"result,omitempty"`
	Error      string         `json:"error,omitempty"`
}

// ServiceEventData contains data for service-related events.
type ServiceEventData struct {
	ServiceName string `json:"service_name"`
	Status      string `json:"status,omitempty"`
	Error       string `json:"error,omitempty"`
}

// =============================================================================
// State Constants
// =============================================================================

// Common resource states
const (
	StateRunning    = "running"
	StateStopped    = "stopped"
	StatePending    = "pending"
	StateTerminated = "terminated"
	StateActive     = "active"
	StateInactive   = "inactive"
	StateDeleting   = "deleting"
	StateCreating   = "creating"
	StateUpdating   = "updating"
	StateAvailable  = "available"
	StateError      = "error"
	StateUnknown    = "unknown"
	StateWarning    = "warning"
)

// =============================================================================
// Output Format Constants
// =============================================================================

// OutputFormat defines supported output formats.
type OutputFormat string

const (
	FormatJSON  OutputFormat = "json"
	FormatTable OutputFormat = "table"
	FormatYAML  OutputFormat = "yaml"
	FormatCSV   OutputFormat = "csv"
)

// =============================================================================
// Service Metadata
// =============================================================================

// ServiceInfo contains metadata about a service.
type ServiceInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Version     string   `json:"version,omitempty"`
	Status      string   `json:"status"`
	Actions     []Action `json:"actions,omitempty"`
}

// ToServiceInfo creates ServiceInfo from an AWSService.
func ToServiceInfo(svc AWSService) ServiceInfo {
	info := ServiceInfo{
		Name:        svc.Name(),
		Description: svc.Description(),
		Icon:        svc.Icon(),
		Status:      StateActive,
	}
	if executor, ok := svc.(ActionExecutor); ok {
		info.Actions = executor.Actions()
	}
	return info
}
