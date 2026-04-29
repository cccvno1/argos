package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"argos/internal/adapters"
	"argos/internal/index"
	"argos/internal/knowledge"
	"argos/internal/registry"
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
	case "validate":
		root, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get current directory: %v\n", err)
			return 1
		}
		reg, err := registry.Load(root)
		if err != nil {
			fmt.Fprintf(stderr, "load registry: %v\n", err)
			return 1
		}
		items, err := knowledge.LoadItems(root)
		if err != nil {
			fmt.Fprintf(stderr, "load knowledge items: %v\n", err)
			return 1
		}

		errorCount := 0
		for _, item := range items {
			for _, err := range knowledge.ValidateItem(item, reg) {
				fmt.Fprintln(stderr, err)
				errorCount++
			}
		}
		if errorCount > 0 {
			fmt.Fprintf(stderr, "validation failed with %d error(s)\n", errorCount)
			return 1
		}

		fmt.Fprintf(stdout, "validated %d knowledge item(s)\n", len(items))
		return 0
	case "index":
		root, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get current directory: %v\n", err)
			return 1
		}
		items, err := knowledge.LoadItems(root)
		if err != nil {
			fmt.Fprintf(stderr, "load knowledge items: %v\n", err)
			return 1
		}
		if err := index.Rebuild(filepath.Join(root, "argos", "index.db"), items); err != nil {
			fmt.Fprintf(stderr, "rebuild index: %v\n", err)
			return 1
		}

		fmt.Fprintf(stdout, "indexed %d knowledge item(s)\n", len(items))
		return 0
	case "install-adapters":
		root, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get current directory: %v\n", err)
			return 1
		}
		reg, err := registry.Load(root)
		if err != nil {
			fmt.Fprintf(stderr, "load registry: %v\n", err)
			return 1
		}
		if err := adapters.Install(root, reg.Projects); err != nil {
			fmt.Fprintf(stderr, "install adapters: %v\n", err)
			return 1
		}

		fmt.Fprintf(stdout, "installed adapters for %d project(s)\n", len(reg.Projects))
		return 0
	case "new", "mcp":
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
