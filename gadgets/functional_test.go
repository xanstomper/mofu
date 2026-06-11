package gadgets_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xanstomper/mofu/gadgets"
)

func TestRealTreeTableExpandCollapse(t *testing.T) {
	table := gadgets.NewRealTreeTable("tree")

	// Create tree structure
	root := &gadgets.RealTreeNode{
		ID:    "root",
		Label: "Root",
		Children: []*gadgets.RealTreeNode{
			{ID: "child1", Label: "Child 1"},
			{ID: "child2", Label: "Child 2"},
		},
	}
	table.AddNode(root)

	// Initially collapsed
	flat := table.GetFlatNodes()
	if len(flat) != 1 {
		t.Errorf("expected 1 visible node, got %d", len(flat))
	}

	// Expand
	table.Expand("root")
	flat = table.GetFlatNodes()
	if len(flat) != 3 {
		t.Errorf("expected 3 visible nodes after expand, got %d", len(flat))
	}

	// Collapse
	table.Collapse("root")
	flat = table.GetFlatNodes()
	if len(flat) != 1 {
		t.Errorf("expected 1 visible node after collapse, got %d", len(flat))
	}
}

func TestRealTreeTableToggle(t *testing.T) {
	table := gadgets.NewRealTreeTable("tree")
	root := &gadgets.RealTreeNode{
		ID:    "root",
		Label: "Root",
		Children: []*gadgets.RealTreeNode{
			{ID: "child1", Label: "Child 1"},
		},
	}
	table.AddNode(root)

	// Toggle expand
	table.Toggle("root")
	flat := table.GetFlatNodes()
	if len(flat) != 2 {
		t.Errorf("expected 2 nodes after toggle, got %d", len(flat))
	}

	// Toggle collapse
	table.Toggle("root")
	flat = table.GetFlatNodes()
	if len(flat) != 1 {
		t.Errorf("expected 1 node after toggle, got %d", len(flat))
	}
}

func TestRealFormValidation(t *testing.T) {
	form := gadgets.NewRealForm("form")

	// Add required field
	form.AddField(&gadgets.FormField{
		Name:     "email",
		Label:    "Email",
		Required: true,
		Validator: func(v any) error {
			s, _ := v.(string)
			if !strings.Contains(s, "@") {
				return fmt.Errorf("invalid email")
			}
			return nil
		},
	})

	// Empty value should fail
	form.SetValue("email", "")
	if form.IsValid() {
		t.Error("expected invalid form")
	}

	// Invalid email should fail
	form.SetValue("email", "invalid")
	if form.IsValid() {
		t.Error("expected invalid form")
	}

	// Valid email should pass
	form.SetValue("email", "test@example.com")
	if !form.IsValid() {
		t.Error("expected valid form")
	}
}

func TestRealKanbanMoveCard(t *testing.T) {
	kanban := gadgets.NewRealKanban("kanban")

	// Add cards to first column
	kanban.AddCard(0, &gadgets.KanbanCard{ID: "1", Title: "Task 1"})
	kanban.AddCard(0, &gadgets.KanbanCard{ID: "2", Title: "Task 2"})

	// Move card from column 0 to column 1
	kanban.MoveCard(0, 1, 0)

	// Check first column
	if len(kanban.GetColumn(0).Cards) != 1 {
		t.Errorf("expected 1 card in column 0, got %d", len(kanban.GetColumn(0).Cards))
	}

	// Check second column
	if len(kanban.GetColumn(1).Cards) != 1 {
		t.Errorf("expected 1 card in column 1, got %d", len(kanban.GetColumn(1).Cards))
	}

	if kanban.GetColumn(1).Cards[0].Title != "Task 1" {
		t.Errorf("expected Task 1, got %s", kanban.GetColumn(1).Cards[0].Title)
	}
}

func TestRealKanbanDeleteCard(t *testing.T) {
	kanban := gadgets.NewRealKanban("kanban")
	kanban.AddCard(0, &gadgets.KanbanCard{ID: "1", Title: "Task 1"})
	kanban.AddCard(0, &gadgets.KanbanCard{ID: "2", Title: "Task 2"})

	kanban.DeleteCard(0, 0)

	if len(kanban.GetColumn(0).Cards) != 1 {
		t.Errorf("expected 1 card after delete, got %d", len(kanban.GetColumn(0).Cards))
	}

	if kanban.GetColumn(0).Cards[0].Title != "Task 2" {
		t.Errorf("expected Task 2, got %s", kanban.GetColumn(0).Cards[0].Title)
	}
}
