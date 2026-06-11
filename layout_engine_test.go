package mofu_test

import (
	"testing"

	"github.com/xanstomper/mofu"
)

func TestTreeDiff(t *testing.T) {
	old := mofu.NewTreeNode("root", "box")
	old.SetProp("title", "Old Title")

	new := mofu.NewTreeNode("root", "box")
	new.SetProp("title", "New Title")

	results := mofu.DiffTrees(old, new)
	if len(results) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(results))
	}
	if results[0].Type != "update" {
		t.Errorf("expected update, got %s", results[0].Type)
	}
}

func TestTreeAddChild(t *testing.T) {
	old := mofu.NewTreeNode("root", "box")

	new := mofu.NewTreeNode("root", "box")
	child := mofu.NewTreeNode("child1", "text")
	child.SetProp("text", "Hello")
	new.AddChild(child)

	results := mofu.DiffTrees(old, new)
	if len(results) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(results))
	}
	if results[0].Type != "add" {
		t.Errorf("expected add, got %s", results[0].Type)
	}
}

func TestTreeRemoveChild(t *testing.T) {
	old := mofu.NewTreeNode("root", "box")
	child := mofu.NewTreeNode("child1", "text")
	old.AddChild(child)

	new := mofu.NewTreeNode("root", "box")

	results := mofu.DiffTrees(old, new)
	if len(results) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(results))
	}
	if results[0].Type != "remove" {
		t.Errorf("expected remove, got %s", results[0].Type)
	}
}

func TestTreeNodeFind(t *testing.T) {
	root := mofu.NewTreeNode("root", "box")
	child1 := mofu.NewTreeNode("child1", "text")
	child2 := mofu.NewTreeNode("child2", "button")
	root.AddChild(child1)
	root.AddChild(child2)

	found := root.Find("child2")
	if found == nil {
		t.Fatal("expected to find child2")
	}
	if found.Type != "button" {
		t.Errorf("expected button, got %s", found.Type)
	}
}

func TestLayoutEngine(t *testing.T) {
	engine := mofu.NewLayoutEngine(80, 24)

	root := &mofu.LayoutNode{
		Children: []*mofu.LayoutNode{
			{MinWidth: 20, Fixed: true},
			{Grow: 1},
			{MinWidth: 30, Fixed: true},
		},
	}

	engine.SetRoot(root)
	engine.Compute()

	if root.Bounds.Width != 80 {
		t.Errorf("root width = %d, want 80", root.Bounds.Width)
	}
}

func TestResponsiveLayout(t *testing.T) {
	rl := mofu.NewResponsiveLayout()

	small := &mofu.LayoutNode{MinWidth: 0}
	medium := &mofu.LayoutNode{MinWidth: 80}
	large := &mofu.LayoutNode{MinWidth: 120}

	rl.AddBreakpoint(120, large)
	rl.AddBreakpoint(80, medium)
	rl.SetDefault(small)

	// Test different widths
	if rl.GetLayout(200) != large {
		t.Error("expected large layout for width 200")
	}
	if rl.GetLayout(100) != medium {
		t.Error("expected medium layout for width 100")
	}
	if rl.GetLayout(50) != small {
		t.Error("expected small layout for width 50")
	}
}
