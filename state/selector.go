package state

type Selector struct {
	*Computed
}

func SelectAtom(a *Atom) *Selector {
	return &Selector{
		Computed: NewComputed([]StateNode{a}, func(deps []any) any {
			return deps[0]
		}),
	}
}

func Map(atom *Atom, fn func(v any) any) *Selector {
	return &Selector{
		Computed: NewComputed([]StateNode{atom}, func(deps []any) any {
			return fn(deps[0])
		}),
	}
}

func Combine(atoms []*Atom, fn func(vals []any) any) *Selector {
	nodes := make([]StateNode, len(atoms))
	for i, a := range atoms {
		nodes[i] = a
	}
	return &Selector{
		Computed: NewComputed(nodes, func(deps []any) any {
			return fn(deps)
		}),
	}
}

func (s *Selector) Subscribe(fn func(any)) {
	s.OnChange(func(ev ChangeEvent) {
		fn(ev.Value)
	})
}
