package meow

import (
	"fmt"

	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// Form State Tracking
// ---------------------------------------------------------------------------

// FormState tracks the overall state of a form.
type FormState struct {
	Dirty    bool // any field changed from initial
	Touched  bool // any field received focus
	Submitting bool // form is being submitted
	Submitted  bool // form was submitted
}

// WizardStep is a single step in a multi-step wizard.
type WizardStep struct {
	Title  string
	Fields []*Field
	Valid  func(values map[string]any) bool
}

// Wizard manages multi-step form navigation.
type Wizard struct {
	steps      []WizardStep
	current    int
	form       *Form
	onComplete func(values map[string]any) mofu.Cmd
	onChange   func(step int)
}

// NewWizard creates a multi-step wizard from steps.
func NewWizard(steps ...WizardStep) *Wizard {
	// Collect all fields from all steps
	var allFields []*Field
	for _, step := range steps {
		allFields = append(allFields, step.Fields...)
	}

	w := &Wizard{
		steps: steps,
		form:  NewForm(allFields...),
	}
	return w
}

// CurrentStep returns the current step index.
func (w *Wizard) CurrentStep() int {
	return w.current
}

// Step returns the current WizardStep.
func (w *Wizard) Step() WizardStep {
	return w.steps[w.current]
}

// Steps returns all steps.
func (w *Wizard) Steps() []WizardStep {
	return w.steps
}

// Title returns the current step title.
func (w *Wizard) Title() string {
	return w.steps[w.current].Title
}

// IsFirst reports whether this is the first step.
func (w *Wizard) IsFirst() bool {
	return w.current == 0
}

// IsLast reports whether this is the last step.
func (w *Wizard) IsLast() bool {
	return w.current == len(w.steps)-1
}

// Progress returns the current progress (0.0 to 1.0).
func (w *Wizard) Progress() float64 {
	if len(w.steps) <= 1 {
		return 1
	}
	return float64(w.current) / float64(len(w.steps)-1)
}

// Next advances to the next step if the current step is valid.
func (w *Wizard) Next() bool {
	if w.IsLast() {
		return false
	}

	// Validate current step
	step := w.steps[w.current]
	if step.Valid != nil && !step.Valid(w.form.Values()) {
		return false
	}

	w.current++
	if w.onChange != nil {
		w.onChange(w.current)
	}
	return true
}

// Prev goes back to the previous step.
func (w *Wizard) Prev() bool {
	if w.IsFirst() {
		return false
	}
	w.current--
	if w.onChange != nil {
		w.onChange(w.current)
	}
	return true
}

// GoTo jumps to a specific step.
func (w *Wizard) GoTo(step int) bool {
	if step < 0 || step >= len(w.steps) {
		return false
	}
	w.current = step
	if w.onChange != nil {
		w.onChange(w.current)
	}
	return true
}

// Form returns the underlying form.
func (w *Wizard) Form() *Form {
	return w.form
}

// OnComplete registers a callback when the wizard completes.
func (w *Wizard) OnComplete(fn func(values map[string]any) mofu.Cmd) *Wizard {
	w.onComplete = fn
	return w
}

// OnChange registers a callback when the step changes.
func (w *Wizard) OnChange(fn func(step int)) *Wizard {
	w.onChange = fn
	return w
}

// Finish validates all steps and submits.
func (w *Wizard) Finish() mofu.Cmd {
	// Validate all steps
	for _, step := range w.steps {
		if step.Valid != nil && !step.Valid(w.form.Values()) {
			return nil
		}
	}

	w.form.validateAll()
	if !w.form.Valid() {
		return nil
	}

	if w.onComplete != nil {
		return w.onComplete(w.form.Values())
	}
	return nil
}

// VisibleFields returns only the fields visible in the current step.
func (w *Wizard) VisibleFields() []*Field {
	step := w.steps[w.current]
	var visible []*Field
	for _, f := range step.Fields {
		if f.Visible == nil || f.Visible(w.form.Values()) {
			visible = append(visible, f)
		}
	}
	return visible
}

// StepStatus returns the validation status of each step.
func (w *Wizard) StepStatus() []bool {
	status := make([]bool, len(w.steps))
	for i, step := range w.steps {
		if step.Valid != nil {
			status[i] = step.Valid(w.form.Values())
		} else {
			status[i] = true
		}
	}
	return status
}

// ---------------------------------------------------------------------------
// Multi-step form rendering helpers
// ---------------------------------------------------------------------------

// WizardRenderData contains all data needed to render a wizard.
type WizardRenderData struct {
	CurrentStep int
	TotalSteps  int
	Title       string
	Fields      []*Field
	Values      map[string]any
	Errors      map[string]error
	IsFirst     bool
	IsLast      bool
	Progress    float64
	StepStatus  []bool
}

// RenderData returns the data needed to render the current wizard state.
func (w *Wizard) RenderData() WizardRenderData {
	return WizardRenderData{
		CurrentStep: w.current,
		TotalSteps:  len(w.steps),
		Title:       w.Title(),
		Fields:      w.VisibleFields(),
		Values:      w.form.Values(),
		Errors:      w.form.Errors(),
		IsFirst:     w.IsFirst(),
		IsLast:      w.IsLast(),
		Progress:    w.Progress(),
		StepStatus:  w.StepStatus(),
	}
}

// RenderProgressBar renders a step progress bar.
func RenderProgressBar(current, total int, width int) string {
	if total <= 1 {
		return ""
	}
	filled := int(float64(width) * float64(current) / float64(total-1))
	if filled > width {
		filled = width
	}
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return fmt.Sprintf("[%s] Step %d/%d", bar, current+1, total)
}

// RenderStepIndicator renders dots for each step.
func RenderStepIndicator(current, total int) string {
	result := ""
	for i := 0; i < total; i++ {
		if i > 0 {
			result += " "
		}
		if i == current {
			result += "●"
		} else if i < current {
			result += "✓"
		} else {
			result += "○"
		}
	}
	return result
}
