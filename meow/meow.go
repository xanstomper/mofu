// Package meow provides a schema-driven reactive form system for MOFU.
//
// Meow replaces manual form construction with declarative schemas:
//
//	form := meow.NewForm(
//	    meow.Input("name", "Name").Required(),
//	    meow.Input("email", "Email").Email(),
//	    meow.Select("role", "Role", []string{"Admin", "User"}),
//	    meow.Checkbox("agree", "I agree to terms"),
//	)
//
//	form.OnSubmit(func(values map[string]any) mofu.Cmd {
//	    // handle submission
//	    return nil
//	})
//
// Features:
//   - Schema-driven UI generation
//   - Live validation
//   - Conditional fields
//   - Computed values
//   - Dependency tracking
package meow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/xanstomper/mofu"
)

// ---------------------------------------------------------------------------
// Field Types
// ---------------------------------------------------------------------------

// FieldType identifies the kind of form field.
type FieldType int

const (
	FieldInput FieldType = iota
	FieldSelect
	FieldCheckbox
	FieldTextarea
	FieldPassword
	FieldNumber
	FieldEmail
	FieldDate
)

// Field is a single form field.
type Field struct {
	Type        FieldType
	Name        string
	Label       string
	Value       any
	Placeholder string
	Required    bool
	Disabled    bool
	Options     []string // for select
	Validator   func(any) error
	Visible     func(values map[string]any) bool // conditional visibility
	Computed    func(values map[string]any) any  // computed value
}

// Input creates a text input field.
func Input(name, label string) *Field {
	return &Field{Type: FieldInput, Name: name, Label: label}
}

// Select creates a select field.
func Select(name, label string, options []string) *Field {
	return &Field{Type: FieldSelect, Name: name, Label: label, Options: options}
}

// Checkbox creates a checkbox field.
func Checkbox(name, label string) *Field {
	return &Field{Type: FieldCheckbox, Name: name, Label: label}
}

// Textarea creates a textarea field.
func Textarea(name, label string) *Field {
	return &Field{Type: FieldTextarea, Name: name, Label: label}
}

// Password creates a password input field.
func Password(name, label string) *Field {
	return &Field{Type: FieldPassword, Name: name, Label: label}
}

// Number creates a number input field.
func Number(name, label string) *Field {
	return &Field{Type: FieldNumber, Name: name, Label: label}
}

// Email creates an email input field.
func Email(name, label string) *Field {
	return &Field{Type: FieldEmail, Name: name, Label: label, Validator: validateEmail}
}

// Date creates a date input field.
func Date(name, label string) *Field {
	return &Field{Type: FieldDate, Name: name, Label: label}
}

// ---------------------------------------------------------------------------
// Field Modifiers
// ---------------------------------------------------------------------------

// SetRequired marks the field as required.
func (f *Field) SetRequired() *Field {
	f.Required = true
	return f
}

// SetDisabled disables the field.
func (f *Field) SetDisabled() *Field {
	f.Disabled = true
	return f
}

// SetPlaceholder sets the placeholder text.
func (f *Field) SetPlaceholder(text string) *Field {
	f.Placeholder = text
	return f
}

// Validate sets a custom validator.
func (f *Field) Validate(fn func(any) error) *Field {
	f.Validator = fn
	return f
}

// When makes the field conditionally visible.
func (f *Field) When(fn func(values map[string]any) bool) *Field {
	f.Visible = fn
	return f
}

// Compute makes the field value computed from other fields.
func (f *Field) Compute(fn func(values map[string]any) any) *Field {
	f.Computed = fn
	return f
}

// ---------------------------------------------------------------------------
// Form
// ---------------------------------------------------------------------------

// Form is a schema-driven reactive form.
type Form struct {
	fields   []*Field
	values   map[string]any
	errors   map[string]error
	focus    int
	onChange func(name string, value any)
	onSubmit func(values map[string]any) mofu.Cmd
	dirty    bool
}

// NewForm creates a form from field definitions.
func NewForm(fields ...*Field) *Form {
	f := &Form{
		fields: fields,
		values: make(map[string]any),
		errors: make(map[string]error),
	}
	// Initialize values
	for _, field := range fields {
		if field.Computed == nil {
			f.values[field.Name] = field.Value
		}
	}
	return f
}

// OnChange registers a callback when any field changes.
func (f *Form) OnChange(fn func(name string, value any)) *Form {
	f.onChange = fn
	return f
}

// OnSubmit registers a callback when the form is submitted.
func (f *Form) OnSubmit(fn func(values map[string]any) mofu.Cmd) *Form {
	f.onSubmit = fn
	return f
}

// Values returns the current form values.
func (f *Form) Values() map[string]any {
	return f.values
}

// Value returns a single field value.
func (f *Form) Value(name string) any {
	return f.values[name]
}

// SetValue sets a field value.
func (f *Form) SetValue(name string, value any) {
	f.values[name] = value
	f.dirty = true

	// Recompute computed fields
	for _, field := range f.fields {
		if field.Computed != nil {
			f.values[field.Name] = field.Computed(f.values)
		}
	}

	// Validate
	f.validateField(name)

	if f.onChange != nil {
		f.onChange(name, value)
	}
}

// Errors returns all validation errors.
func (f *Form) Errors() map[string]error {
	return f.errors
}

// Error returns the error for a specific field.
func (f *Form) Error(name string) error {
	return f.errors[name]
}

// Valid reports whether the form has no errors.
func (f *Form) Valid() bool {
	return len(f.errors) == 0
}

