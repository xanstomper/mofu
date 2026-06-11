package gadgets

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Version tests
// ---------------------------------------------------------------------------

func TestVersionString(t *testing.T) {
	v := Version{1, 2, 3}
	if v.String() != "1.2.3" {
		t.Fatalf("String = %q, want 1.2.3", v.String())
	}
}

func TestVersionCompare(t *testing.T) {
	cases := []struct {
		a, b Version
		want int
	}{
		{Version{1, 0, 0}, Version{1, 0, 0}, 0},
		{Version{1, 0, 0}, Version{2, 0, 0}, -1},
		{Version{2, 0, 0}, Version{1, 0, 0}, 1},
		{Version{1, 1, 0}, Version{1, 2, 0}, -1},
		{Version{1, 0, 1}, Version{1, 0, 2}, -1},
	}
	for _, c := range cases {
		got := c.a.Compare(c.b)
		if got != c.want {
			t.Errorf("%s.Compare(%s) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestVersionCompatible(t *testing.T) {
	a := Version{1, 0, 0}
	b := Version{1, 5, 0}
	c := Version{2, 0, 0}

	if !a.Compatible(b) {
		t.Fatal("1.0.0 should be compatible with 1.5.0")
	}
	if a.Compatible(c) {
		t.Fatal("1.0.0 should not be compatible with 2.0.0")
	}
}

func TestParseVersion(t *testing.T) {
	v, err := ParseVersion("1.2.3")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if v.Major != 1 || v.Minor != 2 || v.Patch != 3 {
		t.Fatalf("parsed = %v, want 1.2.3", v)
	}

	_, err = ParseVersion("invalid")
	if err == nil {
		t.Fatal("invalid version should error")
	}
}

func TestMustParseVersion(t *testing.T) {
	v := MustParseVersion("2.0.1")
	if v.Major != 2 {
		t.Fatalf("major = %d, want 2", v.Major)
	}
}

// ---------------------------------------------------------------------------
// Registry tests
// ---------------------------------------------------------------------------

func newTestPlugin(name string, version string) (PluginManifest, func() Gadget) {
	return PluginManifest{
		Name:        name,
		Version:     MustParseVersion(version),
		Description: "test plugin " + name,
	}, func() Gadget { return newMockGadget(name) }
}

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry()
	manifest, factory := newTestPlugin("test", "1.0.0")

	if err := r.Register(manifest, factory); err != nil {
		t.Fatalf("register: %v", err)
	}

	entry := r.Get("test")
	if entry == nil {
		t.Fatal("Get returned nil")
	}
	if entry.Manifest.Version.String() != "1.0.0" {
		t.Fatalf("version = %s, want 1.0.0", entry.Manifest.Version)
	}
}

func TestRegistryDuplicateRegister(t *testing.T) {
	r := NewRegistry()
	manifest, factory := newTestPlugin("test", "1.0.0")
	r.Register(manifest, factory)

	err := r.Register(manifest, factory)
	if err == nil {
		t.Fatal("duplicate register should error")
	}
}

func TestRegistryVersionUpgrade(t *testing.T) {
	r := NewRegistry()
	m1, f1 := newTestPlugin("test", "1.0.0")
	m2, f2 := newTestPlugin("test", "1.1.0")

	r.Register(m1, f1)
	if err := r.Register(m2, f2); err != nil {
		t.Fatalf("upgrade should succeed: %v", err)
	}

	entry := r.Get("test")
	if entry.Manifest.Version.String() != "1.1.0" {
		t.Fatalf("version = %s, want 1.1.0", entry.Manifest.Version)
	}
}

func TestRegistryVersionDowngrade(t *testing.T) {
	r := NewRegistry()
	m1, f1 := newTestPlugin("test", "1.1.0")
	m2, f2 := newTestPlugin("test", "1.0.0")

	r.Register(m1, f1)
	err := r.Register(m2, f2)
	if err == nil {
		t.Fatal("downgrade should fail")
	}
}

func TestRegistryUnregister(t *testing.T) {
	r := NewRegistry()
	manifest, factory := newTestPlugin("test", "1.0.0")
	r.Register(manifest, factory)

	if err := r.Unregister("test"); err != nil {
		t.Fatalf("unregister: %v", err)
	}

	if r.Get("test") != nil {
		t.Fatal("should be gone after unregister")
	}
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry()
	r.Register(newTestPlugin("b", "1.0.0"))
	r.Register(newTestPlugin("a", "1.0.0"))
	r.Register(newTestPlugin("c", "1.0.0"))

	names := r.List()
	if len(names) != 3 || names[0] != "a" || names[1] != "b" || names[2] != "c" {
		t.Fatalf("list = %v, want [a b c]", names)
	}
}

// ---------------------------------------------------------------------------
// Dependency Resolution tests
// ---------------------------------------------------------------------------

func TestResolveDependencies(t *testing.T) {
	r := NewRegistry()

	r.Register(PluginManifest{
		Name:    "core",
		Version: MustParseVersion("1.0.0"),
	}, func() Gadget { return newMockGadget("core") })

	r.Register(PluginManifest{
		Name:    "ext",
		Version: MustParseVersion("1.0.0"),
		Dependencies: []PluginDependency{
			{Name: "core", Version: MustParseVersion("1.0.0")},
		},
	}, func() Gadget { return newMockGadget("ext") })

	order, err := r.ResolveDependencies()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	// core must come before ext
	coreIdx, extIdx := -1, -1
	for i, name := range order {
		switch name {
		case "core":
			coreIdx = i
		case "ext":
			extIdx = i
		}
	}
	if coreIdx >= extIdx {
		t.Fatalf("core should come before ext: %v", order)
	}
}

func TestResolveMissingDependency(t *testing.T) {
	r := NewRegistry()
	r.Register(PluginManifest{
		Name:    "ext",
		Version: MustParseVersion("1.0.0"),
		Dependencies: []PluginDependency{
			{Name: "missing", Version: MustParseVersion("1.0.0")},
		},
	}, func() Gadget { return newMockGadget("ext") })

	_, err := r.ResolveDependencies()
	if err == nil {
		t.Fatal("missing dependency should error")
	}
}

func TestResolveVersionMismatch(t *testing.T) {
	r := NewRegistry()
	r.Register(PluginManifest{
		Name:    "core",
		Version: MustParseVersion("1.0.0"),
	}, func() Gadget { return newMockGadget("core") })

	r.Register(PluginManifest{
		Name:    "ext",
		Version: MustParseVersion("1.0.0"),
		Dependencies: []PluginDependency{
			{Name: "core", Version: MustParseVersion("2.0.0")},
		},
	}, func() Gadget { return newMockGadget("ext") })

	_, err := r.ResolveDependencies()
	if err == nil {
		t.Fatal("version mismatch should error")
	}
}

func TestResolveCircularDependency(t *testing.T) {
	r := NewRegistry()
	r.Register(PluginManifest{
		Name:    "a",
		Version: MustParseVersion("1.0.0"),
		Dependencies: []PluginDependency{
			{Name: "b", Version: MustParseVersion("1.0.0")},
		},
	}, func() Gadget { return newMockGadget("a") })

	r.Register(PluginManifest{
		Name:    "b",
		Version: MustParseVersion("1.0.0"),
		Dependencies: []PluginDependency{
			{Name: "a", Version: MustParseVersion("1.0.0")},
		},
	}, func() Gadget { return newMockGadget("b") })

	_, err := r.ResolveDependencies()
	if err == nil {
		t.Fatal("circular dependency should error")
	}
}

// ---------------------------------------------------------------------------
// Load / Unload tests
// ---------------------------------------------------------------------------

func TestRegistryLoad(t *testing.T) {
	r := NewRegistry()
	r.Register(newTestPlugin("test", "1.0.0"))

	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	inst, err := r.Load("test", ctx)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if inst == nil {
		t.Fatal("instance should not be nil")
	}

	entry := r.Get("test")
	if !entry.Loaded {
		t.Fatal("should be marked as loaded")
	}
}

func TestRegistryLoadAlreadyLoaded(t *testing.T) {
	r := NewRegistry()
	r.Register(newTestPlugin("test", "1.0.0"))

	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	r.Load("test", ctx)

	_, err := r.Load("test", ctx)
	if err == nil {
		t.Fatal("double load should error")
	}
}

func TestRegistryUnload(t *testing.T) {
	r := NewRegistry()
	r.Register(newTestPlugin("test", "1.0.0"))

	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	r.Load("test", ctx)

	if err := r.Unload("test"); err != nil {
		t.Fatalf("unload: %v", err)
	}

	entry := r.Get("test")
	if entry.Loaded {
		t.Fatal("should not be loaded after unload")
	}
}

func TestRegistryLoadAll(t *testing.T) {
	r := NewRegistry()
	r.Register(PluginManifest{
		Name:    "core",
		Version: MustParseVersion("1.0.0"),
	}, func() Gadget { return newMockGadget("core") })

	r.Register(PluginManifest{
		Name:    "ext",
		Version: MustParseVersion("1.0.0"),
		Dependencies: []PluginDependency{
			{Name: "core", Version: MustParseVersion("1.0.0")},
		},
	}, func() Gadget { return newMockGadget("ext") })

	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	instances, errs := r.LoadAll(ctx)

	if len(errs) > 0 {
		t.Fatalf("load all errors: %v", errs)
	}
	if len(instances) != 2 {
		t.Fatalf("loaded %d, want 2", len(instances))
	}
}

func TestRegistryUnloadAll(t *testing.T) {
	r := NewRegistry()
	r.Register(newTestPlugin("a", "1.0.0"))
	r.Register(newTestPlugin("b", "1.0.0"))

	ctx := GadgetContext{Binder: NewBinder(), Logger: func(f string, a ...any) {}}
	r.LoadAll(ctx)

	errs := r.UnloadAll()
	if len(errs) > 0 {
		t.Fatalf("unload all errors: %v", errs)
	}
}

// ---------------------------------------------------------------------------
// Conflict Detection tests
// ---------------------------------------------------------------------------

func TestDetectConflicts(t *testing.T) {
	r := NewRegistry()
	r.Register(PluginManifest{
		Name:         "a",
		Version:      MustParseVersion("1.0.0"),
		Capabilities: []string{"logging"},
	}, func() Gadget { return newMockGadget("a") })

	r.Register(PluginManifest{
		Name:         "b",
		Version:      MustParseVersion("1.0.0"),
		Capabilities: []string{"logging"},
	}, func() Gadget { return newMockGadget("b") })

	conflicts := r.DetectConflicts()
	if len(conflicts) != 1 {
		t.Fatalf("conflicts = %d, want 1", len(conflicts))
	}
}

// ---------------------------------------------------------------------------
// Search tests
// ---------------------------------------------------------------------------

func TestSearch(t *testing.T) {
	r := NewRegistry()
	r.Register(PluginManifest{
		Name:        "logger",
		Version:     MustParseVersion("1.0.0"),
		Author:      "alice",
		Tags:        []string{"logging", "output"},
		Capabilities: []string{"log"},
	}, func() Gadget { return newMockGadget("logger") })

	r.Register(PluginManifest{
		Name:        "metrics",
		Version:     MustParseVersion("1.0.0"),
		Author:      "bob",
		Tags:        []string{"monitoring"},
		Capabilities: []string{"metrics"},
	}, func() Gadget { return newMockGadget("metrics") })

	// Search by tag
	results := r.Search(SearchOptions{Tag: "logging"})
	if len(results) != 1 || results[0].Name != "logger" {
		t.Fatalf("tag search = %v, want [logger]", results)
	}

	// Search by author
	results = r.Search(SearchOptions{Author: "bob"})
	if len(results) != 1 || results[0].Name != "metrics" {
		t.Fatalf("author search = %v, want [metrics]", results)
	}

	// Search by name pattern
	results = r.Search(SearchOptions{NamePattern: "log"})
	if len(results) != 1 {
		t.Fatalf("name search = %v, want 1 result", results)
	}
}

// ---------------------------------------------------------------------------
// OnChange tests
// ---------------------------------------------------------------------------

func TestRegistryOnChange(t *testing.T) {
	r := NewRegistry()
	var actions []string
	r.OnChange(func(name, action string) {
		actions = append(actions, name+":"+action)
	})

	r.Register(newTestPlugin("test", "1.0.0"))
	r.Unregister("test")

	if len(actions) != 2 {
		t.Fatalf("actions = %d, want 2: %v", len(actions), actions)
	}
	if actions[0] != "test:registered" || actions[1] != "test:unregistered" {
		t.Fatalf("actions = %v", actions)
	}
}
