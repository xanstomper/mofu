# Styling Guide

Cuddles is MOFU's semantic styling engine. Instead of specifying colors directly, you specify meaning.

## Why Semantic Styling?

```go
// ❌ Bad: Visual styling (what most frameworks do)
style := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))

// ✅ Good: Semantic styling (MOFU approach)
style := theme.Style(cuddles.Primary)
```

When you change themes, all "primary" elements update automatically.

## Usage

```go
import "github.com/xanstomper/mofu/cuddles"

// Get current theme
theme := cuddles.Mochi()

// Use semantic tokens
primaryStyle := theme.Style(cuddles.Primary)
errorStyle := theme.Style(cuddles.Error)
successStyle := theme.Style(cuddles.Success)
```

## Semantic Tokens

| Token | Use Case |
|-------|----------|
| Primary | Main brand color |
| Secondary | Supporting color |
| Accent | Highlight color |
| Success | Positive feedback |
| Warning | Caution |
| Error | Negative feedback |
| Info | Informational |
| Muted | De-emphasized |
| Text | Primary text |
| TextDim | Secondary text |
| Background | App background |
| Surface | Card/panel background |
| Border | Borders and dividers |

## Built-in Themes

```go
// Mochi (default)
theme := cuddles.Mochi()

// Catppuccin Mocha
theme := cuddles.Catppuccin()

// Tokyo Night
theme := cuddles.TokyoNight()
```

## Theme Switching

```go
manager := cuddles.NewManager(cuddles.Mochi())

// Register additional themes
manager.Register(cuddles.Catppuccin())
manager.Register(cuddles.TokyoNight())

// Apply a theme
manager.Apply("catppuccin")

// Listen for changes
manager.OnChange(func(old, new *cuddles.Theme) {
    fmt.Printf("Theme changed: %s → %s\n", old.Name, new.Name)
})
```

## Creating Custom Themes

```go
theme := &cuddles.Theme{
    Name: "my-theme",
    Colors: map[cuddles.Semantic]mofu.Color{
        cuddles.Primary:   mofu.Hex("#ff69b4"),
        cuddles.Secondary: mufu.Hex("#9b59b6"),
        cuddles.Error:     mofu.Hex("#ff3355"),
        // ... more colors
    },
    Density: cuddles.DensityNormal,
    Motion:  cuddles.DefaultMotion(),
}
```

## Color Utilities

```go
// Blend two colors
mixed := mofu.Blend(color1, color2, 0.5)

// Darken a color
darker := mofu.Darken(color, 0.2)

// Lighten a color
lighter := mofu.Lighten(color, 0.2)

// Get contrast-appropriate text color
textColor := mofu.TextColorForBackground(bgColor)

// Check WCAG contrast
mofu.MeetsWCAG(fg, bg, "AA")  // true if passes AA
```

## Density Profiles

```go
// Compact: tight spacing
theme.Density = cuddles.DensityCompact

// Normal: default spacing
theme.Density = cuddles.DensityNormal

// Comfortable: loose spacing
theme.Density = cuddles.DensityComfortable
```

## Motion Profiles

```go
theme.Motion = cuddles.MotionProfile{
    Speed:     1.0,  // 0.5=slow, 1.0=normal, 2.0=fast
    Elasticity: 1.0, // 0.0=rigid, 1.0=normal, 2.0=bouncy
    Duration:  300,  // base duration in ms
}
```
