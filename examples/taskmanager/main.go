package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xanstomper/mofu"
)

// TaskManager — a real task management application

type Task struct {
	ID        int
	Title     string
	Status    string // "todo", "doing", "done"
	Priority  int    // 1=low, 2=medium, 3=high
	CreatedAt time.Time
}

type TaskManager struct {
	mofu.Minimal
	tasks      []Task
	selected   int
	view       int // 0=all, 1=todo, 2=doing, 3=done
	width      int
	height     int
	nextID     int
	sortBy     string
	filterText string
}

func NewTaskManager() *TaskManager {
	tm := &TaskManager{
		nextID: 1,
	}
	// Add sample tasks
	tm.addTask("Design new UI", "doing", 3)
	tm.addTask("Write documentation", "todo", 2)
	tm.addTask("Fix login bug", "todo", 3)
	tm.addTask("Update dependencies", "doing", 1)
	tm.addTask("Write tests", "todo", 2)
	tm.addTask("Deploy to production", "done", 3)
	return tm
}

func (tm *TaskManager) addTask(title, status string, priority int) {
	tm.tasks = append(tm.tasks, Task{
		ID:        tm.nextID,
		Title:     title,
		Status:    status,
		Priority:  priority,
		CreatedAt: time.Now(),
	})
	tm.nextID++
}

func (tm *TaskManager) filteredTasks() []Task {
	var filtered []Task
	for _, task := range tm.tasks {
		// Filter by view
		switch tm.view {
		case 1:
			if task.Status != "todo" {
				continue
			}
		case 2:
			if task.Status != "doing" {
				continue
			}
		case 3:
			if task.Status != "done" {
				continue
			}
		}

		// Filter by search
		if tm.filterText != "" && !strings.Contains(strings.ToLower(task.Title), strings.ToLower(tm.filterText)) {
			continue
		}

		filtered = append(filtered, task)
	}

	// Sort
	switch tm.sortBy {
	case "priority":
		for i := 0; i < len(filtered); i++ {
			for j := i + 1; j < len(filtered); j++ {
				if filtered[i].Priority < filtered[j].Priority {
					filtered[i], filtered[j] = filtered[j], filtered[i]
				}
			}
		}
	case "date":
		for i := 0; i < len(filtered); i++ {
			for j := i + 1; j < len(filtered); j++ {
				if filtered[i].CreatedAt.Before(filtered[j].CreatedAt) {
					filtered[i], filtered[j] = filtered[j], filtered[i]
				}
			}
		}
	}

	return filtered
}

