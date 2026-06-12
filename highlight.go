package mofu

import (
	"regexp"
	"strings"
	"sync"
)

type HighlightStyle struct {
	Style      Style
	Regexp     *regexp.Regexp
	Pattern    string
}

type HighlightSet struct {
	mu         sync.Mutex
	styles     []HighlightStyle
	defaultFg  Color
}

func NewHighlightSet(defaultFg Color) *HighlightSet {
	return &HighlightSet{
		defaultFg: defaultFg,
	}
}

func (hs *HighlightSet) Add(pattern string, fg Color) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	re, err := regexp.Compile(pattern)
	if err != nil {
		return
	}
	hs.styles = append(hs.styles, HighlightStyle{
		Pattern: pattern,
		Style:   DefaultStyle().Fg(fg),
		Regexp:  re,
	})
}

func (hs *HighlightSet) Highlight(line string) string {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if len(hs.styles) == 0 {
		return DefaultStyle().Fg(hs.defaultFg).Apply(line)
	}

	type span struct {
		start, end int
		style      Style
	}
	var spans []span

	for _, h := range hs.styles {
		locs := h.Regexp.FindAllStringIndex(line, -1)
		for _, loc := range locs {
			spans = append(spans, span{loc[0], loc[1], h.Style})
		}
	}

	if len(spans) == 0 {
		return DefaultStyle().Fg(hs.defaultFg).Apply(line)
	}

	for i := 0; i < len(spans); i++ {
		for j := i + 1; j < len(spans); j++ {
			if spans[j].start < spans[i].start {
				spans[i], spans[j] = spans[j], spans[i]
			}
		}
	}

	var out strings.Builder
	pos := 0
	for _, s := range spans {
		if s.start > pos {
			out.WriteString(DefaultStyle().Fg(hs.defaultFg).Apply(line[pos:s.start]))
		}
		out.WriteString(s.style.Apply(line[s.start:s.end]))
		pos = s.end
	}
	if pos < len(line) {
		out.WriteString(DefaultStyle().Fg(hs.defaultFg).Apply(line[pos:]))
	}
	return out.String()
}
