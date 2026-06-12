package mofu

import "time"

type EventMiddleware func(next EventFilter) EventFilter

type EventFilter func(ev Event) Event

func Chain(mws ...EventMiddleware) EventMiddleware {
	return func(next EventFilter) EventFilter {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

func ColorProfileMiddleware() EventMiddleware {
	return func(next EventFilter) EventFilter {
		return func(ev Event) Event {
			return next(ev)
		}
	}
}

func FPSMiddleware(fps int) EventMiddleware {
	return func(next EventFilter) EventFilter {
		var last time.Time
		return func(ev Event) Event {
			now := time.Now()
			if fps > 0 && now.Sub(last) < time.Second/time.Duration(fps) {
				return Event{Type: EventSystem, Data: nil}
			}
			last = now
			return next(ev)
		}
	}
}

func PasteFilterMiddleware() EventMiddleware {
	return func(next EventFilter) EventFilter {
		return func(ev Event) Event {
			if pe, ok := ev.Data.(PasteEvent); ok {
				pe.Content = sanitizePaste(pe.Content)
				ev.Data = pe
			}
			return next(ev)
		}
	}
}

func sanitizePaste(s string) string {
	var out []rune
	for _, r := range s {
		if r >= 32 || r == '\n' || r == '\t' || r == '\r' {
			out = append(out, r)
		}
	}
	return string(out)
}

func FocusMiddleware() EventMiddleware {
	return func(next EventFilter) EventFilter {
		return func(ev Event) Event {
			return next(ev)
		}
	}
}

func SequenceMiddleware(mws ...EventMiddleware) EventMiddleware {
	return Chain(mws...)
}
