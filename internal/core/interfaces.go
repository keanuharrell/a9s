// Package core defines the core interfaces and contracts for the a9s application.
// All services, views, plugins, and hooks must implement these interfaces.
package core

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// =============================================================================
// AWS Service Interfaces
// =============================================================================

// AWSService is the base interface for all AWS service implementations.
// Every AWS service (EC2, IAM, S3, Lambda, etc.) must implement this interface.
type AWSService interface {
	// Name returns the unique identifier for this service (e.g., "ec2", "iam", "s3")
	Name() string

	// Description returns a human-readable description of the service
	Description() string

	// Icon returns an icon identifier for the service (used in TUI)
	Icon() string

	// Initialize sets up the service with the given AWS configuration
	Initialize(ctx context.Context, cfg *AWSConfig) error

	// Close releases any resources held by the service
	Close() error

	// HealthCheck verifies the service can communicate with AWS
	HealthCheck(ctx context.Context) error
}

// ResourceLister provides the capability to list AWS resources.
type ResourceLister interface {
	AWSService

	// List returns resources matching the given options
	List(ctx context.Context, opts ListOptions) ([]Resource, error)
}

// ResourceGetter provides the capability to get a specific resource by ID.
type ResourceGetter interface {
	AWSService

	// Get returns a specific resource by its ID
	Get(ctx context.Context, id string) (*Resource, error)
}

// ResourceMutator provides the capability to create, update, and delete resources.
type ResourceMutator interface {
	AWSService

	// Create creates a new resource based on the specification
	Create(ctx context.Context, spec ResourceSpec) (*Resource, error)

	// Update modifies an existing resource
	Update(ctx context.Context, id string, spec ResourceSpec) (*Resource, error)

	// Delete removes a resource by its ID
	Delete(ctx context.Context, id string) error
}

// ActionExecutor provides the capability to execute custom actions on resources.
type ActionExecutor interface {
	AWSService

	// Actions returns the list of available actions for this service
	Actions() []Action

	// Execute runs the specified action on the given resource
	Execute(ctx context.Context, action string, resourceID string, params map[string]any) (*ActionResult, error)
}

// =============================================================================
// TUI View Interfaces
// =============================================================================

// View represents a TUI view for displaying and interacting with a service.
// Each service should have a corresponding View implementation.
type View interface {
	tea.Model

	// Name returns the view's display name
	Name() string

	// Shortcut returns the keyboard shortcut to switch to this view (e.g., "1", "2")
	Shortcut() string

	// ServiceName returns the name of the service this view is associated with
	ServiceName() string

	// SetService associates an AWS service with this view
	SetService(service AWSService)

	// SetDimensions updates the view's width and height
	SetDimensions(width, height int)

	// Refresh triggers a data reload and returns a command
	Refresh() tea.Cmd

	// IsLoading returns whether the view is currently loading data
	IsLoading() bool

	// Error returns the current error state, if any
	Error() error
}

// ViewFactory creates View instances for services.
type ViewFactory interface {
	// Create creates a new view for the given service
	Create(service AWSService) (View, error)

	// ServiceName returns the name of the service this factory creates views for
	ServiceName() string
}

// =============================================================================
// Plugin Interfaces
// =============================================================================

// Plugin represents a loadable extension that can add services, views, and hooks.
type Plugin interface {
	// Manifest returns the plugin's metadata
	Manifest() PluginManifest

	// Initialize sets up the plugin with access to the DI container
	Initialize(ctx context.Context, container Container) error

	// Start activates the plugin
	Start() error

	// Stop deactivates the plugin and releases resources
	Stop() error

	// Services returns the service registrations provided by this plugin
	Services() []ServiceRegistration

	// Views returns the view registrations provided by this plugin
	Views() []ViewRegistration

	// Hooks returns the hook registrations provided by this plugin
	Hooks() []HookRegistration
}

// PluginManifest contains metadata about a plugin.
type PluginManifest struct {
	Name        string   `yaml:"name" json:"name"`
	Version     string   `yaml:"version" json:"version"`
	Description string   `yaml:"description" json:"description"`
	Author      string   `yaml:"author" json:"author"`
	Requires    []string `yaml:"requires" json:"requires"`       // Required plugins or services
	Permissions []string `yaml:"permissions" json:"permissions"` // Required AWS permissions
}

// =============================================================================
// Hook/Event Interfaces
// =============================================================================

// Event represents a system event that hooks can respond to.
type Event interface {
	// Type returns the event type
	Type() EventType

	// Timestamp returns when the event occurred
	Timestamp() time.Time

	// Source returns the origin of the event (e.g., service name)
	Source() string

	// Data returns the event payload
	Data() any
}

