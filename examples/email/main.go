package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xanstomper/mofu"
)

type Email struct {
	From    string
	Subject string
	Body    string
	Date    time.Time
	Read    bool
	Starred bool
	Folder  string
}

type EmailClient struct {
	mofu.Minimal
	folders      []string
	currentFolder int
	emails       map[string][]Email
	selected     int
	showPreview  bool
	width        int
	height       int
	composing    bool
	composeTo    string
	composeSubj  string
	composeBody  string
	composeField int
}

func NewEmailClient() *EmailClient {
	c := &EmailClient{
		folders:  []string{"Inbox", "Sent", "Drafts", "Spam", "Trash"},
		emails:   make(map[string][]Email),
	}
	c.emails["Inbox"] = []Email{
		{From: "alice@dev.io", Subject: "PR #42 needs review", Body: "Hey, can you take a look at the diff? The caching layer changes are critical.\n\nI've added benchmarks showing 3x improvement on cache hits.", Date: time.Now().Add(-30 * time.Minute), Folder: "Inbox"},
		{From: "ci@github.com", Subject: "Build #1247 passed", Body: "All 847 tests passed.\nCoverage: 94.2%\nLint: clean", Date: time.Now().Add(-2 * time.Hour), Folder: "Inbox"},
		{From: "bob@ops.co", Subject: "Incident: DB latency spike", Body: "P1 incident opened.\nConnection pool exhausted at 14:23 UTC.\nFailover to replica completed at 14:28 UTC.", Date: time.Now().Add(-5 * time.Hour), Folder: "Inbox", Read: true},
		{From: "newsletter@go.dev", Subject: "Go 1.24 released", Body: "Major changes:\n- Range-over-func improvements\n- New unique package\n- Weak pointers in runtime", Date: time.Now().Add(-24 * time.Hour), Folder: "Inbox", Read: true},
		{From: "security@aws", Subject: "Rotate access keys", Body: "Your IAM access key will expire in 7 days.\nPlease rotate before 2026-06-18.", Date: time.Now().Add(-48 * time.Hour), Folder: "Inbox", Starred: true},
	}
	c.emails["Sent"] = []Email{
		{From: "me", Subject: "RE: PR #42 needs review", Body: "Reviewed. Two minor nits but overall looks good.", Date: time.Now().Add(-20 * time.Minute), Folder: "Sent"},
	}
	return c
}

func (c *EmailClient) currentFolderName() string {
	return c.folders[c.currentFolder]
}

func (c *EmailClient) currentEmails() []Email {
	return c.emails[c.currentFolderName()]
}

