package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"argos/internal/adapters"
	"argos/internal/index"
	"argos/internal/knowledge"
	"argos/internal/mcp"
	"argos/internal/query"
	"argos/internal/registry"
	"argos/internal/workspace"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	return run(args, os.Stdin, stdout, stderr)
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
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
		flags := flag.NewFlagSet("validate", flag.ContinueOnError)
		flags.SetOutput(stderr)
		includeInbox := flags.Bool("inbox", false, "validate inbox candidates")
		path := flags.String("path", "", "validate a single item or package path")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		if *includeInbox && strings.TrimSpace(*path) != "" {
			fmt.Fprintln(stderr, "validate: --inbox and --path cannot be used together")
			return 2
		}
		root, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get current directory: %v\n", err)
			return 1
		}
		items, err := loadAndValidateKnowledge(root, stderr, validationScope{Inbox: *includeInbox, Path: *path})
		if err != nil {
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
		items, err := loadAndValidateKnowledge(root, stderr, validationScope{})
		if err != nil {
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
	case "context":
		flags := flag.NewFlagSet("context", flag.ContinueOnError)
		flags.SetOutput(stderr)
		flags.Bool("json", false, "print JSON output")
		project := flags.String("project", "", "project id")
		phase := flags.String("phase", "planning", "workflow phase")
		task := flags.String("task", "", "task description")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}

		result := query.New(nil).Context(query.ContextRequest{
			Project: *project,
			Phase:   *phase,
			Task:    *task,
		})
		body, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "marshal context response: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(body))
		return 0
	case "mcp":
		root, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get current directory: %v\n", err)
			return 1
		}
		server, closeServer, _ := openMCPServer(root)
		defer closeServer()
		if err := server.Serve(stdin, stdout); err != nil {
			fmt.Fprintf(stderr, "serve mcp: %v\n", err)
			return 1
		}
		return 0
	case "promote":
		flags := flag.NewFlagSet("promote", flag.ContinueOnError)
		flags.SetOutput(stderr)
		path := flags.String("path", "", "candidate item or package path")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		if strings.TrimSpace(*path) == "" {
			fmt.Fprintln(stderr, "promote: --path is required")
			return 2
		}
		root, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "get current directory: %v\n", err)
			return 1
		}
		target, err := promoteCandidate(root, *path, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "promote: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "promoted %s\n", target)
		fmt.Fprintln(stdout, "run argos index to refresh query results")
		return 0
	case "new":
		fmt.Fprintf(stderr, "command %q is not implemented yet\n", args[0])
		return 1
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func openMCPServer(root string) (*mcp.Server, func(), bool) {
	dbPath := filepath.Join(root, "argos", "index.db")
	info, err := os.Stat(dbPath)
	if err != nil || !info.Mode().IsRegular() {
		return mcp.NewServer(query.New(nil)), func() {}, false
	}

	store, err := index.Open(dbPath)
	if err != nil {
		return mcp.NewServer(query.New(nil)), func() {}, false
	}
	if err := store.CheckSchema(); err != nil {
		_ = store.Close()
		return mcp.NewServer(query.New(nil)), func() {}, false
	}
	return mcp.NewServerWithStore(store), func() {
		_ = store.Close()
	}, true
}

type validationScope struct {
	Inbox bool
	Path  string
}

func loadAndValidateKnowledge(root string, stderr io.Writer, scope validationScope) ([]knowledge.Item, error) {
	reg, err := registry.Load(root)
	if err != nil {
		fmt.Fprintf(stderr, "load registry: %v\n", err)
		return nil, err
	}

	var items []knowledge.Item
	switch {
	case strings.TrimSpace(scope.Path) != "":
		items, err = knowledge.LoadPath(root, scope.Path)
	case scope.Inbox:
		items, err = knowledge.LoadInbox(root)
	default:
		items, err = knowledge.LoadOfficial(root)
	}
	if err != nil {
		fmt.Fprintf(stderr, "load knowledge items: %v\n", err)
		return nil, err
	}

	errs := knowledge.ValidateItems(items, reg)
	for _, err := range errs {
		fmt.Fprintln(stderr, err)
	}
	if len(errs) > 0 {
		err := fmt.Errorf("validation failed with %d error(s)", len(errs))
		fmt.Fprintln(stderr, err)
		return nil, err
	}
	return items, nil
}

func promoteCandidate(root string, relPath string, stderr io.Writer) (string, error) {
	clean := filepath.Clean(relPath)
	if filepath.IsAbs(relPath) || clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("%s: candidate path must be relative and inside workspace", relPath)
	}
	target, err := promotionTarget(clean)
	if err != nil {
		return "", err
	}
	if _, err := loadAndValidateKnowledge(root, stderr, validationScope{Path: clean}); err != nil {
		return "", err
	}
	targetAbs := filepath.Join(root, target)
	if _, err := os.Stat(targetAbs); err == nil {
		return "", fmt.Errorf("target already exists: %s", target)
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat target %s: %w", target, err)
	}
	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
		return "", fmt.Errorf("create target parent: %w", err)
	}
	if err := os.Rename(filepath.Join(root, clean), targetAbs); err != nil {
		return "", fmt.Errorf("move candidate: %w", err)
	}
	if _, err := loadAndValidateKnowledge(root, stderr, validationScope{}); err != nil {
		return "", fmt.Errorf("official validation failed after promotion: %w", err)
	}
	return target, nil
}

func promotionTarget(clean string) (string, error) {
	slash := filepath.ToSlash(clean)
	for _, mapping := range []struct {
		inbox    string
		official string
	}{
		{"knowledge/.inbox/packages/", "knowledge/packages/"},
		{"knowledge/.inbox/items/", "knowledge/items/"},
	} {
		if strings.HasPrefix(slash, mapping.inbox) {
			rest := strings.TrimPrefix(slash, mapping.inbox)
			if rest == "" || strings.Contains(rest, "../") {
				return "", fmt.Errorf("%s: invalid inbox candidate path", clean)
			}
			return filepath.FromSlash(mapping.official + rest), nil
		}
	}
	return "", fmt.Errorf("%s: candidate must be under knowledge/.inbox/items or knowledge/.inbox/packages", clean)
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
	fmt.Fprintln(w, "  context")
	fmt.Fprintln(w, "  mcp")
	fmt.Fprintln(w, "  promote")
}
