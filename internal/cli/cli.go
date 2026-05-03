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
	"argos/internal/author"
	"argos/internal/authoringdogfood"
	"argos/internal/dogfood"
	"argos/internal/index"
	"argos/internal/knowledge"
	"argos/internal/mcp"
	"argos/internal/query"
	"argos/internal/registry"
	"argos/internal/workspace"
)

const (
	defaultDogfoodCasesPath          = "testdata/discovery-golden/cases.json"
	defaultAuthoringDogfoodCasesPath = "testdata/authoring-golden/cases.json"
	defaultAuthoringDogfoodFixtures  = "testdata/authoring-golden/fixtures"
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
		jsonOut := flags.Bool("json", false, "print JSON output")
		project := flags.String("project", "", "project id")
		phase := flags.String("phase", "", "workflow phase")
		task := flags.String("task", "", "task description")
		var files multiValueFlag
		flags.Var(&files, "files", "file path relevant to the current task; may be repeated")
		if err := flags.Parse(args[1:]); err != nil {
			return 2
		}
		if !*jsonOut {
			fmt.Fprintln(stderr, "context: --json is required")
			return 2
		}
		if strings.TrimSpace(*project) == "" {
			fmt.Fprintln(stderr, "context: --project is required")
			return 2
		}
		if strings.TrimSpace(*phase) == "" {
			fmt.Fprintln(stderr, "context: --phase is required")
			return 2
		}
		if strings.TrimSpace(*task) == "" {
			fmt.Fprintln(stderr, "context: --task is required")
			return 2
		}

		result := query.New(nil).Context(query.ContextRequest{
			Project: *project,
			Phase:   *phase,
			Task:    *task,
			Files:   files,
		})
		return printJSON(stdout, stderr, result)
	case "author":
		return runAuthor(args[1:], stdout, stderr)
	case "knowledge":
		return runKnowledge(args[1:], stdout, stderr)
	case "dogfood":
		return runDogfood(args[1:], stdout, stderr)
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

func runAuthor(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "author: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "inspect":
		return runAuthorInspect(args[1:], stdout, stderr)
	case "verify":
		return runAuthorVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "author: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runAuthorInspect(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("author inspect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	project := flags.String("project", "", "project id")
	goal := flags.String("goal", "", "knowledge authoring goal")
	mode := flags.String("mode", "", "authoring mode")
	futureTask := flags.String("future-task", "", "future retrieval task")
	phase := flags.String("phase", "", "workflow phase")
	queryText := flags.String("query", "", "search query")
	candidatePath := flags.String("candidate-path", "", "candidate knowledge path")
	var files multiValueFlag
	var domains multiValueFlag
	var tags multiValueFlag
	flags.Var(&files, "files", "file path relevant to the current task; may be repeated")
	flags.Var(&domains, "domains", "domain relevant to the current task; may be repeated")
	flags.Var(&tags, "tags", "tag relevant to the current task; may be repeated")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "author inspect: --json is required")
		return 2
	}
	if strings.TrimSpace(*project) == "" {
		fmt.Fprintln(stderr, "author inspect: --project is required")
		return 2
	}
	if strings.TrimSpace(*goal) == "" {
		fmt.Fprintln(stderr, "author inspect: --goal is required")
		return 2
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "author inspect: get current directory: %v\n", err)
		return 1
	}
	result, err := author.Inspect(root, author.InspectRequest{
		Project:       *project,
		Goal:          *goal,
		Mode:          *mode,
		FutureTask:    *futureTask,
		Phase:         *phase,
		Query:         *queryText,
		Files:         files,
		Domains:       domains,
		Tags:          tags,
		CandidatePath: *candidatePath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "author inspect: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}

func runAuthorVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("author verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	proposalPath := flags.String("proposal", "", "proposal JSON path")
	candidatePath := flags.String("path", "", "candidate item or package path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "author verify: --json is required")
		return 2
	}
	if strings.TrimSpace(*proposalPath) == "" {
		fmt.Fprintln(stderr, "author verify: --proposal is required")
		return 2
	}
	if strings.TrimSpace(*candidatePath) == "" {
		fmt.Fprintln(stderr, "author verify: --path is required")
		return 2
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "author verify: get current directory: %v\n", err)
		return 1
	}
	result, err := author.Verify(root, author.VerifyRequest{
		ProposalPath:  *proposalPath,
		CandidatePath: *candidatePath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "author verify: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}

func runDogfood(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "dogfood: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "authoring":
		return runDogfoodAuthoring(args[1:], stdout, stderr)
	case "cases":
		return runDogfoodCases(args[1:], stdout, stderr)
	case "packet":
		return runDogfoodPacket(args[1:], stdout, stderr)
	case "evaluate":
		return runDogfoodEvaluate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "dogfood: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runDogfoodAuthoring(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "dogfood authoring: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "cases":
		return runDogfoodAuthoringCases(args[1:], stdout, stderr)
	case "packet":
		return runDogfoodAuthoringPacket(args[1:], stdout, stderr)
	case "evaluate":
		return runDogfoodAuthoringEvaluate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "dogfood authoring: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runDogfoodAuthoringCases(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood authoring cases", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "dogfood authoring cases: --json is required")
		return 2
	}
	cases, err := authoringdogfood.LoadCases(defaultAuthoringDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring cases: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, authoringdogfood.Summaries(cases))
}

func runDogfoodAuthoringPacket(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood authoring packet", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	caseID := flags.String("case", "", "authoring dogfood case id or public handle")
	workspacePath := flags.String("workspace", "", "fixture workspace path")
	argosBinary := flags.String("argos-binary", "", "argos binary path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*caseID) == "" {
		fmt.Fprintln(stderr, "dogfood authoring packet: --case is required")
		return 2
	}
	cases, err := authoringdogfood.LoadCases(defaultAuthoringDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring packet: %v\n", err)
		return 1
	}
	packet, err := authoringdogfood.BuildPacket(cases, authoringdogfood.PacketOptions{
		CaseID:      *caseID,
		Workspace:   *workspacePath,
		ArgosBinary: *argosBinary,
	})
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring packet: %v\n", err)
		return 2
	}
	if err := authoringdogfood.SeedFixtureWorkspace(defaultAuthoringDogfoodFixtures, packet.Fixture, packet.Workspace); err != nil {
		fmt.Fprintf(stderr, "dogfood authoring packet: seed workspace: %v\n", err)
		return 1
	}
	if *jsonOut {
		return printJSON(stdout, stderr, packet)
	}
	fmt.Fprint(stdout, packet.Markdown)
	return 0
}

func runDogfoodAuthoringEvaluate(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood authoring evaluate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	caseID := flags.String("case", "", "authoring dogfood case id or public handle")
	reportPath := flags.String("report", "", "markdown report path")
	workspacePath := flags.String("workspace", "", "workspace path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "dogfood authoring evaluate: --json is required")
		return 2
	}
	if strings.TrimSpace(*caseID) == "" {
		fmt.Fprintln(stderr, "dogfood authoring evaluate: --case is required")
		return 2
	}
	if strings.TrimSpace(*reportPath) == "" {
		fmt.Fprintln(stderr, "dogfood authoring evaluate: --report is required")
		return 2
	}
	if strings.TrimSpace(*workspacePath) == "" {
		fmt.Fprintln(stderr, "dogfood authoring evaluate: --workspace is required")
		return 2
	}
	cases, err := authoringdogfood.LoadCases(defaultAuthoringDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring evaluate: %v\n", err)
		return 1
	}
	data, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring evaluate: read report: %v\n", err)
		return 1
	}
	report, err := authoringdogfood.ParseMarkdownReport(data)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring evaluate: parse report: %v\n", err)
		return 1
	}
	evaluation, err := authoringdogfood.EvaluateCase(cases, *caseID, *workspacePath, report)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood authoring evaluate: %v\n", err)
		return 2
	}
	return printJSON(stdout, stderr, evaluation)
}

