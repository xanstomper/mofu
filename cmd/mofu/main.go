package main

import (
	"flag"
	"fmt"
	"os"

	mofu "github.com/xanstomper/mofu"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Mofu - A modern Go TUI framework

Usage:
  mofu new <name>     Create a new Mofu project
  mofu dev            Run the current Mofu project in dev mode (hot-reload)
  mofu build          Build the current Mofu project
  mofu ssh            Serve the app over SSH (default: :23234)
  mofu version        Print version information

Examples:
  mofu new myapp
  cd myapp && mofu dev
  mofu ssh --addr :23234 --host-key /path/to/key
`)
	}

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new":
		cmdNew(os.Args[2:])
	case "dev":
		cmdDev()
	case "build":
		cmdBuild()
	case "ssh":
		cmdSSH(os.Args[2:])
	case "version":
		cmdVersion()
	default:
		flag.Usage()
		os.Exit(1)
	}
}

func cmdNew(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: mofu new <name>")
		os.Exit(1)
	}
	name := args[0]
	fmt.Printf("Creating new Mofu project: %s\n", name)
	fmt.Println("  (scaffolding not yet implemented)")
}

func cmdDev() {
	fmt.Println("Running Mofu dev mode (hot-reload)")
	fmt.Println("  (not yet implemented)")
}

func cmdBuild() {
	fmt.Println("Building Mofu project")
	fmt.Println("  (not yet implemented)")
}

func cmdVersion() {
	fmt.Println("Mofu v0.1.0 - Mochi TUI Framework")
}

func cmdSSH(args []string) {
	fs := flag.NewFlagSet("ssh", flag.ExitOnError)
	addr := fs.String("addr", ":23234", "SSH listen address")
	hostKeyPath := fs.String("host-key", "", "Path to Ed25519/RSA host key")
	fs.Parse(args)

	var hostKey []byte
	if *hostKeyPath != "" {
		var err error
		hostKey, err = os.ReadFile(*hostKeyPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read host key: %v\n", err)
			os.Exit(1)
		}
	}

	server, err := mofu.NewSSHServer(mofu.SSHServerConfig{
		Addr:       *addr,
		HostKey:    hostKey,
		NewProgram: newProgramFromEnv,
		Middlewares: []mofu.Middleware{
			mofu.LoggingMiddleware(nil),
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create SSH server: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting SSH server on %s\n", *addr)
	if err := server.Serve(*addr); err != nil {
		fmt.Fprintf(os.Stderr, "SSH server error: %v\n", err)
		os.Exit(1)
	}
}

type sshDemo struct {
	mofu.BaseNode
	msg string
}

func (a *sshDemo) Render(ctx *mofu.RenderContext) {
	r := ctx.Bounds
	ctx.Renderer.WriteStyledString(a.msg, r.X, r.Y, *a.Style())
}

func (a *sshDemo) HandleEvent(event mofu.Event) mofu.Cmd {
	if event.Type != mofu.EventKeyPress {
		return nil
	}
	ke, ok := event.Data.(mofu.KeyEvent)
	if !ok {
		return nil
	}
	if len(ke.Runes) > 0 && (ke.Runes[0] == 'q' || ke.Runes[0] == 'Q') {
		return func() mofu.Msg {
			os.Exit(0)
			return nil
		}
	}
	a.msg = "Connected via SSH! Press 'q' to quit."
	return nil
}

func newProgramFromEnv(sess *mofu.SSHSession) *mofu.Program {
	node := &sshDemo{msg: "Welcome to MOFU over SSH! Press 'q' to quit."}
	return mofu.New(node,
		mofu.WithInput(sess),
		mofu.WithOutputWriter(sess),
		mofu.WithTheme(mofu.MochiTheme()),
	)
}
