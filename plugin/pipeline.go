package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// Plugin Pipeline (Anthology Ch.19)
// ---------------------------------------------------------------------------

// PluginError wraps plugin failures.
type PluginError struct {
	Plugin string
	Err    error
}

func (e *PluginError) Error() string { return fmt.Sprintf("plugin %s: %v", e.Plugin, e.Err) }
func (e *PluginError) Unwrap() error { return e.Err }

// PluginActionKind identifies plugin action types.
type PluginActionKind string

const (
	ActionRender PluginActionKind = "render"
	ActionEvent  PluginActionKind = "event"
	ActionState  PluginActionKind = "state"
)

// PluginAction carries a plugin mutation.
type PluginAction struct {
	Kind    PluginActionKind
	Payload any
}

// Handler processes a pipeline request.
type Handler func(any) any

// Middleware wraps a handler.
type Middleware func(Handler) Handler

// PluginPipeline is a composable middleware chain.
type PluginPipeline struct {
	mu          sync.Mutex
	middlewares []Middleware
}

// Add appends a middleware.
func (p *PluginPipeline) Add(m Middleware) {
	p.mu.Lock()
	p.middlewares = append(p.middlewares, m)
	p.mu.Unlock()
}

// Build returns the composed handler for an initial request.
func (p *PluginPipeline) Build(initial any) Handler {
	p.mu.Lock()
	middlewares := append([]Middleware(nil), p.middlewares...)
	p.mu.Unlock()
	next := func(req any) any { return req }
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}
	return func(req any) any { return next(req) }
}

// HotSwappablePlugin reloads a plugin manifest when its file changes.
type HotSwappablePlugin struct {
	Path         string
	LastModified time.Time
	Manifest     mofu.PluginManifest
	mu           sync.Mutex
}

// NewHotSwappablePlugin creates a hot-swap wrapper.
func NewHotSwappablePlugin(path string) *HotSwappablePlugin {
	return &HotSwappablePlugin{Path: path}
}

// CheckReload checks file modification time.
func (h *HotSwappablePlugin) CheckReload() (bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	info, err := os.Stat(h.Path)
	if err != nil {
		return false, err
	}
	if info.ModTime().After(h.LastModified) {
		h.LastModified = info.ModTime()
		return true, nil
	}
	return false, nil
}

// PluginSandbox restricts plugin access to explicit capabilities.
type PluginSandbox struct {
	mu           sync.Mutex
	AllowedAPIs  map[string]bool
	AllowedKeys  []string
	Capabilities []string
}

// NewPluginSandbox returns a sandbox with an allowlist.
func NewPluginSandbox(capabilities []string) *PluginSandbox {
	return &PluginSandbox{AllowedAPIs: make(map[string]bool), Capabilities: capabilities}
}

// AllowAPI grants an API.
func (s *PluginSandbox) AllowAPI(name string) {
	s.mu.Lock()
	s.AllowedAPIs[name] = true
	s.mu.Unlock()
}

// CanUseAPI reports whether a plugin can use an API.
func (s *PluginSandbox) CanUseAPI(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.AllowedAPIs[name]
}

// IsolateFailure runs fn and converts panics/errors to PluginError.
func IsolateFailure(pluginName string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &PluginError{Plugin: pluginName, Err: fmt.Errorf("panic: %v", r)}
		}
	}()
	if err := fn(); err != nil {
		return &PluginError{Plugin: pluginName, Err: err}
	}
	return nil
}

// WatchHotPlugins periodically checks plugin files and calls reload.
func WatchHotPlugins(paths []string, reload func(path string) error, interval time.Duration) *PluginWatcher {
	w := &PluginWatcher{paths: paths, reload: reload, interval: interval, stop: make(chan struct{})}
	go w.run()
	return w
}

// PluginWatcher watches plugin files.
type PluginWatcher struct {
	paths    []string
	reload   func(path string) error
	interval time.Duration
	stop     chan struct{}
}

// Stop stops watching.
func (w *PluginWatcher) Stop() { close(w.stop) }

func (w *PluginWatcher) run() {
	last := make(map[string]time.Time)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, path := range w.paths {
				info, err := os.Stat(path)
				if err != nil || filepath.Ext(path) == "" {
					continue
				}
				if info.ModTime().After(last[path]) {
					last[path] = info.ModTime()
					_ = w.reload(path)
				}
			}
		case <-w.stop:
			return
		}
	}
}