// Submit validates and submits the form.
func (f *Form) Submit() mofu.Cmd {
	f.validateAll()
	if !f.Valid() {
		return nil
	}
	if f.onSubmit != nil {
		return f.onSubmit(f.values)
	}
	return nil
}

// FocusNext moves focus to the next visible field.
func (f *Form) FocusNext() {
	start := f.focus
	for {
		f.focus = (f.focus + 1) % len(f.fields)
		if f.fields[f.focus].Visible == nil || f.fields[f.focus].Visible(f.values) {
			return
		}
		if f.focus == start {
			return
		}
	}
}

// FocusPrev moves focus to the previous visible field.
func (f *Form) FocusPrev() {
	start := f.focus
	for {
		f.focus--
		if f.focus < 0 {
			f.focus = len(f.fields) - 1
		}
		if f.fields[f.focus].Visible == nil || f.fields[f.focus].Visible(f.values) {
			return
		}
		if f.focus == start {
			return
		}
	}
}

// CurrentField returns the currently focused field.
func (f *Form) CurrentField() *Field {
	if f.focus >= 0 && f.focus < len(f.fields) {
		return f.fields[f.focus]
	}
	return nil
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func (f *Form) validateField(name string) {
	for _, field := range f.fields {
		if field.Name == name {
			if field.Validator != nil {
				if err := field.Validator(f.values[name]); err != nil {
					f.errors[name] = err
					return
				}
			}
			if field.Required {
				val := f.values[name]
				if val == nil {
					f.errors[name] = fmt.Errorf("%s is required", field.Label)
					return
				}
				if s, ok := val.(string); ok && s == "" {
					f.errors[name] = fmt.Errorf("%s is required", field.Label)
					return
				}
			}
			delete(f.errors, name)
			return
		}
	}
}

func (f *Form) validateAll() {
	f.errors = make(map[string]error)
	for _, field := range f.fields {
		if field.Visible != nil && !field.Visible(f.values) {
			continue
		}
		f.validateField(field.Name)
	}
}

// ---------------------------------------------------------------------------
// Validation Helpers
// ---------------------------------------------------------------------------

func validateEmail(v any) error {
	s, ok := v.(string)
	if !ok {
		return nil
	}
	if s == "" {
		return nil
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(s) {
		return fmt.Errorf("invalid email address")
	}
	return nil
}

func validateMinLength(min int) func(any) error {
	return func(v any) error {
		s, ok := v.(string)
		if !ok {
			return nil
		}
		if len(strings.TrimSpace(s)) < min {
			return fmt.Errorf("must be at least %d characters", min)
		}
		return nil
	}
}

func validateMaxLength(max int) func(any) error {
	return func(v any) error {
		s, ok := v.(string)
		if !ok {
			return nil
		}
		if len(s) > max {
			return fmt.Errorf("must be at most %d characters", max)
		}
		return nil
	}
}

// ValidateMinValue validates that a number is at least min.
func ValidateMinValue(min float64) func(any) error {
	return func(v any) error {
		switch n := v.(type) {
		case int:
			if float64(n) < min {
				return fmt.Errorf("must be at least %.0f", min)
			}
		case float64:
			if n < min {
				return fmt.Errorf("must be at least %.2f", min)
			}
		}
		return nil
	}
}

// ValidateMaxValue validates that a number is at most max.
func ValidateMaxValue(max float64) func(any) error {
	return func(v any) error {
		switch n := v.(type) {
		case int:
			if float64(n) > max {
				return fmt.Errorf("must be at most %.0f", max)
			}
		case float64:
			if n > max {
				return fmt.Errorf("must be at most %.2f", max)
			}
		}
		return nil
	}
}

// ValidateRange validates that a number is within [min, max].
func ValidateRange(min, max float64) func(any) error {
	return func(v any) error {
		switch n := v.(type) {
		case int:
			if float64(n) < min || float64(n) > max {
				return fmt.Errorf("must be between %.0f and %.0f", min, max)
			}
		case float64:
			if n < min || n > max {
				return fmt.Errorf("must be between %.2f and %.2f", min, max)
			}
		}
		return nil
	}
}

// ValidateURL validates URL format.
func ValidateURL(v any) error {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return fmt.Errorf("must be a valid URL")
	}
	return nil
}

// ValidatePhone validates phone number format.
func ValidatePhone(v any) error {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	// Simple phone validation: digits, spaces, dashes, plus
	for _, c := range s {
		if !(c >= '0' && c <= '9') && c != ' ' && c != '-' && c != '+' && c != '(' && c != ')' {
			return fmt.Errorf("must be a valid phone number")
		}
	}
	return nil
}

// ValidateMatch validates that two fields match.
func ValidateMatch(field1, field2 string, values map[string]any) func(any) error {
	return func(v any) error {
		v1, ok1 := values[field1]
		v2, ok2 := values[field2]
		if ok1 && ok2 && fmt.Sprintf("%v", v1) != fmt.Sprintf("%v", v2) {
			return fmt.Errorf("%s must match %s", field1, field2)
		}
		return nil
	}
}

// ValidateOneOf validates that value is one of the allowed values.
func ValidateOneOf(allowed ...string) func(any) error {
	return func(v any) error {
		s, ok := v.(string)
		if !ok {
			return nil
		}
		for _, a := range allowed {
			if s == a {
				return nil
			}
		}
		return fmt.Errorf("must be one of: %s", strings.Join(allowed, ", "))
	}
}

// ValidateCustom creates a custom validator with a message.
func ValidateCustom(fn func(any) bool, message string) func(any) error {
	return func(v any) error {
		if !fn(v) {
			return fmt.Errorf("%s", message)
		}
		return nil
	}
}