func (tm *TaskManager) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	tm.width = r.Width
	tm.height = r.Height

	// Title
	titleStyle := mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Task Manager", r.X, r.Y, titleStyle.Foreground, titleStyle.Background, titleStyle.Attrs)

	sep := strings.Repeat("─", r.Width-2)
	ctx.Renderer.WriteString(sep, r.X+1, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// View tabs
	views := []string{" All ", " Todo ", " Doing ", " Done "}
	for i, v := range views {
		style := mofu.DefaultStyle().Fg(mofu.Hex("666666"))
		if i == tm.view {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
		}
		ctx.Renderer.WriteString(v, r.X+2+i*10, r.Y+1, style.Foreground, style.Background, style.Attrs)
	}

	// Task list
	tasks := tm.filteredTasks()
	y := r.Y + 3

	// Header
	headerStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(fmt.Sprintf(" %-4s %-30s %-10s %-8s", "ID", "TITLE", "STATUS", "PRIORITY"), r.X+1, y, headerStyle.Foreground, headerStyle.Background, headerStyle.Attrs)
	y++

	ctx.Renderer.WriteString(strings.Repeat("─", r.Width-2), r.X+1, y, mofu.Hex("444444"), mofu.ColorBlack, 0)
	y++

	for i, task := range tasks {
		if y >= r.Y+r.Height-3 {
			break
		}

		style := mofu.DefaultStyle()
		if i == tm.selected {
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
			ctx.Renderer.WriteString(strings.Repeat(" ", r.Width-2), r.X+1, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
		}

		// Status color
		statusColor := mofu.Hex("666666")
		switch task.Status {
		case "todo":
			statusColor = mofu.Hex("f9e2af")
		case "doing":
			statusColor = mofu.Hex("89b4fa")
		case "done":
			statusColor = mofu.Hex("a6e3a1")
		}

		// Priority color
		priorityColor := mofu.Hex("666666")
		switch task.Priority {
		case 3:
			priorityColor = mofu.Hex("f38ba8")
		case 2:
			priorityColor = mofu.Hex("f9e2af")
		case 1:
			priorityColor = mofu.Hex("a6e3a1")
		}

		title := task.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}

		ctx.Renderer.WriteString(fmt.Sprintf(" %-4d ", task.ID), r.X+1, y, style.Foreground, style.Background, style.Attrs)
		ctx.Renderer.WriteString(title, r.X+6, y, style.Foreground, style.Background, style.Attrs)
		ctx.Renderer.WriteString(task.Status, r.X+37, y, statusColor, mofu.ColorBlack, 0)

		priorityStr := "Low"
		if task.Priority == 2 {
			priorityStr = "Med"
		} else if task.Priority == 3 {
			priorityStr = "High"
		}
		ctx.Renderer.WriteString(priorityStr, r.X+48, y, priorityColor, mofu.ColorBlack, 0)

		y++
	}

	// Stats
	total := len(tm.tasks)
	todo := 0
	doing := 0
	done := 0
	for _, t := range tm.tasks {
		switch t.Status {
		case "todo":
			todo++
		case "doing":
			doing++
		case "done":
			done++
		}
	}
	ctx.Renderer.WriteString(fmt.Sprintf(" Total: %d | Todo: %d | Doing: %d | Done: %d", total, todo, doing, done), r.X, r.Y+r.Height-1, mofu.Hex("666666"), mofu.ColorBlack, 0)
}

func (tm *TaskManager) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)

	tasks := tm.filteredTasks()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && (ke.Runes[0] == 'q' || ke.Runes[0] == 'Q')):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if tm.selected < len(tasks)-1 {
			tm.selected++
		}

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if tm.selected > 0 {
			tm.selected--
		}

	// View switching
	case len(ke.Runes) > 0 && ke.Runes[0] == '1':
		tm.view = 0
		tm.selected = 0
	case len(ke.Runes) > 0 && ke.Runes[0] == '2':
		tm.view = 1
		tm.selected = 0
	case len(ke.Runes) > 0 && ke.Runes[0] == '3':
		tm.view = 2
		tm.selected = 0
	case len(ke.Runes) > 0 && ke.Runes[0] == '4':
		tm.view = 3
		tm.selected = 0

	// Status change
	case len(ke.Runes) > 0 && ke.Runes[0] == 'd':
		if tm.selected >= 0 && tm.selected < len(tasks) {
			task := &tasks[tm.selected]
			for i := range tm.tasks {
				if tm.tasks[i].ID == task.ID {
					if tm.tasks[i].Status == "todo" {
						tm.tasks[i].Status = "doing"
					} else if tm.tasks[i].Status == "doing" {
						tm.tasks[i].Status = "done"
					} else {
						tm.tasks[i].Status = "todo"
					}
					break
				}
			}
		}

	// Delete
	case len(ke.Runes) > 0 && ke.Runes[0] == 'x':
		if tm.selected >= 0 && tm.selected < len(tasks) {
			task := tasks[tm.selected]
			for i := range tm.tasks {
				if tm.tasks[i].ID == task.ID {
					tm.tasks = append(tm.tasks[:i], tm.tasks[i+1:]...)
					if tm.selected >= len(tm.filteredTasks()) {
						tm.selected = len(tm.filteredTasks()) - 1
					}
					break
				}
			}
		}

	// Sort
	case len(ke.Runes) > 0 && ke.Runes[0] == 's':
		if tm.sortBy == "" {
			tm.sortBy = "priority"
		} else if tm.sortBy == "priority" {
			tm.sortBy = "date"
		} else {
			tm.sortBy = ""
		}
	}

	return nil
}

func main() {
	app := NewTaskManager()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
