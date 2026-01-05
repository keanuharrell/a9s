// Package registry provides a central registry for managing AWS services and TUI views.
// It enables dynamic registration, discovery, and hot-reloading of components.
package registry

import (
	"sort"
	"sync"
	"time"

	"github.com/keanuharrell/a9s/internal/core"
)

// =============================================================================
// Registry Implementation
// =============================================================================

// Registry manages service and view registrations.
type Registry struct {
	mu        sync.RWMutex
	services  map[string]serviceEntry
	views     map[string]viewEntry
	shortcuts map[string]string // shortcut -> view name
	observers []func(core.RegistryEvent)
}

type serviceEntry struct {
	service  core.AWSService
	priority int
}

type viewEntry struct {
	view     core.View
	priority int
}

// New creates a new registry.
func New() *Registry {
	return &Registry{
		services:  make(map[string]serviceEntry),
		views:     make(map[string]viewEntry),
		shortcuts: make(map[string]string),
	}
}

// =============================================================================
// Service Registry Methods
// =============================================================================

// RegisterService registers an AWS service.
func (r *Registry) RegisterService(service core.AWSService) error {
	return r.RegisterServiceWithPriority(service, 0)
}

// RegisterServiceWithPriority registers a service with a display priority.
func (r *Registry) RegisterServiceWithPriority(service core.AWSService, priority int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := service.Name()
	if _, exists := r.services[name]; exists {
		return core.ErrServiceAlreadyExists
	}

	r.services[name] = serviceEntry{
		service:  service,
		priority: priority,
	}

	r.notify(core.RegistryEvent{
		Type:      core.RegistryEventRegistered,
		Name:      name,
		Timestamp: time.Now(),
	})

	return nil
}

// UnregisterService removes a service from the registry.
func (r *Registry) UnregisterService(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[name]; !exists {
		return core.ErrServiceNotFound
	}

	// Close service if it implements Close
	if entry, ok := r.services[name]; ok {
		_ = entry.service.Close()
	}

	delete(r.services, name)

	// Also remove associated view
	if _, hasView := r.views[name]; hasView {
		if view, ok := r.views[name]; ok {
			delete(r.shortcuts, view.view.Shortcut())
		}
		delete(r.views, name)
	}

	r.notify(core.RegistryEvent{
		Type:      core.RegistryEventUnregistered,
		Name:      name,
		Timestamp: time.Now(),
	})

	return nil
}

// GetService returns a service by name.
func (r *Registry) GetService(name string) (core.AWSService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.services[name]
	if !exists {
		return nil, core.ErrServiceNotFound
	}

	return entry.service, nil
}

// ListServices returns all registered services.
func (r *Registry) ListServices() []core.AWSService {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]core.AWSService, 0, len(r.services))
	for _, entry := range r.services {
		services = append(services, entry.service)
	}

	return services
}

// ListServicesOrdered returns services ordered by priority (highest first).
func (r *Registry) ListServicesOrdered() []core.AWSService {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]serviceEntry, 0, len(r.services))
	for _, entry := range r.services {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].priority > entries[j].priority
	})

	services := make([]core.AWSService, len(entries))
	for i, entry := range entries {
		services[i] = entry.service
	}

	return services
}

// HasService checks if a service is registered.
func (r *Registry) HasService(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.services[name]
	return exists
}

// =============================================================================
// View Registry Methods
// =============================================================================

// RegisterView registers a TUI view.
func (r *Registry) RegisterView(view core.View) error {
	return r.RegisterViewWithPriority(view, 0)
}

// RegisterViewWithPriority registers a view with a display priority.
func (r *Registry) RegisterViewWithPriority(view core.View, priority int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := view.Name()
	shortcut := view.Shortcut()

	if _, exists := r.views[name]; exists {
		return core.ErrViewAlreadyExists
	}

	// Check for shortcut conflicts
	if existing, exists := r.shortcuts[shortcut]; exists {
		return core.Wrapf(core.ErrShortcutConflict, "shortcut '%s' already used by '%s'", shortcut, existing)
	}

	r.views[name] = viewEntry{
		view:     view,
		priority: priority,
	}
	r.shortcuts[shortcut] = name

	r.notify(core.RegistryEvent{
		Type:      core.RegistryEventRegistered,
		Name:      name,
		Timestamp: time.Now(),
	})

	return nil
}

// UnregisterView removes a view from the registry.
func (r *Registry) UnregisterView(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.views[name]
	if !exists {
		return core.ErrViewNotFound
	}

	delete(r.shortcuts, entry.view.Shortcut())
	delete(r.views, name)

	r.notify(core.RegistryEvent{
		Type:      core.RegistryEventUnregistered,
		Name:      name,
		Timestamp: time.Now(),
	})

	return nil
}

// GetView returns a view by name.
func (r *Registry) GetView(name string) (core.View, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.views[name]
	if !exists {
		return nil, core.ErrViewNotFound
	}

	return entry.view, nil
}

// GetViewByShortcut returns a view by its keyboard shortcut.
func (r *Registry) GetViewByShortcut(shortcut string) (core.View, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	name, exists := r.shortcuts[shortcut]
	if !exists {
		return nil, core.Wrapf(core.ErrViewNotFound, "no view with shortcut '%s'", shortcut)
	}

	entry, exists := r.views[name]
	if !exists {
		return nil, core.ErrViewNotFound
	}

	return entry.view, nil
}

