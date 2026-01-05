// Package hooks provides an event dispatching system for a9s.
// It allows registering hooks that respond to various system events
// such as resource listing, action execution, and errors.
package hooks

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/keanuharrell/a9s/internal/core"
)

// =============================================================================
// Dispatcher Implementation
// =============================================================================

// Dispatcher manages event dispatch to registered hooks.
type Dispatcher struct {
	mu          sync.RWMutex
	hooks       map[string]core.Hook
	byEventType map[core.EventType][]core.Hook
	middlewares []core.HookMiddleware
	async       bool
	errorChan   chan error
}

// Option configures the dispatcher.
type Option func(*Dispatcher)

// WithAsync enables asynchronous event dispatch.
func WithAsync(errChan chan error) Option {
	return func(d *Dispatcher) {
		d.async = true
		d.errorChan = errChan
	}
}

// NewDispatcher creates a new event dispatcher.
func NewDispatcher(opts ...Option) *Dispatcher {
	d := &Dispatcher{
		hooks:       make(map[string]core.Hook),
		byEventType: make(map[core.EventType][]core.Hook),
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// =============================================================================
// Hook Management
// =============================================================================

// Register adds a hook to the dispatcher.
func (d *Dispatcher) Register(hook core.Hook) {
	d.mu.Lock()
	defer d.mu.Unlock()

	name := hook.Name()

	// Remove existing hook with same name
	if existing, ok := d.hooks[name]; ok {
		d.removeFromEventTypes(existing)
	}

	// Add hook
	d.hooks[name] = hook

	// Index by event types
	for _, eventType := range hook.EventTypes() {
		d.byEventType[eventType] = append(d.byEventType[eventType], hook)
		// Sort by priority (descending)
		sort.Slice(d.byEventType[eventType], func(i, j int) bool {
			return d.byEventType[eventType][i].Priority() > d.byEventType[eventType][j].Priority()
		})
	}
}

// Unregister removes a hook by name.
func (d *Dispatcher) Unregister(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	hook, ok := d.hooks[name]
	if !ok {
		return
	}

	d.removeFromEventTypes(hook)
	delete(d.hooks, name)
}

// removeFromEventTypes removes a hook from all event type indexes.
func (d *Dispatcher) removeFromEventTypes(hook core.Hook) {
	for _, eventType := range hook.EventTypes() {
		hooks := d.byEventType[eventType]
		for i, h := range hooks {
			if h.Name() == hook.Name() {
				d.byEventType[eventType] = append(hooks[:i], hooks[i+1:]...)
				break
			}
		}
	}
}

// Use adds middleware to the dispatch chain.
func (d *Dispatcher) Use(middleware core.HookMiddleware) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.middlewares = append(d.middlewares, middleware)
}

// =============================================================================
// Event Dispatch
// =============================================================================

// Dispatch sends an event to all registered hooks for that event type.
func (d *Dispatcher) Dispatch(ctx context.Context, event core.Event) error {
	d.mu.RLock()
	hooks := d.byEventType[event.Type()]
	middlewares := d.middlewares
	d.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	if d.async {
		go func() {
			if err := d.dispatchToHooks(ctx, event, hooks, middlewares); err != nil {
				if d.errorChan != nil {
					select {
					case d.errorChan <- err:
					default:
						// Channel full, drop error
					}
				}
			}
		}()
		return nil
	}

	return d.dispatchToHooks(ctx, event, hooks, middlewares)
}

// dispatchToHooks dispatches an event to a list of hooks.
func (d *Dispatcher) dispatchToHooks(ctx context.Context, event core.Event, hooks []core.Hook, middlewares []core.HookMiddleware) error {
	var errs []error

	for _, hook := range hooks {
		// Build handler chain with middlewares
		handler := hook.Handle

		// Apply middlewares in reverse order (last added runs first)
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i].Wrap(handler)
		}

		// Execute handler
		if err := handler(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf("hook %s: %w", hook.Name(), err))
		}
	}

	if len(errs) > 0 {
		return &DispatchError{Errors: errs}
	}

	return nil
}

