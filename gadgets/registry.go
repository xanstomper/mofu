package gadgets

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Plugin Registry — versioned runtime plugin loading
// ---------------------------------------------------------------------------

// Version represents a semantic version.
type Version struct {
	Major, Minor, Patch int
}

// String returns the version as "major.minor.patch".
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare returns -1, 0, or 1 for less, equal, greater.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// Compatible reports whether other is compatible (same major version).
func (v Version) Compatible(other Version) bool {
	return v.Major == other.Major
}

// ParseVersion parses a "major.minor.patch" string.
func ParseVersion(s string) (Version, error) {
	var v Version
	_, err := fmt.Sscanf(s, "%d.%d.%d", &v.Major, &v.Minor, &v.Patch)
	if err != nil {
		return Version{}, fmt.Errorf("invalid version %q: %w", s, err)
	}
	return v, nil
}

// MustParseVersion parses a version string or panics.
func MustParseVersion(s string) Version {
	v, err := ParseVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

// ---------------------------------------------------------------------------
// Plugin Manifest
// ---------------------------------------------------------------------------

// PluginManifest describes a gadget plugin's metadata and requirements.
type PluginManifest struct {
	Name         string
	Version      Version
	Description  string
	Author       string
	Dependencies []PluginDependency
	Capabilities []string
	Tags         []string
}

// PluginDependency specifies a required plugin.
type PluginDependency struct {
	Name    string
	Version Version // minimum version
}

// ---------------------------------------------------------------------------
// Plugin Entry
// ---------------------------------------------------------------------------

// PluginEntry is a registered plugin in the registry.
type PluginEntry struct {
	Manifest PluginManifest
	Factory  func() Gadget
	Loaded   bool
	Instance *GadgetInstance
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

// Registry manages versioned gadget plugins with dependency resolution.
type Registry struct {
	mu       sync.Mutex
	plugins  map[string]*PluginEntry
	onChange []func(name string, action string)
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]*PluginEntry),
	}
}

// Register adds a plugin to the registry. Returns error if a plugin with
// the same name and version is already registered.
func (r *Registry) Register(manifest PluginManifest, factory func() Gadget) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := manifest.Name
	existing, exists := r.plugins[name]
	if exists {
		// Allow upgrading to a newer version
		if manifest.Version.Compare(existing.Manifest.Version) <= 0 {
			return fmt.Errorf("plugin %s v%s already registered (>= v%s)", name, existing.Manifest.Version, manifest.Version)
		}
	}

	r.plugins[name] = &PluginEntry{
		Manifest: manifest,
		Factory:  factory,
		Loaded:   false,
	}

	r.fireChange(name, "registered")
	return nil
}

// Unregister removes a plugin from the registry.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.plugins[name]
	if !ok {
		return fmt.Errorf("plugin %s not found", name)
	}

	if entry.Loaded {
		return fmt.Errorf("plugin %s is loaded, unload first", name)
	}

	delete(r.plugins, name)
	r.fireChange(name, "unregistered")
	return nil
}

// Get returns a plugin entry by name.
func (r *Registry) Get(name string) *PluginEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.plugins[name]
}

// List returns all registered plugin names.
func (r *Registry) List() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListManifests returns all registered manifests.
func (r *Registry) ListManifests() []PluginManifest {
	r.mu.Lock()
	defer r.mu.Unlock()
	manifests := make([]PluginManifest, 0, len(r.plugins))
	for _, entry := range r.plugins {
		manifests = append(manifests, entry.Manifest)
	}
	return manifests
}

// ---------------------------------------------------------------------------
// Dependency Resolution
// ---------------------------------------------------------------------------

// ResolveDependencies checks that all dependencies are satisfied.
// Returns a topologically sorted load order, or an error.
func (r *Registry) ResolveDependencies() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check all dependencies exist
	for name, entry := range r.plugins {
		for _, dep := range entry.Manifest.Dependencies {
			depEntry, ok := r.plugins[dep.Name]
			if !ok {
				return nil, fmt.Errorf("plugin %s requires %s v%s, but it is not registered", name, dep.Name, dep.Version)
			}
			if depEntry.Manifest.Version.Compare(dep.Version) < 0 {
				return nil, fmt.Errorf("plugin %s requires %s v%s, but only v%s is registered", name, dep.Name, dep.Version, depEntry.Manifest.Version)
			}
		}
	}

	// Topological sort (Kahn's algorithm)
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // dep → dependents

	for name := range r.plugins {
		inDegree[name] = 0
	}

	for name, entry := range r.plugins {
		for _, dep := range entry.Manifest.Dependencies {
			if _, ok := r.plugins[dep.Name]; ok {
				inDegree[name]++
				dependents[dep.Name] = append(dependents[dep.Name], name)
			}
		}
	}

	// Start with nodes that have no dependencies
	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	var order []string
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		order = append(order, name)

		for _, dep := range dependents[name] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
		sort.Strings(queue)
	}

	if len(order) != len(r.plugins) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return order, nil
}

