package core

import (
	"errors"
	"fmt"
)

// =============================================================================
// Sentinel Errors
// =============================================================================

var (
	// Service errors
	ErrServiceNotFound      = errors.New("service not found")
	ErrServiceAlreadyExists = errors.New("service already registered")
	ErrServiceNotReady      = errors.New("service not ready")
	ErrServiceClosed        = errors.New("service is closed")

	// View errors
	ErrViewNotFound      = errors.New("view not found")
	ErrViewAlreadyExists = errors.New("view already registered")
	ErrShortcutConflict  = errors.New("shortcut already in use")

	// Resource errors
	ErrResourceNotFound = errors.New("resource not found")
	ErrResourceExists   = errors.New("resource already exists")
	ErrInvalidResource  = errors.New("invalid resource")

	// Action errors
	ErrActionNotFound       = errors.New("action not found")
	ErrActionNotSupported   = errors.New("action not supported")
	ErrActionFailed         = errors.New("action failed")
	ErrActionCancelled      = errors.New("action cancelled")
	ErrInvalidActionParams  = errors.New("invalid action parameters")
	ErrConfirmationRequired = errors.New("confirmation required for dangerous action")

	// Plugin errors
	ErrPluginNotFound          = errors.New("plugin not found")
	ErrPluginAlreadyLoaded     = errors.New("plugin already loaded")
	ErrPluginLoadFailed        = errors.New("plugin load failed")
	ErrPluginInitFailed        = errors.New("plugin initialization failed")
	ErrPluginDependencyMissing = errors.New("plugin dependency missing")
	ErrInvalidPluginManifest   = errors.New("invalid plugin manifest")

	// Configuration errors
	ErrConfigNotFound   = errors.New("configuration not found")
	ErrConfigInvalid    = errors.New("invalid configuration")
	ErrConfigReadFailed = errors.New("failed to read configuration")

	// Container errors
	ErrDependencyNotFound = errors.New("dependency not found")
	ErrCircularDependency = errors.New("circular dependency detected")
	ErrInvalidFactory     = errors.New("invalid factory function")
	ErrResolutionFailed   = errors.New("dependency resolution failed")

	// AWS errors
	ErrAWSConfigFailed = errors.New("failed to load AWS configuration")
	ErrAWSCredentials  = errors.New("AWS credentials error")
	ErrAWSPermission   = errors.New("AWS permission denied")
	ErrAWSRateLimit    = errors.New("AWS rate limit exceeded")
	ErrAWSServiceError = errors.New("AWS service error")

	// General errors
	ErrNotImplemented = errors.New("not implemented")
	ErrTimeout        = errors.New("operation timed out")
	ErrCancelled      = errors.New("operation cancelled")
)

// =============================================================================
// Error Types
// =============================================================================

// ServiceError represents a service-related error with context.
type ServiceError struct {
	Service string
	Op      string
	Err     error
}

func (e *ServiceError) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("service %s: %s: %v", e.Service, e.Op, e.Err)
	}
	return fmt.Sprintf("service %s: %v", e.Service, e.Err)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

// NewServiceError creates a new ServiceError.
func NewServiceError(service, op string, err error) *ServiceError {
	return &ServiceError{
		Service: service,
		Op:      op,
		Err:     err,
	}
}

// ResourceError represents a resource-related error with context.
type ResourceError struct {
	ResourceType string
	ResourceID   string
	Op           string
	Err          error
}

func (e *ResourceError) Error() string {
	if e.ResourceID != "" {
		return fmt.Sprintf("resource %s/%s: %s: %v", e.ResourceType, e.ResourceID, e.Op, e.Err)
	}
	return fmt.Sprintf("resource %s: %s: %v", e.ResourceType, e.Op, e.Err)
}

func (e *ResourceError) Unwrap() error {
	return e.Err
}

// NewResourceError creates a new ResourceError.
func NewResourceError(resourceType, resourceID, op string, err error) *ResourceError {
	return &ResourceError{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Op:           op,
		Err:          err,
	}
}

// ActionError represents an action-related error with context.
type ActionError struct {
	Action     string
	ResourceID string
	Err        error
}

func (e *ActionError) Error() string {
	if e.ResourceID != "" {
		return fmt.Sprintf("action %s on %s: %v", e.Action, e.ResourceID, e.Err)
	}
	return fmt.Sprintf("action %s: %v", e.Action, e.Err)
}

func (e *ActionError) Unwrap() error {
	return e.Err
}

// NewActionError creates a new ActionError.
func NewActionError(action, resourceID string, err error) *ActionError {
	return &ActionError{
		Action:     action,
		ResourceID: resourceID,
		Err:        err,
	}
}

// PluginError represents a plugin-related error with context.
type PluginError struct {
	Plugin string
	Op     string
	Err    error
}

func (e *PluginError) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("plugin %s: %s: %v", e.Plugin, e.Op, e.Err)
	}
	return fmt.Sprintf("plugin %s: %v", e.Plugin, e.Err)
}

func (e *PluginError) Unwrap() error {
	return e.Err
}

// NewPluginError creates a new PluginError.
func NewPluginError(plugin, op string, err error) *PluginError {
	return &PluginError{
		Plugin: plugin,
		Op:     op,
		Err:    err,
	}
}

// ValidationError represents a validation failure.
type ValidationError struct {
	Field   string
	Value   any
	Message string
}

func (e *ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("validation failed for %s=%v: %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field string, value any, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// =============================================================================
// Error Helpers
// =============================================================================

// IsNotFound checks if an error is a "not found" type error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrServiceNotFound) ||
		errors.Is(err, ErrViewNotFound) ||
		errors.Is(err, ErrResourceNotFound) ||
		errors.Is(err, ErrActionNotFound) ||
		errors.Is(err, ErrPluginNotFound) ||
		errors.Is(err, ErrConfigNotFound) ||
		errors.Is(err, ErrDependencyNotFound)
}

// IsAlreadyExists checks if an error is an "already exists" type error.
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrServiceAlreadyExists) ||
		errors.Is(err, ErrViewAlreadyExists) ||
		errors.Is(err, ErrResourceExists) ||
		errors.Is(err, ErrPluginAlreadyLoaded)
}

// IsPermission checks if an error is a permission-related error.
func IsPermission(err error) bool {
	return errors.Is(err, ErrAWSPermission) ||
		errors.Is(err, ErrAWSCredentials)
}

// IsTimeout checks if an error is a timeout error.
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}

// IsCancelled checks if an error is a cancellation error.
func IsCancelled(err error) bool {
	return errors.Is(err, ErrCancelled) ||
		errors.Is(err, ErrActionCancelled)
}

// Wrap wraps an error with additional context.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with a formatted message.
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
