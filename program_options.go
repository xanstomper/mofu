package mofu

import (
	"sync"
	"time"
)

func WithAltScreen() Option {
	return func(p *Program) { p.altScreen = true }
}

func WithMouseCellMotion() Option {
	return func(p *Program) { p.mouseCellMotion = true }
}

func WithMouseAllMotion() Option {
	return func(p *Program) { p.mouseAllMotion = true }
}

func WithBracketedPaste() Option {
	return func(p *Program) { p.bracketedPaste = true }
}

func WithBracketedPasteCancel() Option {
	return func(p *Program) { p.bracketedPasteCancel = true }
}

func WithSyncOutput() Option {
	return func(p *Program) { p.syncOutput = true }
}

func WithReportFocus() Option {
	return func(p *Program) { p.reportFocus = true }
}

func WithFilterPaste() Option {
	return func(p *Program) { p.filterPaste = true }
}

func WithMiddleware(mws ...EventMiddleware) Option {
	return func(p *Program) {
		p.middlewares = append(p.middlewares, mws...)
	}
}

func WithEventFilter(fn func(Event) Event) Option {
	return func(p *Program) { p.eventFilter = fn }
}

func WithStatusMessageLifetime(d time.Duration) Option {
	return func(p *Program) { p.statusMessageLifetime = d }
}

type StatusBar struct {
	mu        sync.Mutex
	message   string
	createdAt time.Time
	lifetime  time.Duration
}

func (sb *StatusBar) Set(msg string, lifetime time.Duration) {
	sb.mu.Lock()
	sb.message = msg
	sb.createdAt = time.Now()
	sb.lifetime = lifetime
	sb.mu.Unlock()
}

func (sb *StatusBar) Message() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.message == "" {
		return ""
	}
	if sb.lifetime > 0 && time.Since(sb.createdAt) > sb.lifetime {
		sb.message = ""
		return ""
	}
	return sb.message
}

func (sb *StatusBar) Clear() {
	sb.mu.Lock()
	sb.message = ""
	sb.mu.Unlock()
}


