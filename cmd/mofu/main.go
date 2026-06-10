package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Mofu - A modern Go TUI framework

Usage:
  mofu new <name>     Create a new Mofu project
  mofu dev            Run the current Mofu project in dev mode (hot-reload)
  mofu build          Build the current Mofu project
  mofu version        Print version information

Examples:
  mofu new myapp
  cd myapp && mofu dev
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
