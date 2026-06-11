# Forms Guide

Meow is MOFU's schema-driven form system. Define forms declaratively, get validation, computation, and conditional visibility for free.

## Basic Usage

```go
import "github.com/xanstomper/mofu/meow"

form := meow.NewForm(
    meow.Input("name", "Name").SetRequired(),
    meow.Input("email", "Email").Validate(meow.ValidateEmail),
    meow.Select("role", "Role", []string{"Admin", "User", "Guest"}),
    meow.Checkbox("agree", "I agree to terms"),
)

form.OnSubmit(func(values map[string]any) mofu.Cmd {
    fmt.Printf("Name: %s\n", values["name"])
    fmt.Printf("Email: %s\n", values["email"])
    return nil
})
```

## Field Types

| Type | Constructor | Use Case |
|------|-------------|----------|
| Text | `meow.Input(name, label)` | General text input |
| Email | `meow.Email(name, label)` | Email validation |
| Password | `meow.Password(name, label)` | Masked input |
| Number | `meow.Number(name, label)` | Numeric input |
| Select | `meow.Select(name, label, options)` | Dropdown |
| Checkbox | `meow.Checkbox(name, label)` | Boolean toggle |
| Textarea | `meow.Textarea(name, label)` | Multi-line text |
| Date | `meow.Date(name, label)` | Date input |

## Field Modifiers

```go
// Required field
meow.Input("name", "Name").SetRequired()

// Disabled field
meow.Input("email", "Email").SetDisabled()

// Custom placeholder
meow.Input("search", "Search").SetPlaceholder("Type to search...")

// Custom validation
meow.Input("age", "Age").Validate(func(v any) error {
    if n, ok := v.(int); ok && n < 0 {
        return fmt.Errorf("age must be positive")
    }
    return nil
})

// Conditional visibility
meow.Input("company", "Company").When(func(values map[string]any) bool {
    role, _ := values["role"].(string)
    return role == "business"
})

// Computed value
meow.Input("fullName", "Full Name").Compute(func(values map[string]any) any {
    first, _ := values["firstName"].(string)
    last, _ := values["lastName"].(string)
    return first + " " + last
})
```

## Built-in Validators

```go
meow.ValidateEmail    // Email format
meow.ValidateMinLength(3)  // Minimum length
meow.ValidateMaxLength(100) // Maximum length
```

## Form Operations

```go
// Get form values
values := form.Values()
name := form.Value("name")

// Set values
form.SetValue("name", "John")

// Check validity
if form.Valid() {
    form.Submit()
}

// Navigation
form.FocusNext()
form.FocusPrev()

// Get errors
errors := form.Errors()
nameError := form.Error("name")
```

## Complete Example

```go
type RegistrationForm struct {
    mofu.Minimal
    form *meow.Form
}

func NewRegistrationForm() *RegistrationForm {
    f := &RegistrationForm{}
    f.form = meow.NewForm(
        meow.Input("firstName", "First Name").SetRequired(),
        meow.Input("lastName", "Last Name").SetRequired(),
        meow.Email("email", "Email").SetRequired(),
        meow.Password("password", "Password").Validate(meow.ValidateMinLength(8)),
        meow.Select("role", "Role", []string{"User", "Admin"}),
        meow.Input("company", "Company").When(func(values map[string]any) bool {
            role, _ := values["role"].(string)
            return role == "admin"
        }),
        meow.Checkbox("terms", "I agree to Terms of Service").SetRequired(),
    )

    f.form.OnSubmit(func(values map[string]any) mofu.Cmd {
        fmt.Printf("Registering: %v\n", values)
        return mofu.QuitCmd()
    })

    return f
}

func (f *RegistrationForm) Render(ctx *mofu.RenderContext) {
    // Render form fields
}

func (f *RegistrationForm) HandleEvent(event mofu.Event) mofu.Cmd {
    // Route events to form
    return nil
}
```
