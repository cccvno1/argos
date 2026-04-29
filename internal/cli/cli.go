package cli

import (
	"fmt"
	"io"
	"os"

	"argos/internal/workspace"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "init":
		root, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get current directory: %v\n", err)
			return 1
		}
		if err := workspace.Init(root); err != nil {
			fmt.Fprintf(stderr, "initialize workspace: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "initialized Argos workspace")
		return 0
	case "new", "validate", "index", "install-adapters", "mcp":
		fmt.Fprintf(stderr, "command %q is not implemented yet\n", args[0])
		return 1
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: argos <command> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init")
	fmt.Fprintln(w, "  new")
	fmt.Fprintln(w, "  validate")
	fmt.Fprintln(w, "  index")
	fmt.Fprintln(w, "  install-adapters")
	fmt.Fprintln(w, "  mcp")
}
