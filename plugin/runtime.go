package plugin

import (
	"fmt"
	"sync"

	"github.com/xanstomper/mofu"
	"github.com/xanstomper/mofu/kernel"
	"github.com/xanstomper/mofu/message"
	"github.com/xanstomper/mofu/state"
)

type Runtime struct {
	mu      sync.Mutex
	plugins map[string]*Instance
	kernel  *kernel.Kernel
}

type Instance struct {
	Plugin mofu.Plugin
	State  mofu.PluginStatus
	Realm  *mofu.PluginStateRealm
}

type Sandbox struct {
	State  *state.Graph
	Bus    *message.Bus
	Kernel *kernel.Kernel
}

func NewRuntime(k *kernel.Kernel) *Runtime {
	return &Runtime{
		plugins: make(map[string]*Instance),
		kernel:  k,
	}
}

func (r *Runtime) Register(plugin mofu.Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := plugin.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	sandbox := &Sandbox{
		State:  r.kernel.State,
		Bus:    r.kernel.Bus,
		Kernel: r.kernel,
	}

	ctx := &mofu.PluginContext{
		DataStore:  nil,
		EventBus:   nil,
		Theme:      nil,
		StateRealm: mofu.NewStateRealm(name),
	}

	if err := plugin.Init(ctx); err != nil {
		return fmt.Errorf("plugin %s init failed: %w", name, err)
	}

	r.plugins[name] = &Instance{
		Plugin: plugin,
		State:  mofu.PluginActive,
		Realm:  ctx.StateRealm,
	}

	_ = sandbox
	return nil
}

func (r *Runtime) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	inst, ok := r.plugins[name]
	if !ok {
		return fmt.Errorf("plugin %s not found", name)
	}

	inst.Plugin.Shutdown()
	inst.State = mofu.PluginUnloaded
	delete(r.plugins, name)
	return nil
}

func (r *Runtime) Get(name string) *Instance {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.plugins[name]
}

func (r *Runtime) All() []*Instance {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Instance, 0, len(r.plugins))
	for _, inst := range r.plugins {
		out = append(out, inst)
	}
	return out
}

func (r *Runtime) Active() []*Instance {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*Instance
	for _, inst := range r.plugins {
		if inst.State == mofu.PluginActive {
			out = append(out, inst)
		}
	}
	return out
}

func (r *Runtime) DispatchAll(msg message.Message) {
	for _, inst := range r.Active() {
		ev := mofu.Event{
			Type: mofu.EventSystem,
			Data: msg.Payload,
		}
		actions := inst.Plugin.OnEvent(ev)
		for _, action := range actions {
			if action.Type == "command" {
				r.kernel.Bus.Publish(message.NewCommand("plugin", action.Payload))
			}
		}
	}
}

func (r *Runtime) BroadcastAll(msg message.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inst := range r.plugins {
		if inst.State == mofu.PluginActive {
			ev := mofu.Event{
				Type: mofu.EventSystem,
				Data: msg.Payload,
			}
			inst.Plugin.OnEvent(ev)
		}
	}
}