func (c *EmailClient) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	c.width = r.Width
	c.height = r.Height

	leftW := 18
	midW := r.Width/2 - leftW
	rightW := r.Width - leftW - midW

	// Folder pane
	folderStyle := mofu.DefaultStyle().Fg(mofu.Hex("89b4fa")).WithAttrs(mofu.AttrBold)
	ctx.Renderer.WriteString(" Mailboxes", r.X, r.Y, folderStyle.Foreground, folderStyle.Background, folderStyle.Attrs)

	for i, folder := range c.folders {
		y := r.Y + 1 + i
		count := len(c.emails[folder])
		unread := 0
		for _, e := range c.emails[folder] {
			if !e.Read {
				unread++
			}
		}

		style := mofu.DefaultStyle()
		prefix := "  "
		if i == c.currentFolder {
			prefix = "▸ "
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4")).WithAttrs(mofu.AttrBold)
		}

		label := fmt.Sprintf("%s%s (%d)", prefix, folder, count)
		if unread > 0 {
			label += fmt.Sprintf(" [%d]", unread)
		}
		ctx.Renderer.WriteString(label, r.X, y, style.Foreground, style.Background, style.Attrs)
	}

	// Separator
	ctx.Renderer.WriteString("│", r.X+leftW-1, r.Y, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Email list
	emails := c.currentEmails()
	ctx.Renderer.WriteString(fmt.Sprintf(" %s (%d)", c.currentFolderName(), len(emails)), r.X+leftW, r.Y, mofu.Hex("ff69b4"), mofu.ColorBlack, mofu.AttrBold)
	ctx.Renderer.WriteString(strings.Repeat("─", midW-1), r.X+leftW, r.Y+1, mofu.Hex("444444"), mofu.ColorBlack, 0)

	for i, email := range emails {
		y := r.Y + 2 + i
		if y >= r.Y+r.Height-1 {
			break
		}

		style := mofu.DefaultStyle()
		icon := "●"
		if email.Read {
			icon = "○"
		}
		star := ""
		if email.Starred {
			star = "★"
		}

		subject := email.Subject
		if len(subject) > midW-16 {
			subject = subject[:midW-19] + "..."
		}

		line := fmt.Sprintf("%s %s%-12s %s", icon, star, email.From[:min(12, len(email.From))], subject)
		if len(line) > midW-1 {
			line = line[:midW-1]
		}

		if i == c.selected {
			ctx.Renderer.WriteString(strings.Repeat(" ", midW-1), r.X+leftW, y, mofu.ColorWhite, mofu.Hex("2a2a3e"), 0)
			style = mofu.DefaultStyle().Fg(mofu.Hex("ff69b4"))
		} else if !email.Read {
			style = mofu.DefaultStyle().WithAttrs(mofu.AttrBold)
		}

		ctx.Renderer.WriteString(line, r.X+leftW, y, style.Foreground, style.Background, style.Attrs)
	}

	// Right separator
	ctx.Renderer.WriteString("│", r.X+leftW+midW-1, r.Y, mofu.Hex("444444"), mofu.ColorBlack, 0)

	// Preview pane
	if c.showPreview && c.selected < len(emails) {
		email := emails[c.selected]
		ctx.Renderer.WriteString(" Preview", r.X+leftW+midW, r.Y, mofu.Hex("89b4fa"), mofu.ColorBlack, mofu.AttrBold)

		ctx.Renderer.WriteString(fmt.Sprintf(" From: %s", email.From), r.X+leftW+midW, r.Y+2, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		ctx.Renderer.WriteString(fmt.Sprintf(" Subj: %s", email.Subject), r.X+leftW+midW, r.Y+3, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		ctx.Renderer.WriteString(fmt.Sprintf(" Date: %s", email.Date.Format("Jan 2 15:04")), r.X+leftW+midW, r.Y+4, mofu.Hex("6c7086"), mofu.ColorBlack, 0)
		ctx.Renderer.WriteString(strings.Repeat("─", rightW-1), r.X+leftW+midW, r.Y+5, mofu.Hex("444444"), mofu.ColorBlack, 0)

		lines := strings.Split(email.Body, "\n")
		for j, line := range lines {
			y := r.Y + 6 + j
			if y >= r.Y+r.Height-1 {
				break
			}
			if len(line) > rightW-2 {
				line = line[:rightW-5] + "..."
			}
			ctx.Renderer.WriteString(line, r.X+leftW+midW, y, mofu.Hex("cdd6f4"), mofu.ColorBlack, 0)
		}
	} else {
		ctx.Renderer.WriteString(" Press Enter to preview", r.X+leftW+midW, r.Y+r.Height/2, mofu.Hex("585b70"), mofu.ColorBlack, 0)
	}

	// Status bar
	status := " q:quit ←→:folder ↑↓:email Enter:preview r:read s:star n:new d:delete"
	if len(status) > r.Width {
		status = status[:r.Width]
	}
	ctx.Renderer.WriteString(status, r.X, r.Y+r.Height-1, mofu.Hex("585b70"), mofu.Hex("1e1e2e"), 0)
}

func (c *EmailClient) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke := event.Data.(mofu.KeyEvent)
	emails := c.currentEmails()

	switch {
	case ke.Key == mofu.KeyEsc || (len(ke.Runes) > 0 && ke.Runes[0] == 'q'):
		return mofu.QuitCmd()

	case ke.Key == mofu.KeyLeft || (len(ke.Runes) > 0 && ke.Runes[0] == 'h'):
		if c.currentFolder > 0 {
			c.currentFolder--
			c.selected = 0
			c.showPreview = false
		}

	case ke.Key == mofu.KeyRight || (len(ke.Runes) > 0 && ke.Runes[0] == 'l'):
		if c.currentFolder < len(c.folders)-1 {
			c.currentFolder++
			c.selected = 0
			c.showPreview = false
		}

	case ke.Key == mofu.KeyDown || (len(ke.Runes) > 0 && ke.Runes[0] == 'j'):
		if c.selected < len(emails)-1 {
			c.selected++
		}

	case ke.Key == mofu.KeyUp || (len(ke.Runes) > 0 && ke.Runes[0] == 'k'):
		if c.selected > 0 {
			c.selected--
		}

	case ke.Key == mofu.KeyEnter:
		c.showPreview = !c.showPreview
		if c.showPreview && c.selected < len(emails) {
			emails[c.selected].Read = true
		}

	case (len(ke.Runes) > 0 && ke.Runes[0] == 'r') && c.selected < len(emails):
		emails[c.selected].Read = !emails[c.selected].Read

	case (len(ke.Runes) > 0 && ke.Runes[0] == 's') && c.selected < len(emails):
		emails[c.selected].Starred = !emails[c.selected].Starred
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	app := NewEmailClient()
	if err := mofu.Run(app); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
