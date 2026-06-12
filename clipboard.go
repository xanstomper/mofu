package mofu

import (
	"os/exec"
	"runtime"
	"strings"
)

type ClipboardMsg struct {
	Content string
}

func ReadClipboard() Cmd {
	return func() Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("powershell", "-command", "Get-Clipboard")
		case "darwin":
			cmd = exec.Command("pbpaste")
		default:
			cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
		}
		out, err := cmd.Output()
		if err != nil {
			return ClipboardMsg{Content: ""}
		}
		return ClipboardMsg{Content: strings.TrimRight(string(out), "\r\n")}
	}
}

func WriteClipboard(text string) Cmd {
	return func() Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("powershell", "-command", "Set-Clipboard", "-InputObject", "-")
		case "darwin":
			cmd = exec.Command("pbcopy")
		default:
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
		cmd.Stdin = strings.NewReader(text)
		cmd.Run()
		return nil
	}
}