// Hook is a handler that responds to system events.
type Hook interface {
	// Name returns the unique identifier for this hook
	Name() string

	// EventTypes returns the event types this hook handles
	EventTypes() []EventType

	// Priority returns the execution priority (higher = runs first)
	Priority() int

	// Handle processes an event
	Handle(ctx context.Context, event Event) error
}

// HookHandler is a function type for processing events.
type HookHandler func(ctx context.Context, event Event) error

// HookMiddleware wraps hook execution to add cross-cutting concerns.
type HookMiddleware interface {
	// Wrap wraps the next handler with middleware logic
	Wrap(next HookHandler) HookHandler
}

// EventDispatcher manages event dispatch to registered hooks.
type EventDispatcher interface {
	// Register adds a hook to the dispatcher
	Register(hook Hook)

	// Unregister removes a hook by name
	Unregister(name string)

	// Dispatch sends an event to all registered hooks
	Dispatch(ctx context.Context, event Event) error

	// Use adds middleware to the dispatch chain
	Use(middleware HookMiddleware)
}

// =============================================================================
// Configuration Interfaces
// =============================================================================

// ConfigProvider provides access to configuration values.
type ConfigProvider interface {
	// Get returns a configuration value by key
	Get(key string) any

	// GetString returns a string configuration value
	GetString(key string) string

	// GetInt returns an integer configuration value
	GetInt(key string) int

	// GetBool returns a boolean configuration value
	GetBool(key string) bool

	// GetDuration returns a duration configuration value
	GetDuration(key string) time.Duration

	// GetStringSlice returns a string slice configuration value
	GetStringSlice(key string) []string

	// GetStringMap returns a string map configuration value
	GetStringMap(key string) map[string]any

	// Sub returns a new ConfigProvider scoped to a sub-key
	Sub(key string) ConfigProvider

	// IsSet returns whether a key is set
	IsSet(key string) bool
}

// ConfigWatcher watches for configuration changes and notifies observers.
type ConfigWatcher interface {
	// Watch registers a callback for configuration changes
	Watch(callback func(key string, oldValue, newValue any))

	// Stop stops watching for changes
	Stop()
}

// =============================================================================
// Container/DI Interfaces
// =============================================================================

// Container is the dependency injection container.
type Container interface {
	// Register registers a factory function for creating instances
	Register(name string, factory any)

	// RegisterSingleton registers a singleton instance
	RegisterSingleton(name string, instance any)

	// Resolve returns an instance by name
	Resolve(name string) (any, error)

	// MustResolve returns an instance by name or panics
	MustResolve(name string) any

	// ResolveAll returns all instances of a given type
	ResolveAll(typeName string) []any

	// Has checks if a registration exists
	Has(name string) bool

	// Start initializes all registered services
	Start(ctx context.Context) error

	// Stop shuts down all registered services
	Stop(ctx context.Context) error
}

// =============================================================================
// Registry Interfaces
// =============================================================================

// ServiceRegistry manages AWS service registrations.
type ServiceRegistry interface {
	// Register adds a service to the registry
	Register(service AWSService) error

	// Unregister removes a service from the registry
	Unregister(name string) error

	// Get returns a service by name
	Get(name string) (AWSService, error)

	// List returns all registered services
	List() []AWSService

	// Has checks if a service is registered
	Has(name string) bool

	// Watch registers a callback for registry changes
	Watch(callback func(event RegistryEvent))
}

// ViewRegistry manages TUI view registrations.
type ViewRegistry interface {
	// Register adds a view to the registry
	Register(view View) error

	// Unregister removes a view from the registry
	Unregister(name string) error

	// Get returns a view by name
	Get(name string) (View, error)

	// GetByShortcut returns a view by its keyboard shortcut
	GetByShortcut(shortcut string) (View, error)

	// List returns all registered views
	List() []View

	// ListOrdered returns all registered views in display order
	ListOrdered() []View

	// Has checks if a view is registered
	Has(name string) bool

	// Watch registers a callback for registry changes
	Watch(callback func(event RegistryEvent))
}

// =============================================================================
// Registration Types
// =============================================================================

// ServiceRegistration represents a service to be registered.
type ServiceRegistration struct {
	Service     AWSService
	ViewFactory ViewFactory
	Priority    int // Display order priority (higher = appears first)
}

// ViewRegistration represents a view to be registered.
type ViewRegistration struct {
	View        View
	ServiceName string
	Priority    int
}

// HookRegistration represents a hook to be registered.
type HookRegistration struct {
	Hook     Hook
	Priority int
}

// RegistryEvent represents a change in the registry.
type RegistryEvent struct {
	Type      RegistryEventType
	Name      string
	Timestamp time.Time
}

// RegistryEventType defines the type of registry event.
type RegistryEventType string

const (
	RegistryEventRegistered   RegistryEventType = "registered"
	RegistryEventUnregistered RegistryEventType = "unregistered"
	RegistryEventUpdated      RegistryEventType = "updated"
)
