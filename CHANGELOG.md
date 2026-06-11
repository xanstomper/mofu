# Changelog

All notable changes to MOFU will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.1.0] - 2024-01-01

### Added
- Initial release
- Reactive state graph with O(1) dirty tracking
- Double-buffered diff renderer
- Synchronized Output (CSI 2026)
- Complete input parser (arrows, F-keys, Ctrl+key, Alt+key, mouse)
- Spring physics animations
- Theme system with semantic colors
- Layout engine (flex row/column)
- Terminal capability detection
- Design grammar primitives
- Widget library:
  - Text
  - Input
  - List
  - Button
  - ProgressBar
  - Tabs
  - Checkbox
  - Select
  - Modal
  - Table
  - Toast
  - Tooltip
  - Tree
  - Menu
  - Container
- Examples:
  - Counter
  - Dashboard
  - Chat
  - File Manager
  - Form
  - Settings
  - Log Viewer
- 86 tests passing
- GoDoc documentation
- CI/CD with GitHub Actions
- CONTRIBUTING.md
- CODE_OF_CONDUCT.md
- SECURITY.md
- LICENSE (MIT)

### Fixed
- Goroutine leak in DataNode.Set
- Non-deterministic HashState
- ANSI color emission bug
- AlignStretch style mutation
- EaseOutElastic math
- Layout cache typo
- CollectDirty double-lock
- Node field shadowing

### Removed
- Dead scheduler code
- Unused particle system
- Unused special effects
- Dead OutputChannel implementations
