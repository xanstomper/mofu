package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/xanstomper/mofu"
)

// CSVViewer — a CSV file viewer example

type CSVViewer struct {
	mofu.Minimal
	headers    []string
	rows       [][]string
	selected   int
	sortCol    int
	sortAsc    bool
	filter     string
	width      int
	height     int
	filePath   string
}

func NewCSVViewer(filePath string) *CSVViewer {
	v := &CSVViewer{filePath: filePath}
	v.loadData()
	return v
}

func (v *CSVViewer) loadData() {
	file, err := os.Open(v.filePath)
	if err != nil {
		v.headers = []string{"Error loading file"}
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		v.headers = []string{"Error parsing CSV"}
		return
	}

	if len(records) > 0 {
		v.headers = records[0]
		v.rows = records[1:]
	}
}

func (v *CSVViewer) filteredRows() [][]string {
	if v.filter == "" {
		return v.rows
	}

	var filtered [][]string
	for _, row := range v.rows {
		for _, cell := range row {
			if strings.Contains(strings.ToLower(cell), strings.ToLower(v.filter)) {
				filtered = append(filtered, row)
				break
			}
		}
	}
	return filtered
}

func (v *CSVViewer) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	v.width = r.Width
	v.height = r.Height

	// Title
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" CSV Viewer: "+v.filePath, r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Headers
	y := r.Y + 2
	headerStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
	header := ""
	for i, h := range v.headers {
		if i == v.sortCol {
			if v.sortAsc {
				header += fmt.Sprintf("%-15s ▲", h)
			} else {
				header += fmt.Sprintf("%-15s ▼", h)
			}
		} else {
			header += fmt.Sprintf("%-15s ", h)
		}
	}
	ctx.Renderer.WriteString(header, r.X+1, y, headerStyle.Foreground, headerStyle.Background, headerStyle.Attrs)
	y++

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	// Rows
	rows := v.filteredRows()
	for i := 0; i < len(rows) && y < r.Y+r.Height-3; i++ {
		row := rows[i]
		text := ""
		for j, cell := range row {
			if j >= len(v.headers) {
				break
			}
			if len(cell) > 15 {
				cell = cell[:12] + "..."
			}
			text += fmt.Sprintf("%-15s ", cell)
		}

		style := mofu.DefaultStyle()
		if i == v.selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		ctx.Renderer.WriteString(text, r.X+1, y, style.Foreground, style.Background, style.Attrs)
		y++
	}

	// Status
	total := len(v.rows)
	filtered := len(v.filteredRows())
	ctx.Renderer.WriteString(fmt.Sprintf(" %d/%d rows | Sort: %s | Filter: %s", filtered, total, v.headers[v.sortCol], v.filter), r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (v *CSVViewer) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	rows := v.filteredRows()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if v.selected < len(rows)-1 {
			v.selected++
		}

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if v.selected > 0 {
			v.selected--
		}

	case ke.Key == mofu.KeyHome:
		v.selected = 0

	case ke.Key == mofu.KeyEnd:
		v.selected = len(rows) - 1

	// Sort by column
	case len(ke.Runes) > 0 && ke.Runes[0] >= '1' && ke.Runes[0] <= '9':
		col := int(ke.Runes[0] - '1')
		if col < len(v.headers) {
			if col == v.sortCol {
				v.sortAsc = !v.sortAsc
			} else {
				v.sortCol = col
				v.sortAsc = true
			}
			v.sortRows()
		}

	// Filter
	case len(ke.Runes) > 0 && ke.Runes[0] == '/':
		v.filter = ""
		v.selected = 0

	case ke.Key == mofu.KeyBack && len(v.filter) > 0:
		v.filter = v.filter[:len(v.filter)-1]
		v.selected = 0

	case len(ke.Runes) > 0 && ke.Runes[0] >= 'a' && ke.Runes[0] <= 'z':
		v.filter += string(ke.Runes)
		v.selected = 0

	case len(ke.Runes) > 0 && ke.Runes[0] >= 'A' && ke.Runes[0] <= 'Z':
		v.filter += strings.ToLower(string(ke.Runes))
		v.selected = 0
	}

	return nil
}

func (v *CSVViewer) sortRows() {
	sort.SliceStable(v.rows, func(i, j int) bool {
		if v.sortCol >= len(v.rows[i]) || v.sortCol >= len(v.rows[j]) {
			return false
		}
		a, b := v.rows[i][v.sortCol], v.rows[j][v.sortCol]
		if v.sortAsc {
			return a < b
		}
		return a > b
	})
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: csvviewer <file.csv>")
		os.Exit(1)
	}

	app := NewCSVViewer(os.Args[1])
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
