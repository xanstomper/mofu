package uicompile

import (
	"github.com/anomalyco/mofu"
	"github.com/anomalyco/mofu/state"
)

type Compiler struct {
	graph    *state.Graph
	bindings map[string]state.NodeID
}

func NewCompiler(graph *state.Graph) *Compiler {
	return &Compiler{
		graph:    graph,
		bindings: make(map[string]state.NodeID),
	}
}

func (c *Compiler) Bind(name string, node state.StateNode) {
	c.bindings[name] = node.ID()
}

func (c *Compiler) Compile(dsl *UIDSL) *UINode {
	return c.compileNode(dsl.Root)
}

func (c *Compiler) compileNode(dsl *UINode) *UINode {
	node := &UINode{
		Type:    dsl.Type,
		ID:      dsl.ID,
		Style:   dsl.Style,
		Bounds:  dsl.Bounds,
		Content: dsl.Content,
		Visible: true,
		Dirty:   true,
	}

	if dsl.StateRef != "" {
		if nodeID, ok := c.bindings[dsl.StateRef]; ok {
			n := c.graph.Get(nodeID)
			if n != nil {
				node.Content = toString(n.Value())
			}
		}
	}

	for _, child := range dsl.Children {
		node.Children = append(node.Children, c.compileNode(child))
	}

	return node
}

func (c *Compiler) Materialize(node *UINode) mofu.Node {
	switch node.Type {
	case NodeText:
		n := &mofu.TextNode{}
		n.SetBounds(node.Bounds)
		return n
	case NodeBox:
		n := &mofu.BoxNode{}
		n.SetBounds(node.Bounds)
		for _, child := range node.Children {
			n.AddChild(c.Materialize(child))
		}
		return n
	default:
		n := &mofu.BoxNode{}
		n.SetBounds(node.Bounds)
		return n
	}
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