// ---------------------------------------------------------------------------
// Load / Unload
// ---------------------------------------------------------------------------

// Load creates a gadget instance from a registered plugin.
func (r *Registry) Load(name string, ctx GadgetContext) (*GadgetInstance, error) {
	r.mu.Lock()
	entry, ok := r.plugins[name]
	if !ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	if entry.Loaded {
		r.mu.Unlock()
		return nil, fmt.Errorf("plugin %s already loaded", name)
	}
	factory := entry.Factory
	r.mu.Unlock()

	gadget := factory()
	inst := NewGadgetInstance(gadget)

	// Initialize
	if err := gadget.Init(ctx); err != nil {
		return nil, fmt.Errorf("plugin %s init failed: %w", name, err)
	}

	if ctx.Binder != nil {
		gadget.Bind(ctx.Binder)
	}

	r.mu.Lock()
	entry.Loaded = true
	entry.Instance = inst
	r.mu.Unlock()

	r.fireChange(name, "loaded")
	return inst, nil
}

// Unload disposes a loaded plugin.
func (r *Registry) Unload(name string) error {
	r.mu.Lock()
	entry, ok := r.plugins[name]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("plugin %s not found", name)
	}
	if !entry.Loaded {
		r.mu.Unlock()
		return fmt.Errorf("plugin %s not loaded", name)
	}
	instance := entry.Instance
	r.mu.Unlock()

	if instance != nil {
		instance.Gadget().Dispose()
		instance.transitionTo(GadgetUnmounted)
	}

	r.mu.Lock()
	entry.Loaded = false
	entry.Instance = nil
	r.mu.Unlock()

	r.fireChange(name, "unloaded")
	return nil
}

// LoadAll loads all plugins in dependency order.
func (r *Registry) LoadAll(ctx GadgetContext) ([]*GadgetInstance, []error) {
	order, err := r.ResolveDependencies()
	if err != nil {
		return nil, []error{err}
	}

	var instances []*GadgetInstance
	var errs []error

	for _, name := range order {
		inst, err := r.Load(name, ctx)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		instances = append(instances, inst)
	}

	return instances, errs
}

// UnloadAll unloads all loaded plugins in reverse dependency order.
func (r *Registry) UnloadAll() []error {
	order, err := r.ResolveDependencies()
	if err != nil {
		return []error{err}
	}

	var errs []error
	// Reverse order
	for i := len(order) - 1; i >= 0; i-- {
		if err := r.Unload(order[i]); err != nil {
			// Skip "not loaded" errors
			if !strings.Contains(err.Error(), "not loaded") {
				errs = append(errs, err)
			}
		}
	}

	return errs
}

// ---------------------------------------------------------------------------
// Conflict Detection
// ---------------------------------------------------------------------------

// Conflict describes a plugin conflict.
type Conflict struct {
	Plugin1, Plugin2 string
	Reason           string
}

// DetectConflicts finds conflicts between registered plugins.
func (r *Registry) DetectConflicts() []Conflict {
	r.mu.Lock()
	defer r.mu.Unlock()

	var conflicts []Conflict

	// Check for capability conflicts
	capabilityMap := make(map[string][]string) // capability → plugins
	for name, entry := range r.plugins {
		for _, cap := range entry.Manifest.Capabilities {
			capabilityMap[cap] = append(capabilityMap[cap], name)
		}
	}

	for cap, plugins := range capabilityMap {
		if len(plugins) > 1 {
			sort.Strings(plugins)
			conflicts = append(conflicts, Conflict{
				Plugin1: plugins[0],
				Plugin2: plugins[1],
				Reason:  fmt.Sprintf("both provide capability %q", cap),
			})
		}
	}

	return conflicts
}

// ---------------------------------------------------------------------------
// Search / Filter
// ---------------------------------------------------------------------------

// SearchOptions filters plugins by various criteria.
type SearchOptions struct {
	Tag         string
	Capability  string
	Author      string
	NamePattern string
}

// Search returns plugins matching the search criteria.
func (r *Registry) Search(opts SearchOptions) []PluginManifest {
	r.mu.Lock()
	defer r.mu.Unlock()

	var results []PluginManifest
	for _, entry := range r.plugins {
		m := entry.Manifest

		if opts.Tag != "" && !containsTag(m.Tags, opts.Tag) {
			continue
		}
		if opts.Capability != "" && !containsTag(m.Capabilities, opts.Capability) {
			continue
		}
		if opts.Author != "" && m.Author != opts.Author {
			continue
		}
		if opts.NamePattern != "" && !strings.Contains(strings.ToLower(m.Name), strings.ToLower(opts.NamePattern)) {
			continue
		}

		results = append(results, m)
	}

	return results
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// OnChange
// ---------------------------------------------------------------------------

// OnChange registers a callback for registry changes.
func (r *Registry) OnChange(fn func(name string, action string)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onChange = append(r.onChange, fn)
}

func (r *Registry) fireChange(name, action string) {
	for _, fn := range r.onChange {
		fn(name, action)
	}
}
