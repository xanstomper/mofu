package uicompile

import "github.com/anomalyco/mofu"

type NodeType int

const (
	NodeRoot NodeType = iota
	NodeBox
	NodeText
	NodeStack
	NodeInput
	NodeList
	NodeTable
	NodeCanvas
	NodePane
	NodeCustom
)

type UINode struct {
	Type     NodeType
	ID       string
	Style    mofu.Style
	Bounds   mofu.Rect
	Children []*UINode
	Content  string
	StateRef string
	Widget   mofu.Node
	Visible  bool
	Dirty    bool
}

type UIDSL struct {
	Version  string                `json:"version"`
	Root     *UINode               `json:"root"`
	Bindings map[string]string     `json:"bindings,omitempty"`
	Styles   map[string]mofu.Style `json:"styles,omitempty"`
}

func NewUIRoot() *UINode {
	return &UINode{
		Type:    NodeRoot,
		Visible: true,
		Dirty:   true,
	}
}

func (n *UINode) Add(child *UINode) {
	n.Children = append(n.Children, child)
}

func (n *UINode) FindByID(id string) *UINode {
	if n.ID == id {
		return n
	}
	for _, c := range n.Children {
		if found := c.FindByID(id); found != nil {
			return found
		}
	}
	return nil
}

func (n *UINode) Flatten() []*UINode {
	var out []*UINode
	out = append(out, n)
	for _, c := range n.Children {
		out = append(out, c.Flatten()...)
	}
	return out
}
