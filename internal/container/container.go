// Package container provides a dependency injection container for the a9s application.
// It supports singleton registration, factory functions, and lifecycle management.
package container

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/keanuharrell/a9s/internal/core"
)

// Container implements the core.Container interface.
type Container struct {
	mu         sync.RWMutex
	singletons map[string]any
	factories  map[string]any
	resolved   map[string]any
	order      []string // Track registration order for shutdown
}

// New creates a new dependency injection container.
func New() *Container {
	return &Container{
		singletons: make(map[string]any),
		factories:  make(map[string]any),
		resolved:   make(map[string]any),
		order:      make([]string, 0),
	}
}

// RegisterSingleton registers a singleton instance by name.
func (c *Container) RegisterSingleton(name string, instance any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.singletons[name]; !exists {
		c.order = append(c.order, name)
	}
	c.singletons[name] = instance
}

// Register registers a factory function by name.
// The factory function should return the desired type and optionally an error.
// Example: func() (*MyService, error) or func(dep *OtherService) *MyService
func (c *Container) Register(name string, factory any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.validateFactory(factory); err != nil {
		panic(fmt.Sprintf("invalid factory for %s: %v", name, err))
	}

	c.factories[name] = factory
}

// Resolve returns an instance by name.
func (c *Container) Resolve(name string) (any, error) {
	c.mu.RLock()

	// Check singletons first
	if instance, exists := c.singletons[name]; exists {
		c.mu.RUnlock()
		return instance, nil
	}

	// Check resolved cache
	if instance, exists := c.resolved[name]; exists {
		c.mu.RUnlock()
		return instance, nil
	}

	// Check factories
	factory, exists := c.factories[name]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", core.ErrDependencyNotFound, name)
	}

	// Invoke factory
	instance, err := c.invokeFactory(factory)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", core.ErrResolutionFailed, name, err)
	}

	// Cache resolved instance
	c.mu.Lock()
	c.resolved[name] = instance
	c.order = append(c.order, name)
	c.mu.Unlock()

	return instance, nil
}

// MustResolve returns an instance by name or panics.
func (c *Container) MustResolve(name string) any {
	instance, err := c.Resolve(name)
	if err != nil {
		panic(err)
	}
	return instance
}

// ResolveAll returns all instances that match a type by name prefix.
func (c *Container) ResolveAll(prefix string) []any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []any

	// Check singletons
	for name, instance := range c.singletons {
		if matchesPrefix(name, prefix) {
			results = append(results, instance)
		}
	}

	// Check resolved
	for name, instance := range c.resolved {
		if matchesPrefix(name, prefix) {
			results = append(results, instance)
		}
	}

	return results
}

// Has checks if a registration exists.
func (c *Container) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if _, exists := c.singletons[name]; exists {
		return true
	}
	if _, exists := c.factories[name]; exists {
		return true
	}
	if _, exists := c.resolved[name]; exists {
		return true
	}
	return false
}

// Start initializes all registered services that implement Initializable.
func (c *Container) Start(ctx context.Context) error {
	c.mu.RLock()
	singletons := make(map[string]any)
	for k, v := range c.singletons {
		singletons[k] = v
	}
	c.mu.RUnlock()

	for name, instance := range singletons {
		if initializable, ok := instance.(interface {
			Initialize(context.Context) error
		}); ok {
			if err := initializable.Initialize(ctx); err != nil {
				return fmt.Errorf("failed to initialize %s: %w", name, err)
			}
		}
	}

	return nil
}