// DispatchAll sends an event to all registered hooks regardless of event type.
func (d *Dispatcher) DispatchAll(ctx context.Context, event core.Event) error {
	d.mu.RLock()
	allHooks := make([]core.Hook, 0, len(d.hooks))
	for _, hook := range d.hooks {
		allHooks = append(allHooks, hook)
	}
	middlewares := d.middlewares
	d.mu.RUnlock()

	// Sort by priority
	sort.Slice(allHooks, func(i, j int) bool {
		return allHooks[i].Priority() > allHooks[j].Priority()
	})

	return d.dispatchToHooks(ctx, event, allHooks, middlewares)
}

// =============================================================================
// Queries
// =============================================================================

// Hooks returns all registered hooks.
func (d *Dispatcher) Hooks() []core.Hook {
	d.mu.RLock()
	defer d.mu.RUnlock()

	hooks := make([]core.Hook, 0, len(d.hooks))
	for _, hook := range d.hooks {
		hooks = append(hooks, hook)
	}

	return hooks
}

// HooksForEvent returns hooks registered for a specific event type.
func (d *Dispatcher) HooksForEvent(eventType core.EventType) []core.Hook {
	d.mu.RLock()
	defer d.mu.RUnlock()

	hooks := d.byEventType[eventType]
	result := make([]core.Hook, len(hooks))
	copy(result, hooks)

	return result
}

// HasHook checks if a hook is registered.
func (d *Dispatcher) HasHook(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.hooks[name]
	return ok
}

// =============================================================================
// Error Types
// =============================================================================

// DispatchError represents errors from multiple hooks.
type DispatchError struct {
	Errors []error
}

func (e *DispatchError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("dispatch failed with %d errors: %v", len(e.Errors), e.Errors[0])
}

// Unwrap returns the first error for errors.Is/As compatibility.
func (e *DispatchError) Unwrap() error {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return nil
}

// =============================================================================
// Base Hook Implementation
// =============================================================================

// BaseHook provides a base implementation for hooks.
type BaseHook struct {
	name       string
	eventTypes []core.EventType
	priority   int
	handler    core.HookHandler
}

// NewBaseHook creates a new base hook.
func NewBaseHook(name string, eventTypes []core.EventType, priority int, handler core.HookHandler) *BaseHook {
	return &BaseHook{
		name:       name,
		eventTypes: eventTypes,
		priority:   priority,
		handler:    handler,
	}
}

// Name returns the hook name.
func (h *BaseHook) Name() string {
	return h.name
}

// EventTypes returns the event types this hook handles.
func (h *BaseHook) EventTypes() []core.EventType {
	return h.eventTypes
}

// Priority returns the execution priority.
func (h *BaseHook) Priority() int {
	return h.priority
}

// Handle processes an event.
func (h *BaseHook) Handle(ctx context.Context, event core.Event) error {
	if h.handler != nil {
		return h.handler(ctx, event)
	}
	return nil
}

// =============================================================================
// Common Middlewares
// =============================================================================

// RecoveryMiddleware recovers from panics in hook handlers.
type RecoveryMiddleware struct {
	OnPanic func(hook string, r any)
}

// Wrap implements HookMiddleware.
func (m *RecoveryMiddleware) Wrap(next core.HookHandler) core.HookHandler {
	return func(ctx context.Context, event core.Event) (err error) {
		defer func() {
			if r := recover(); r != nil {
				if m.OnPanic != nil {
					m.OnPanic(event.Source(), r)
				}
				err = fmt.Errorf("hook panic: %v", r)
			}
		}()
		return next(ctx, event)
	}
}

// TimeoutMiddleware adds timeout to hook execution.
type TimeoutMiddleware struct {
	Timeout context.Context
}

// MetricsMiddleware collects metrics about hook execution.
type MetricsMiddleware struct {
	OnExecute func(hookName string, eventType core.EventType, durationMs int64, err error)
}

// Wrap implements HookMiddleware.
func (m *MetricsMiddleware) Wrap(next core.HookHandler) core.HookHandler {
	return func(ctx context.Context, event core.Event) error {
		// Note: In a real implementation, we'd track time here
		err := next(ctx, event)
		if m.OnExecute != nil {
			m.OnExecute(event.Source(), event.Type(), 0, err)
		}
		return err
	}
}

// =============================================================================
// Interface Assertions
// =============================================================================

var (
	_ core.EventDispatcher = (*Dispatcher)(nil)
	_ core.Hook            = (*BaseHook)(nil)
	_ core.HookMiddleware  = (*RecoveryMiddleware)(nil)
	_ core.HookMiddleware  = (*MetricsMiddleware)(nil)
)
