package mofu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type PluginStatus int

const (
	PluginDiscovered PluginStatus = iota
	PluginResolved
	PluginLoaded
	PluginInitialized
	PluginActive
	PluginDisabled
	PluginFailed
	PluginUnloaded
)

type Plugin interface {
	Name() string
	Version() string
	Init(ctx *PluginContext) error
	Shutdown() error
	OnEvent(event Event) []PluginAction
	OnRender(ctx *RenderContext)
	OnTick(dt float64)
}

type PluginAction struct {
	Type    string
	Payload Msg
}

type PluginContext struct {
	DataStore  *DataStore
	EventBus   *EventBus
	Theme      *Theme
	StateRealm *PluginStateRealm
}

type PluginStateRealm struct {
	namespace string
	inner     map[string]json.RawMessage
	mu        sync.Mutex
}

func NewStateRealm(namespace string) *PluginStateRealm {
	return &PluginStateRealm{
		namespace: namespace,
		inner:     make(map[string]json.RawMessage),
	}
}

func (r *PluginStateRealm) Get(key string) (json.RawMessage, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.inner[key]
	return v, ok
}

func (r *PluginStateRealm) Set(key string, value json.RawMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inner[key] = value
}

type PluginManifest struct {
	Name         string   `json:"name" yaml:"name"`
	Version      string   `json:"version" yaml:"version"`
	Description  string   `json:"description" yaml:"description"`
	Capabilities []string `json:"capabilities" yaml:"capabilities"`
	Layers       []string `json:"layers" yaml:"layers"`
	Load         string   `json:"load" yaml:"load"`
	HotReload    bool     `json:"hot_reload" yaml:"hot_reload"`
}

type PluginInstance struct {
	Plugin   Plugin
	Manifest PluginManifest
	State    PluginStatus
	Realm    *PluginStateRealm
}

type EventFilter func(ctx *PluginContext, event Event) *Event

type PipelineStage struct {
	PluginID string
	Filter   EventFilter
}

type PluginManager struct {
	mu       sync.Mutex
	plugins  map[string]*PluginInstance
	pipeline []PipelineStage
	ctx      *PluginContext
	watchDir string
	onChange []func(string, PluginStatus)
}

func NewPluginManager(ctx *PluginContext) *PluginManager {
	return &PluginManager{
		plugins:  make(map[string]*PluginInstance),
		pipeline: make([]PipelineStage, 0),
		ctx:      ctx,
	}
}

func (pm *PluginManager) Register(plugin Plugin) error {
	return pm.RegisterWithManifest(plugin, PluginManifest{
		Name:    plugin.Name(),
		Version: plugin.Version(),
	})
}

func (pm *PluginManager) RegisterWithManifest(plugin Plugin, manifest PluginManifest) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	name := plugin.Name()
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	realm := NewStateRealm(name)

	inst := &PluginInstance{
		Plugin:   plugin,
		Manifest: manifest,
		State:    PluginDiscovered,
		Realm:    realm,
	}
	pm.plugins[name] = inst

	if err := pm.initPlugin(inst); err != nil {
		inst.State = PluginFailed
		return err
	}

	return nil
}

func (pm *PluginManager) initPlugin(inst *PluginInstance) error {
	inst.State = PluginResolved
	inst.State = PluginLoaded

	pluginCtx := &PluginContext{
		DataStore:  pm.ctx.DataStore,
		EventBus:   pm.ctx.EventBus,
		Theme:      pm.ctx.Theme,
		StateRealm: inst.Realm,
	}

	if err := inst.Plugin.Init(pluginCtx); err != nil {
		inst.State = PluginFailed
		return fmt.Errorf("plugin %s init failed: %w", inst.Plugin.Name(), err)
	}

	inst.State = PluginInitialized
	inst.State = PluginActive
	pm.fireChange(inst.Plugin.Name(), PluginActive)
	return nil
}

func (pm *PluginManager) Unregister(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	inst, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	if err := inst.Plugin.Shutdown(); err != nil {
		return err
	}

	inst.State = PluginUnloaded
	delete(pm.plugins, name)

	pruned := make([]PipelineStage, 0, len(pm.pipeline))
	for _, s := range pm.pipeline {
		if s.PluginID != name {
			pruned = append(pruned, s)
		}
	}
	pm.pipeline = pruned

	pm.fireChange(name, PluginUnloaded)
	return nil
}

func (pm *PluginManager) AddFilter(pluginID string, filter EventFilter) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pipeline = append(pm.pipeline, PipelineStage{PluginID: pluginID, Filter: filter})
}

func (pm *PluginManager) ProcessEvent(event Event) Event {
	pm.mu.Lock()
	pipeline := make([]PipelineStage, len(pm.pipeline))
	copy(pipeline, pm.pipeline)
	pm.mu.Unlock()

	current := event
	for _, stage := range pipeline {
		inst, ok := pm.plugins[stage.PluginID]
		if !ok || inst.State != PluginActive {
			continue
		}
		result := stage.Filter(pm.ctx, current)
		if result == nil {
			return Event{}
		}
		current = *result
	}
	return current
}

func (pm *PluginManager) DispatchRender(ctx *RenderContext) {
	pm.mu.Lock()
	active := make([]*PluginInstance, 0, len(pm.plugins))
	for _, inst := range pm.plugins {
		if inst.State == PluginActive {
			active = append(active, inst)
		}
	}
	pm.mu.Unlock()

	for _, inst := range active {
		inst.Plugin.OnRender(ctx)
	}
}

func (pm *PluginManager) DispatchTick(dt float64) {
	pm.mu.Lock()
	active := make([]*PluginInstance, 0, len(pm.plugins))
	for _, inst := range pm.plugins {
		if inst.State == PluginActive {
			active = append(active, inst)
		}
	}
	pm.mu.Unlock()

	for _, inst := range active {
		inst.Plugin.OnTick(dt)
	}
}

func (pm *PluginManager) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".json" && ext != ".yaml" && ext != ".yml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var manifest PluginManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}

		if plugin, err := pm.loadPluginFromManifest(&manifest); err == nil {
			pm.RegisterWithManifest(plugin, manifest)
		}
	}

	return nil
}

func (pm *PluginManager) loadPluginFromManifest(m *PluginManifest) (Plugin, error) {
	return nil, fmt.Errorf("no plugin loader for manifest: %s", m.Name)
}

func (pm *PluginManager) OnChange(fn func(string, PluginStatus)) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.onChange = append(pm.onChange, fn)
}

func (pm *PluginManager) fireChange(name string, status PluginStatus) {
	for _, fn := range pm.onChange {
		fn(name, status)
	}
}

func (pm *PluginManager) Plugin(name string) *PluginInstance {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.plugins[name]
}

func (pm *PluginManager) Active() []*PluginInstance {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	out := make([]*PluginInstance, 0, len(pm.plugins))
	for _, inst := range pm.plugins {
		if inst.State == PluginActive {
			out = append(out, inst)
		}
	}
	return out
}

func (pm *PluginManager) All() []*PluginInstance {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	out := make([]*PluginInstance, 0, len(pm.plugins))
	for _, inst := range pm.plugins {
		out = append(out, inst)
	}
	return out
}