// ListViews returns all registered views.
func (r *Registry) ListViews() []core.View {
	r.mu.RLock()
	defer r.mu.RUnlock()

	views := make([]core.View, 0, len(r.views))
	for _, entry := range r.views {
		views = append(views, entry.view)
	}

	return views
}

// ListViewsOrdered returns views ordered by priority (highest first).
func (r *Registry) ListViewsOrdered() []core.View {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]viewEntry, 0, len(r.views))
	for _, entry := range r.views {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].priority > entries[j].priority
	})

	views := make([]core.View, len(entries))
	for i, entry := range entries {
		views[i] = entry.view
	}

	return views
}

// HasView checks if a view is registered.
func (r *Registry) HasView(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.views[name]
	return exists
}

// GetShortcuts returns a map of shortcuts to view names.
func (r *Registry) GetShortcuts() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	shortcuts := make(map[string]string)
	for k, v := range r.shortcuts {
		shortcuts[k] = v
	}

	return shortcuts
}

// =============================================================================
// Combined Registration
// =============================================================================

// RegisterServiceAndView registers both a service and its view.
func (r *Registry) RegisterServiceAndView(reg core.ServiceRegistration) error {
	// Register service
	if err := r.RegisterServiceWithPriority(reg.Service, reg.Priority); err != nil {
		return err
	}

	// Create and register view if factory provided
	if reg.ViewFactory != nil {
		view, err := reg.ViewFactory.Create(reg.Service)
		if err != nil {
			// Rollback service registration
			_ = r.UnregisterService(reg.Service.Name())
			return core.Wrapf(err, "failed to create view for %s", reg.Service.Name())
		}

		if err := r.RegisterViewWithPriority(view, reg.Priority); err != nil {
			// Rollback service registration
			_ = r.UnregisterService(reg.Service.Name())
			return err
		}
	}

	return nil
}

// =============================================================================
// Observer Pattern
// =============================================================================

// Watch registers an observer for registry events.
func (r *Registry) Watch(callback func(core.RegistryEvent)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observers = append(r.observers, callback)
}

// notify sends an event to all observers.
func (r *Registry) notify(event core.RegistryEvent) {
	for _, observer := range r.observers {
		go observer(event)
	}
}

// =============================================================================
// Statistics
// =============================================================================

// Stats holds registry statistics.
type Stats struct {
	ServiceCount int
	ViewCount    int
	Services     []string
	Views        []string
}

// Stats returns registry statistics.
func (r *Registry) Stats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := Stats{
		ServiceCount: len(r.services),
		ViewCount:    len(r.views),
		Services:     make([]string, 0, len(r.services)),
		Views:        make([]string, 0, len(r.views)),
	}

	for name := range r.services {
		stats.Services = append(stats.Services, name)
	}

	for name := range r.views {
		stats.Views = append(stats.Views, name)
	}

	sort.Strings(stats.Services)
	sort.Strings(stats.Views)

	return stats
}

// =============================================================================
// Service Registry Adapter
// =============================================================================

// ServiceRegistryAdapter adapts Registry to core.ServiceRegistry.
type ServiceRegistryAdapter struct {
	*Registry
}

// AsServiceRegistry returns an adapter implementing core.ServiceRegistry.
func (r *Registry) AsServiceRegistry() core.ServiceRegistry {
	return &ServiceRegistryAdapter{r}
}

func (a *ServiceRegistryAdapter) Register(service core.AWSService) error {
	return a.RegisterService(service)
}

func (a *ServiceRegistryAdapter) Unregister(name string) error {
	return a.UnregisterService(name)
}

func (a *ServiceRegistryAdapter) Get(name string) (core.AWSService, error) {
	return a.GetService(name)
}

func (a *ServiceRegistryAdapter) List() []core.AWSService {
	return a.ListServices()
}

func (a *ServiceRegistryAdapter) Has(name string) bool {
	return a.HasService(name)
}

func (a *ServiceRegistryAdapter) Watch(callback func(core.RegistryEvent)) {
	a.Registry.Watch(callback)
}

// =============================================================================
// View Registry Adapter
// =============================================================================

// ViewRegistryAdapter adapts Registry to core.ViewRegistry.
type ViewRegistryAdapter struct {
	*Registry
}

// AsViewRegistry returns an adapter implementing core.ViewRegistry.
func (r *Registry) AsViewRegistry() core.ViewRegistry {
	return &ViewRegistryAdapter{r}
}

func (a *ViewRegistryAdapter) Register(view core.View) error {
	return a.RegisterView(view)
}

func (a *ViewRegistryAdapter) Unregister(name string) error {
	return a.UnregisterView(name)
}

func (a *ViewRegistryAdapter) Get(name string) (core.View, error) {
	return a.GetView(name)
}

func (a *ViewRegistryAdapter) GetByShortcut(shortcut string) (core.View, error) {
	return a.GetViewByShortcut(shortcut)
}

func (a *ViewRegistryAdapter) List() []core.View {
	return a.ListViews()
}

func (a *ViewRegistryAdapter) ListOrdered() []core.View {
	return a.ListViewsOrdered()
}

func (a *ViewRegistryAdapter) Has(name string) bool {
	return a.HasView(name)
}

func (a *ViewRegistryAdapter) Watch(callback func(core.RegistryEvent)) {
	a.Registry.Watch(callback)
}

// Ensure adapters implement the interfaces
var (
	_ core.ServiceRegistry = (*ServiceRegistryAdapter)(nil)
	_ core.ViewRegistry    = (*ViewRegistryAdapter)(nil)
)
