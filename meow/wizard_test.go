package meow

import (
	"testing"

	"github.com/xanstomper/mofu"
)

func TestWizardCreation(t *testing.T) {
	w := NewWizard(
		WizardStep{
			Title:  "Personal Info",
			Fields: []*Field{Input("name", "Name").SetRequired()},
		},
		WizardStep{
			Title:  "Contact",
			Fields: []*Field{Email("email", "Email")},
		},
	)

	if w.CurrentStep() != 0 {
		t.Fatalf("current step = %d, want 0", w.CurrentStep())
	}
	if w.Title() != "Personal Info" {
		t.Fatalf("title = %q, want 'Personal Info'", w.Title())
	}
	if len(w.Steps()) != 2 {
		t.Fatalf("steps = %d, want 2", len(w.Steps()))
	}
}

func TestWizardNavigation(t *testing.T) {
	w := NewWizard(
		WizardStep{Title: "Step 1", Fields: []*Field{Input("a", "A")}},
		WizardStep{Title: "Step 2", Fields: []*Field{Input("b", "B")}},
		WizardStep{Title: "Step 3", Fields: []*Field{Input("c", "C")}},
	)

	if !w.Next() {
		t.Fatal("Next should succeed")
	}
	if w.CurrentStep() != 1 {
		t.Fatalf("step = %d, want 1", w.CurrentStep())
	}

	if !w.Next() {
		t.Fatal("Next should succeed")
	}
	if w.CurrentStep() != 2 {
		t.Fatalf("step = %d, want 2", w.CurrentStep())
	}

	// Should not go past last
	if w.Next() {
		t.Fatal("Next on last step should fail")
	}

	if !w.Prev() {
		t.Fatal("Prev should succeed")
	}
	if w.CurrentStep() != 1 {
		t.Fatalf("step = %d, want 1", w.CurrentStep())
	}
}

func TestWizardFirstLast(t *testing.T) {
	w := NewWizard(
		WizardStep{Title: "Step 1", Fields: []*Field{Input("a", "A")}},
		WizardStep{Title: "Step 2", Fields: []*Field{Input("b", "B")}},
	)

	if !w.IsFirst() {
		t.Fatal("should be first")
	}
	if w.IsLast() {
		t.Fatal("should not be last")
	}

	w.Next()

	if w.IsFirst() {
		t.Fatal("should not be first")
	}
	if !w.IsLast() {
		t.Fatal("should be last")
	}
}

func TestWizardProgress(t *testing.T) {
	w := NewWizard(
		WizardStep{Title: "Step 1", Fields: []*Field{Input("a", "A")}},
		WizardStep{Title: "Step 2", Fields: []*Field{Input("b", "B")}},
		WizardStep{Title: "Step 3", Fields: []*Field{Input("c", "C")}},
	)

	if w.Progress() != 0 {
		t.Fatalf("progress = %v, want 0", w.Progress())
	}

	w.Next()
	if w.Progress() != 0.5 {
		t.Fatalf("progress = %v, want 0.5", w.Progress())
	}

	w.Next()
	if w.Progress() != 1.0 {
		t.Fatalf("progress = %v, want 1.0", w.Progress())
	}
}

func TestWizardGoTo(t *testing.T) {
	w := NewWizard(
		WizardStep{Title: "Step 1", Fields: []*Field{Input("a", "A")}},
		WizardStep{Title: "Step 2", Fields: []*Field{Input("b", "B")}},
		WizardStep{Title: "Step 3", Fields: []*Field{Input("c", "C")}},
	)

	if !w.GoTo(2) {
		t.Fatal("GoTo(2) should succeed")
	}
	if w.CurrentStep() != 2 {
		t.Fatalf("step = %d, want 2", w.CurrentStep())
	}

	if w.GoTo(-1) {
		t.Fatal("GoTo(-1) should fail")
	}
	if w.GoTo(5) {
		t.Fatal("GoTo(5) should fail")
	}
}

func TestWizardValidation(t *testing.T) {
	w := NewWizard(
		WizardStep{
			Title:  "Step 1",
			Fields: []*Field{Input("name", "Name").SetRequired()},
			Valid: func(values map[string]any) bool {
				v, ok := values["name"]
				return ok && v != nil && v != ""
			},
		},
		WizardStep{
			Title:  "Step 2",
			Fields: []*Field{Email("email", "Email")},
		},
	)

	// Should not advance without name
	if w.Next() {
		t.Fatal("should not advance without name")
	}

	w.form.SetValue("name", "Alice")
	if !w.Next() {
		t.Fatal("should advance with name")
	}
}