func runDogfoodCases(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood cases", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "dogfood cases: --json is required")
		return 2
	}
	cases, err := dogfood.LoadCases(defaultDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood cases: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, dogfood.Summaries(cases))
}

func runDogfoodPacket(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood packet", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	caseID := flags.String("case", "", "dogfood case id or public handle")
	workspacePath := flags.String("workspace", "", "fixture workspace path")
	argosBinary := flags.String("argos-binary", "", "argos binary path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*caseID) == "" {
		fmt.Fprintln(stderr, "dogfood packet: --case is required")
		return 2
	}
	cases, err := dogfood.LoadCases(defaultDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood packet: %v\n", err)
		return 1
	}
	packet, err := dogfood.BuildPacket(cases, dogfood.PacketOptions{
		CaseID:      *caseID,
		Workspace:   *workspacePath,
		ArgosBinary: *argosBinary,
	})
	if err != nil {
		fmt.Fprintf(stderr, "dogfood packet: %v\n", err)
		return 2
	}
	if *jsonOut {
		return printJSON(stdout, stderr, packet)
	}
	fmt.Fprint(stdout, packet.Markdown)
	return 0
}

func runDogfoodEvaluate(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood evaluate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	caseID := flags.String("case", "", "dogfood case id or public handle")
	reportPath := flags.String("report", "", "markdown report path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "dogfood evaluate: --json is required")
		return 2
	}
	if strings.TrimSpace(*caseID) == "" {
		fmt.Fprintln(stderr, "dogfood evaluate: --case is required")
		return 2
	}
	if strings.TrimSpace(*reportPath) == "" {
		fmt.Fprintln(stderr, "dogfood evaluate: --report is required")
		return 2
	}
	cases, err := dogfood.LoadCases(defaultDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood evaluate: %v\n", err)
		return 1
	}
	data, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood evaluate: read report: %v\n", err)
		return 1
	}
	report, err := dogfood.ParseMarkdownReport(data)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood evaluate: parse report: %v\n", err)
		return 1
	}
	evaluation, err := dogfood.EvaluateCase(cases, *caseID, report)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood evaluate: %v\n", err)
		return 2
	}
	if *jsonOut {
		return printJSON(stdout, stderr, evaluation)
	}
	fmt.Fprintf(stdout, "%s: %s\n", evaluation.CaseID, evaluation.Result)
	return 0
}

