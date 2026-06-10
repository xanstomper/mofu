package mofu

import (
	"os/exec"
	"runtime"
	"syscall"
)

type Clipboard struct{}

func NewClipboard() *Clipboard {
	return &Clipboard{}
}

func (c *Clipboard) Copy(text string) error {
	switch runtime.GOOS {
	case "windows":
		return c.copyWindows(text)
	case "darwin":
		return c.copyDarwin(text)
	default:
		return c.copyLinux(text)
	}
}

func (c *Clipboard) Paste() (string, error) {
	switch runtime.GOOS {
	case "windows":
		return c.pasteWindows()
	case "darwin":
		return c.pasteDarwin()
	default:
		return c.pasteLinux()
	}
}

func (c *Clipboard) copyWindows(text string) error {
	return exec.Command("clip").Run()
}

func (c *Clipboard) pasteWindows() (string, error) {
	out, err := exec.Command("powershell", "-Command", "Get-Clipboard").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Clipboard) copyDarwin(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = nil
	return cmd.Run()
}

func (c *Clipboard) pasteDarwin() (string, error) {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Clipboard) copyLinux(text string) error {
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = nil
		return cmd.Run()
	}
	if _, err := exec.LookPath("wl-copy"); err == nil {
		return exec.Command("wl-copy").Run()
	}
	return nil
}

func (c *Clipboard) pasteLinux() (string, error) {
	if _, err := exec.LookPath("xclip"); err == nil {
		out, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output()
		if err != nil {
			return "", err
		}
		return string(out), nil
	}
	if _, err := exec.LookPath("wl-paste"); err == nil {
		out, err := exec.Command("wl-paste").Output()
		if err != nil {
			return "", err
		}
		return string(out), nil
	}
	return "", nil
}

func init() {
	runtime.GC()
	_ = syscall.StringToUTF16
}
