package gadgets_test

import (
	"testing"

	"github.com/xanstomper/mofu/gadgets"
)

func TestRealLiveTableAddRow(t *testing.T) {
	table := gadgets.NewRealLiveTable("test", []string{"Name", "Value"})
	table.AddRow([]string{"Alice", "100"})
	table.AddRow([]string{"Bob", "200"})

	if len(table.GetRows()) != 2 {
		t.Errorf("expected 2 rows, got %d", len(table.GetRows()))
	}
}

func TestRealLiveTableSort(t *testing.T) {
	table := gadgets.NewRealLiveTable("test", []string{"Name", "Value"})
	table.AddRow([]string{"Charlie", "300"})
	table.AddRow([]string{"Alice", "100"})
	table.AddRow([]string{"Bob", "200"})

	table.Sort(0) // Sort by name

	rows := table.GetRows()
	if rows[0][0] != "Alice" {
		t.Errorf("expected Alice first, got %s", rows[0][0])
	}
	if rows[1][0] != "Bob" {
		t.Errorf("expected Bob second, got %s", rows[1][0])
	}
}

func TestRealLiveTableFilter(t *testing.T) {
	table := gadgets.NewRealLiveTable("test", []string{"Name", "Value"})
	table.AddRow([]string{"Alice", "100"})
	table.AddRow([]string{"Bob", "200"})
	table.AddRow([]string{"Charlie", "300"})

	table.SetFilter("ali")

	// Check filtered count
	filtered := table.FilteredRows()
	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered row, got %d", len(filtered))
	}
}

func TestRealMetricBoard(t *testing.T) {
	board := gadgets.NewRealMetricBoard("metrics")
	board.Set("cpu", 23.5, "%")
	board.Set("memory", 4.2, "GB")

	if board.Get("cpu") != 23.5 {
		t.Errorf("expected 23.5, got %f", board.Get("cpu"))
	}
	if board.Get("memory") != 4.2 {
		t.Errorf("expected 4.2, got %f", board.Get("memory"))
	}
}

func TestRealCommandPalette(t *testing.T) {
	palette := gadgets.NewRealCommandPalette("palette")
	palette.AddCommand(gadgets.CommandItem{
		Name:     "Save",
		Shortcut: "Ctrl+S",
		Category: "File",
	})

	palette.Show()
	if !palette.IsVisible() {
		t.Error("expected visible")
	}

	palette.Search("sav")
	if len(palette.GetFiltered()) != 1 {
		t.Errorf("expected 1 result, got %d", len(palette.GetFiltered()))
	}

	palette.Hide()
	if palette.IsVisible() {
		t.Error("expected not visible")
	}
}

func TestRealLogStream(t *testing.T) {
	stream := gadgets.NewRealLogStream("logs")
	stream.Append("INFO: Server started")
	stream.Append("ERROR: Connection failed")
	stream.Append("INFO: Retrying...")

	if stream.Count() != 3 {
		t.Errorf("expected 3 lines, got %d", stream.Count())
	}

	stream.SetLevel("ERROR")
	filtered := stream.FilteredLines()
	if len(filtered) != 1 {
		t.Errorf("expected 1 error line, got %d", len(filtered))
	}
}