func runKnowledge(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "knowledge: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "find":
		return runKnowledgeFind(args[1:], stdout, stderr)
	case "list":
		return runKnowledgeList(args[1:], stdout, stderr)
	case "read":
		return runKnowledgeRead(args[1:], stdout, stderr)
	case "cite":
		return runKnowledgeCite(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "knowledge: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runKnowledgeFind(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge find", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	project := flags.String("project", "", "project id")
	phase := flags.String("phase", "", "workflow phase")
	task := flags.String("task", "", "task description")
	queryText := flags.String("query", "", "search query")
	var files multiValueFlag
	var types multiValueFlag
	var tags multiValueFlag
	var domains multiValueFlag
	var statuses multiValueFlag
	flags.Var(&files, "files", "file path to match; may be repeated")
	flags.Var(&types, "types", "knowledge item type to include; may be repeated")
	flags.Var(&tags, "tags", "tag to include; may be repeated")
	flags.Var(&domains, "domains", "domain to include; may be repeated")
	flags.Var(&statuses, "status", "status to include; may be repeated")
	includeDeprecated := flags.Bool("include-deprecated", false, "include deprecated knowledge items")
	limit := flags.Int("limit", 0, "maximum number of knowledge items to return")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "knowledge find: --json is required")
		return 2
	}
	if strings.TrimSpace(*project) == "" {
		fmt.Fprintln(stderr, "knowledge find: --project is required")
		return 2
	}
	if strings.TrimSpace(*task) == "" && strings.TrimSpace(*queryText) == "" {
		fmt.Fprintln(stderr, "knowledge find: --task or --query is required")
		return 2
	}
	if flagProvided(flags, "limit") && (*limit < 1 || *limit > 20) {
		fmt.Fprintln(stderr, "knowledge find: --limit must be between 1 and 20")
		return 2
	}
	store, closeStore, available := openIndexStore(stderr)
	if !available {
		return 1
	}
	defer closeStore()
	result, err := query.New(store).FindKnowledge(query.FindKnowledgeRequest{
		Project:           *project,
		Phase:             *phase,
		Task:              *task,
		Query:             *queryText,
		Files:             files,
		Types:             types,
		Tags:              tags,
		Domains:           domains,
		Status:            statuses,
		IncludeDeprecated: *includeDeprecated,
		Limit:             *limit,
	})
	if err != nil {
		fmt.Fprintf(stderr, "knowledge find: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}

func runKnowledgeList(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	project := flags.String("project", "", "project id")
	domain := flags.String("domain", "", "domain filter")
	var types multiValueFlag
	flags.Var(&types, "types", "knowledge item type to include; may be repeated")
	includeDeprecated := flags.Bool("include-deprecated", false, "include deprecated knowledge items")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "knowledge list: --json is required")
		return 2
	}
	if strings.TrimSpace(*project) == "" {
		fmt.Fprintln(stderr, "knowledge list: --project is required")
		return 2
	}
	store, closeStore, available := openIndexStore(stderr)
	if !available {
		return 1
	}
	defer closeStore()
	result, err := query.New(store).ListKnowledge(query.ListKnowledgeRequest{
		Project:           *project,
		Domain:            *domain,
		Types:             types,
		IncludeDeprecated: *includeDeprecated,
	})
	if err != nil {
		fmt.Fprintf(stderr, "knowledge list: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}

func runKnowledgeRead(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge read", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "knowledge read: --json is required")
		return 2
	}
	if flags.NArg() != 1 || strings.TrimSpace(flags.Arg(0)) == "" {
		fmt.Fprintln(stderr, "knowledge read: id is required")
		return 2
	}
	store, closeStore, available := openIndexStore(stderr)
	if !available {
		return 1
	}
	defer closeStore()
	result, err := query.New(store).ReadKnowledge(flags.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "knowledge read: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}

func runKnowledgeCite(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge cite", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "knowledge cite: --json is required")
		return 2
	}
	if len(flags.Args()) == 0 {
		fmt.Fprintln(stderr, "knowledge cite: at least one id is required")
		return 2
	}
	store, closeStore, available := openIndexStore(stderr)
	if !available {
		return 1
	}
	defer closeStore()
	result := query.New(store).CiteKnowledge(flags.Args())
	return printJSON(stdout, stderr, result)
}

type multiValueFlag []string

func (f *multiValueFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *multiValueFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*f = append(*f, value)
	return nil
}

func flagProvided(flags *flag.FlagSet, name string) bool {
	found := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func openIndexStore(stderr io.Writer) (*index.Store, func(), bool) {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "get current directory: %v\n", err)
		return nil, func() {}, false
	}

	dbPath := filepath.Join(root, "argos", "index.db")
	info, err := os.Stat(dbPath)
	if err != nil || !info.Mode().IsRegular() {
		fmt.Fprintln(stderr, "index not available: run argos index first")
		return nil, func() {}, false
	}

	store, err := index.Open(dbPath)
	if err != nil {
		fmt.Fprintln(stderr, "index not available: run argos index first")
		return nil, func() {}, false
	}
	if err := store.CheckSchema(); err != nil {
		_ = store.Close()
		fmt.Fprintln(stderr, "index not available: run argos index first")
		return nil, func() {}, false
	}
	return store, func() {
		_ = store.Close()
	}, true
}

func printJSON(stdout io.Writer, stderr io.Writer, value any) int {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "marshal JSON response: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, string(body))
	return 0
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
	fmt.Fprintln(w, "  author")
	fmt.Fprintln(w, "  knowledge")
	fmt.Fprintln(w, "  dogfood")
	fmt.Fprintln(w, "  mcp")
	fmt.Fprintln(w, "  promote")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  argos knowledge list --json --project <project>")
	fmt.Fprintln(w, "  argos knowledge find --json --project <project> --task <task>")
	fmt.Fprintln(w, "  argos knowledge read --json <id>")
	fmt.Fprintln(w, "  argos knowledge cite --json <id>...")
	fmt.Fprintln(w, "  argos author inspect --json --project <project> --goal <goal>")
	fmt.Fprintln(w, "  argos author verify --json --proposal <proposal.json> --path <candidate>")
}