func TestWizardOnChange(t *testing.T) {
	w := NewWizard(
		WizardStep{Title: "Step 1", Fields: []*Field{Input("a", "A")}},
		WizardStep{Title: "Step 2", Fields: []*Field{Input("b", "B")}},
	)

	var changedTo int
	w.OnChange(func(step int) {
		changedTo = step
	})

	w.Next()
	if changedTo != 1 {
		t.Fatalf("onChange called with %d, want 1", changedTo)
	}
}

func TestWizardComplete(t *testing.T) {
	w := NewWizard(
		WizardStep{Title: "Step 1", Fields: []*Field{Input("a", "A")}},
		WizardStep{Title: "Step 2", Fields: []*Field{Input("b", "B")}},
	)

	var completed bool
	w.OnComplete(func(values map[string]any) mofu.Cmd {
		completed = true
		return nil
	})

	w.form.SetValue("a", "1")
	w.form.SetValue("b", "2")

	w.Finish()
	if !completed {
		t.Fatal("OnComplete should have been called")
	}
}

func TestWizardStepStatus(t *testing.T) {
	w := NewWizard(
		WizardStep{
			Title:  "Step 1",
			Fields: []*Field{Input("a", "A")},
			Valid: func(values map[string]any) bool {
				v, ok := values["a"]
				return ok && v != nil && v != ""
			},
		},
		WizardStep{
			Title:  "Step 2",
			Fields: []*Field{Input("b", "B")},
			Valid: func(values map[string]any) bool {
				v, ok := values["b"]
				return ok && v != nil && v != ""
			},
		},
	)

	status := w.StepStatus()
	if status[0] || status[1] {
		t.Fatalf("both steps should be invalid: %v", status)
	}

	w.form.SetValue("a", "hello")
	status = w.StepStatus()
	if !status[0] || status[1] {
		t.Fatalf("step 0 should be valid: %v", status)
	}
}

func TestWizardVisibleFields(t *testing.T) {
	w := NewWizard(
		WizardStep{
			Title: "Step 1",
			Fields: []*Field{
				Input("a", "A"),
				Input("b", "B").When(func(v map[string]any) bool {
					return v["a"] == "show"
				}),
			},
		},
	)

	fields := w.VisibleFields()
	if len(fields) != 1 {
		t.Fatalf("visible fields = %d, want 1 (b is hidden)", len(fields))
	}

	w.form.SetValue("a", "show")
	fields = w.VisibleFields()
	if len(fields) != 2 {
		t.Fatalf("visible fields = %d, want 2 (b is visible)", len(fields))
	}
}

func TestWizardRenderData(t *testing.T) {
	w := NewWizard(
		WizardStep{Title: "Step 1", Fields: []*Field{Input("a", "A")}},
		WizardStep{Title: "Step 2", Fields: []*Field{Input("b", "B")}},
	)

	data := w.RenderData()
	if data.CurrentStep != 0 {
		t.Fatalf("CurrentStep = %d, want 0", data.CurrentStep)
	}
	if data.TotalSteps != 2 {
		t.Fatalf("TotalSteps = %d, want 2", data.TotalSteps)
	}
	if data.Title != "Step 1" {
		t.Fatalf("Title = %q, want 'Step 1'", data.Title)
	}
	if !data.IsFirst {
		t.Fatal("should be first")
	}
	if data.IsLast {
		t.Fatal("should not be last")
	}
}

func TestRenderProgressBar(t *testing.T) {
	bar := RenderProgressBar(0, 3, 20)
	if len(bar) == 0 {
		t.Fatal("progress bar should not be empty")
	}

	bar2 := RenderProgressBar(2, 3, 20)
	if len(bar2) == 0 {
		t.Fatal("progress bar should not be empty")
	}
}

func TestRenderStepIndicator(t *testing.T) {
	ind := RenderStepIndicator(1, 3)
	if ind != "✓ ● ○" {
		t.Fatalf("indicator = %q, want '✓ ● ○'", ind)
	}
}
