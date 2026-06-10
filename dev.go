package mofu

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type DevMode struct {
	watchDir  string
	buildCmd  string
	runArgs   []string
	lastBuild time.Time
	proc      *os.Process
}

func NewDevMode(dir string) *DevMode {
	return &DevMode{
		watchDir: dir,
		buildCmd: "go build -o .",
	}
}

func (d *DevMode) Watch() error {
	buildPath := filepath.Join(d.watchDir, "mofu-dev.exe")

	for {
		info, err := os.Stat(filepath.Join(d.watchDir, "go.mod"))
		if err != nil {
			return fmt.Errorf("no go.mod found: %w", err)
		}

		if info.ModTime().After(d.lastBuild) {
			d.lastBuild = info.ModTime()
		}

		needsBuild := false
		filepath.Walk(d.watchDir, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if filepath.Ext(path) == ".go" && fi.ModTime().After(d.lastBuild) {
				needsBuild = true
			}
			return nil
		})

		if needsBuild {
			if d.proc != nil {
				d.proc.Kill()
				d.proc.Wait()
				d.proc = nil
			}

			cmd := exec.Command("go", "build", "-o", buildPath, ".")
			cmd.Dir = d.watchDir
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "Build failed: %s\n%s\n", err, out)
			} else {
				d.lastBuild = time.Now()
				proc, err := os.StartProcess(buildPath, d.runArgs, &os.ProcAttr{
					Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Run failed: %s\n", err)
				} else {
					d.proc = proc
				}
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (d *DevMode) Stop() {
	if d.proc != nil {
		d.proc.Kill()
		d.proc = nil
	}
}
