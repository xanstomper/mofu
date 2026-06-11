package meow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/xanstomper/mofu"
)

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

type Field struct {
	Type        FieldType
	Name        string
	Label       string
	Value       any
	Placeholder string
	Required    bool
	Disabled    bool
	Options     []string
	Validator   func(any) error
	Visible     func(values map[string]any) bool
	Computed    func(values map[string]any) any
}

func Input(name, label string) *Field {
	return &Field{Type: FieldInput, Name: name, Label: label}
}

func Select(name, label string, options []string) *Field {
	return &Field{Type: FieldSelect, Name: name, Label: label, Options: options}
}

func Checkbox(name, label string) *Field {
	return &Field{Type: FieldCheckbox, Name: name, Label: label}
}

func Textarea(name, label string) *Field {
	return &Field{Type: FieldTextarea, Name: name, Label: label}
}

func Password(name, label string) *Field {
	return &Field{Type: FieldPassword, Name: name, Label: label}
}

func Number(name, label string) *Field {
	return &Field{Type: FieldNumber, Name: name, Label: label}
}

func Email(name, label string) *Field {
	return &Field{Type: FieldEmail, Name: name, Label: label, Validator: validateEmail}
}

func Date(name, label string) *Field {
	return &Field{Type: FieldDate, Name: name, Label: label}
}

func (f *Field) SetRequired() *Field   { f.Required = true; return f }
func (f *Field) SetDisabled() *Field   { f.Disabled = true; return f }
func (f *Field) SetPlaceholder(text string) *Field { f.Placeholder = text; return f }
func (f *Field) Validate(fn func(any) error) *Field { f.Validator = fn; return f }
func (f *Field) When(fn func(values map[string]any) bool) *Field { f.Visible = fn; return f }
func (f *Field) Compute(fn func(values map[string]any) any) *Field { f.Computed = fn; return f }

type Form struct {
	fields   []*Field
	values   map[string]any
	errors   map[string]error
	focus    int
	onChange func(name string, value any)
	onSubmit func(values map[string]any) mofu.Cmd
	dirty    bool
}

func NewForm(fields ...*Field) *Form {
	f := &Form{
		fields: fields,
		values: make(map[string]any),
		errors: make(map[string]error),
	}
	for _, field := range fields {
		if field.Computed == nil {
			f.values[field.Name] = field.Value
		}
	}
	return f
}

func (f *Form) OnChange(fn func(name string, value any)) *Form {
	f.onChange = fn
	return f
}

func (f *Form) OnSubmit(fn func(values map[string]any) mofu.Cmd) *Form {
	f.onSubmit = fn
	return f
}

func (f *Form) Values() map[string]any      { return f.values }
func (f *Form) Value(name string) any        { return f.values[name] }
func (f *Form) Errors() map[string]error     { return f.errors }
func (f *Form) Error(name string) error      { return f.errors[name] }
func (f *Form) Valid() bool                  { return len(f.errors) == 0 }
func (f *Form) CurrentField() *Field         { return f.fields[f.focus] }

func (f *Form) SetValue(name string, value any) {
	f.values[name] = value
	f.dirty = true
	for _, field := range f.fields {
		if field.Computed != nil {
			f.values[field.Name] = field.Computed(f.values)
		}
	}
	f.validateField(name)
	if f.onChange != nil {
		f.onChange(name, value)
	}
}

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

func validateEmail(v any) error {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(s) {
		return fmt.Errorf("invalid email address")
	}
	return nil
}

func ValidateMinLength(min int) func(any) error {
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

func ValidateMaxLength(max int) func(any) error {
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

func ValidatePhone(v any) error {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9') && c != ' ' && c != '-' && c != '+' && c != '(' && c != ')' {
			return fmt.Errorf("must be a valid phone number")
		}
	}
	return nil
}

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

func ValidateCustom(fn func(any) bool, message string) func(any) error {
	return func(v any) error {
		if !fn(v) {
			return fmt.Errorf("%s", message)
		}
		return nil
	}
}
