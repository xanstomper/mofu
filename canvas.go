package mofu

type Canvas struct {
	Width, Height int
	cells         []CanvasCell
}

type CanvasCell struct {
	Char   rune
	Fg, Bg Color
}

func NewCanvas(w, h int) *Canvas {
	return &Canvas{
		Width:  w,
		Height: h,
		cells:  make([]CanvasCell, w*h),
	}
}

func (c *Canvas) Set(x, y int, ch rune, fg, bg Color) {
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height {
		return
	}
	c.cells[y*c.Width+x] = CanvasCell{Char: ch, Fg: fg, Bg: bg}
}

func (c *Canvas) Get(x, y int) CanvasCell {
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height {
		return CanvasCell{}
	}
	return c.cells[y*c.Width+x]
}

func (c *Canvas) Clear() {
	for i := range c.cells {
		c.cells[i] = CanvasCell{Char: ' '}
	}
}

func (c *Canvas) DrawLine(x0, y0, x1, y1 int, ch rune, fg, bg Color) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		c.Set(x0, y0, ch, fg, bg)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (c *Canvas) DrawRect(x, y, w, h int, ch rune, fg, bg Color) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			c.Set(x+dx, y+dy, ch, fg, bg)
		}
	}
}

func (c *Canvas) DrawRectBorder(x, y, w, h int, bs BorderStyle, fg, bg Color) {
	if w < 2 || h < 2 {
		return
	}
	x2, y2 := x+w-1, y+h-1
	c.Set(x, y, bs.TopLeft, fg, bg)
	c.Set(x2, y, bs.TopRight, fg, bg)
	c.Set(x, y2, bs.BottomLeft, fg, bg)
	c.Set(x2, y2, bs.BottomRight, fg, bg)
	for dx := 1; dx < w-1; dx++ {
		c.Set(x+dx, y, bs.Top, fg, bg)
		c.Set(x+dx, y2, bs.Bottom, fg, bg)
	}
	for dy := 1; dy < h-1; dy++ {
		c.Set(x, y+dy, bs.Left, fg, bg)
		c.Set(x2, y+dy, bs.Right, fg, bg)
	}
}

func (c *Canvas) DrawCircle(cx, cy, r int, ch rune, fg, bg Color) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r*r {
				c.Set(x, y, ch, fg, bg)
			}
		}
	}
}

func (c *Canvas) FillCircle(cx, cy, r int, fg, bg Color) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r*r {
				c.Set(x, y, ' ', fg, bg)
			}
		}
	}
}

func (c *Canvas) DrawEllipse(cx, cy, rx, ry int, ch rune, fg, bg Color) {
	for y := cy - ry; y <= cy+ry; y++ {
		for x := cx - rx; x <= cx+rx; x++ {
			dx := float64(x-cx) / float64(rx)
			dy := float64(y-cy) / float64(ry)
			if dx*dx+dy*dy <= 1.0 {
				c.Set(x, y, ch, fg, bg)
			}
		}
	}
}

func (c *Canvas) DrawText(x, y int, text string, fg, bg Color) {
	for i, ch := range text {
		c.Set(x+i, y, ch, fg, bg)
	}
}

func (c *Canvas) RenderBraille(x, y int, data [][]float64, min, max float64, scale ColorScale) {
	for by := 0; by < len(data); by++ {
		for bx := 0; bx < len(data[by]); bx++ {
			cx := x + bx/2
			cy := y + by/4
			if cx >= c.Width || cy >= c.Height {
				continue
			}

			di := by % 4
			dj := bx % 2
			dotIdx := di*2 + dj

			v := data[by][bx]
			normalized := (v - min) / (max - min)
			on := normalized > float64(by%4)/4.0

			cell := c.Get(cx, cy)
			dots := brailleDecode(cell.Char)
			if on {
				dots[dotIdx] = true
			}
			col := scale.At(normalized)
			c.Set(cx, cy, dots.encode(), col, Color{})
		}
	}
}

func (c *Canvas) RenderTo(r *Renderer, ox, oy int) {
	for y := 0; y < c.Height; y++ {
		for x := 0; x < c.Width; x++ {
			cell := c.cells[y*c.Width+x]
			r.front.Set(ox+x, oy+y, cell.Char, cell.Fg, cell.Bg, 0)
		}
	}
}

type CanvasNode struct {
	BaseNode
	Canvas *Canvas
}

func NewCanvasNode(w, h int) *CanvasNode {
	return &CanvasNode{
		Canvas: NewCanvas(w, h),
	}
}

func (n *CanvasNode) Render(ctx *RenderContext) {
	b := n.BaseNode.Bounds()
	n.Canvas.RenderTo(ctx.Renderer, b.X, b.Y)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
