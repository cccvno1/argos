package cli

import (
	"fmt"
	"io"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "init", "new", "validate", "index", "install-adapters", "mcp":
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