// Stop shuts down all registered services in reverse order.
func (c *Container) Stop(ctx context.Context) error {
	c.mu.RLock()
	order := make([]string, len(c.order))
	copy(order, c.order)
	c.mu.RUnlock()

	var errs []error

	// Shutdown in reverse order
	for i := len(order) - 1; i >= 0; i-- {
		name := order[i]
		var instance any

		c.mu.RLock()
		if inst, exists := c.singletons[name]; exists {
			instance = inst
		} else if inst, exists := c.resolved[name]; exists {
			instance = inst
		}
		c.mu.RUnlock()

		if instance == nil {
			continue
		}

		if closer, ok := instance.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close %s: %w", name, err))
			}
		}

		if stopper, ok := instance.(interface{ Stop(context.Context) error }); ok {
			if err := stopper.Stop(ctx); err != nil {
				errs = append(errs, fmt.Errorf("failed to stop %s: %w", name, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	return nil
}

// validateFactory checks if a factory function is valid.
func (c *Container) validateFactory(factory any) error {
	v := reflect.ValueOf(factory)
	if v.Kind() != reflect.Func {
		return fmt.Errorf("factory must be a function, got %T", factory)
	}

	t := v.Type()
	if t.NumOut() == 0 {
		return fmt.Errorf("factory must return at least one value")
	}
	if t.NumOut() > 2 {
		return fmt.Errorf("factory must return at most two values")
	}
	if t.NumOut() == 2 && !t.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return fmt.Errorf("second return value must be error")
	}

	return nil
}

// invokeFactory calls a factory function and returns its result.
func (c *Container) invokeFactory(factory any) (any, error) {
	v := reflect.ValueOf(factory)
	t := v.Type()

	// Build arguments by resolving dependencies
	args := make([]reflect.Value, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		paramType := t.In(i)
		dep, err := c.resolveDependency(paramType)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve parameter %d (%s): %w", i, paramType.String(), err)
		}
		args[i] = reflect.ValueOf(dep)
	}

	// Call the factory
	results := v.Call(args)

	if len(results) == 0 {
		return nil, fmt.Errorf("factory returned no values")
	}

	// Check for error return
	if len(results) == 2 && !results[1].IsNil() {
		if err, ok := results[1].Interface().(error); ok {
			return nil, err
		}
	}

	return results[0].Interface(), nil
}

// resolveDependency attempts to resolve a dependency by type.
func (c *Container) resolveDependency(paramType reflect.Type) (any, error) {
	// Try to find a registered instance that matches the type
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check singletons
	for _, instance := range c.singletons {
		instanceType := reflect.TypeOf(instance)
		if instanceType.AssignableTo(paramType) {
			return instance, nil
		}
		// Check for pointer/interface compatibility
		if paramType.Kind() == reflect.Interface && instanceType.Implements(paramType) {
			return instance, nil
		}
	}

	// Check resolved
	for _, instance := range c.resolved {
		instanceType := reflect.TypeOf(instance)
		if instanceType.AssignableTo(paramType) {
			return instance, nil
		}
		if paramType.Kind() == reflect.Interface && instanceType.Implements(paramType) {
			return instance, nil
		}
	}

	return nil, fmt.Errorf("%w: type %s", core.ErrDependencyNotFound, paramType.String())
}

// matchesPrefix checks if a name matches a prefix pattern.
func matchesPrefix(name, prefix string) bool {
	if len(prefix) > len(name) {
		return false
	}
	return name[:len(prefix)] == prefix
}

// =============================================================================
// Helper Types
// =============================================================================

// Initializable is an interface for types that need initialization.
type Initializable interface {
	Initialize(ctx context.Context) error
}

// Closeable is an interface for types that need cleanup.
type Closeable interface {
	Close() error
}

// Stoppable is an interface for types that need graceful shutdown.
type Stoppable interface {
	Stop(ctx context.Context) error
}

// =============================================================================
// Builder Pattern
// =============================================================================

// Builder provides a fluent API for building a container.
type Builder struct {
	container *Container
}

// NewBuilder creates a new container builder.
func NewBuilder() *Builder {
	return &Builder{
		container: New(),
	}
}

// Singleton adds a singleton to the container.
func (b *Builder) Singleton(name string, instance any) *Builder {
	b.container.RegisterSingleton(name, instance)
	return b
}

// Factory adds a factory to the container.
func (b *Builder) Factory(name string, factory any) *Builder {
	b.container.Register(name, factory)
	return b
}

// Build returns the configured container.
func (b *Builder) Build() *Container {
	return b.container
}
