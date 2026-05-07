package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"argos/internal/version"
	"argos/internal/audit"
	"argos/internal/dogfood"
	"argos/internal/index"
	"argos/internal/knowledge"
	"argos/internal/knowledgewrite"
	"argos/internal/mcp"
	"argos/internal/provenance"
	"argos/internal/query"
	"argos/internal/registry"
	"argos/internal/workspace"
	"argos/internal/writedogfood"
)

const (
	defaultDogfoodCasesPath      = "testdata/discovery-golden/cases.json"
	defaultWriteDogfoodCasesPath = "testdata/write-golden/cases.json"
	defaultWriteDogfoodFixtures  = "testdata/write-golden/fixtures"
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
	case "--version", "-version":
		fmt.Fprintln(stdout, version.Version)
		return 0
	case "--help", "-help", "-h":
		printUsage(stdout)
		return 0
	case "version":
		fmt.Fprintln(stdout, version.Version)
		return 0
	case "help":
		printUsage(stdout)
		return 0
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
		includeInbox := flags.Bool("inbox", false, "validate inbox drafts")
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
	case "project":
		return runProject(args[1:], stdout, stderr)
	case "knowledge":
		return runKnowledge(args[1:], stdout, stderr)
	case "provenance":
		return runProvenance(args[1:], stdout, stderr)
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
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runDogfood(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "dogfood: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "write":
		return runDogfoodWrite(args[1:], stdout, stderr)
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

func runDogfoodWrite(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "dogfood write: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "cases":
		return runDogfoodWriteCases(args[1:], stdout, stderr)
	case "packet":
		return runDogfoodWritePacket(args[1:], stdout, stderr)
	case "evaluate":
		return runDogfoodWriteEvaluate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "dogfood write: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runDogfoodWriteCases(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood write cases", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "dogfood write cases: --json is required")
		return 2
	}
	cases, err := writedogfood.LoadCases(defaultWriteDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood write cases: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, writedogfood.Summaries(cases))
}

func runDogfoodWritePacket(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood write packet", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	caseID := flags.String("case", "", "write dogfood case id or public handle")
	workspacePath := flags.String("workspace", "", "fixture workspace path")
	argosBinary := flags.String("argos-binary", "", "argos binary path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*caseID) == "" {
		fmt.Fprintln(stderr, "dogfood write packet: --case is required")
		return 2
	}
	cases, err := writedogfood.LoadCases(defaultWriteDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood write packet: %v\n", err)
		return 1
	}
	tc, _, err := writedogfood.FindCase(cases, *caseID)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood write packet: %v\n", err)
		return 2
	}
	workspace := strings.TrimSpace(*workspacePath)
	if workspace == "" {
		fmt.Fprintln(stderr, "dogfood write packet: workspace is required")
		return 2
	}
	if err := writedogfood.SeedFixtureWorkspace(defaultWriteDogfoodFixtures, tc.Fixture, workspace); err != nil {
		fmt.Fprintf(stderr, "dogfood write packet: seed workspace: %v\n", err)
		return 1
	}
	packet, err := writedogfood.BuildPacket(cases, writedogfood.PacketOptions{
		CaseID:      *caseID,
		Workspace:   workspace,
		ArgosBinary: *argosBinary,
	})
	if err != nil {
		fmt.Fprintf(stderr, "dogfood write packet: %v\n", err)
		return 2
	}
	if *jsonOut {
		return printJSON(stdout, stderr, packet)
	}
	fmt.Fprint(stdout, packet.Markdown)
	return 0
}

func runDogfoodWriteEvaluate(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("dogfood write evaluate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	caseID := flags.String("case", "", "write dogfood case id or public handle")
	reportPath := flags.String("report", "", "markdown report path")
	workspacePath := flags.String("workspace", "", "workspace path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "dogfood write evaluate: --json is required")
		return 2
	}
	if strings.TrimSpace(*caseID) == "" {
		fmt.Fprintln(stderr, "dogfood write evaluate: --case is required")
		return 2
	}
	if strings.TrimSpace(*reportPath) == "" {
		fmt.Fprintln(stderr, "dogfood write evaluate: --report is required")
		return 2
	}
	if strings.TrimSpace(*workspacePath) == "" {
		fmt.Fprintln(stderr, "dogfood write evaluate: --workspace is required")
		return 2
	}
	cases, err := writedogfood.LoadCases(defaultWriteDogfoodCasesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood write evaluate: %v\n", err)
		return 1
	}
	data, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood write evaluate: read report: %v\n", err)
		return 1
	}
	report, err := writedogfood.ParseMarkdownReport(data)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood write evaluate: parse report: %v\n", err)
		return 1
	}
	evaluation, err := writedogfood.EvaluateCase(cases, *caseID, *workspacePath, report)
	if err != nil {
		fmt.Fprintf(stderr, "dogfood write evaluate: %v\n", err)
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

func runProject(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "project: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "add":
		return runProjectAdd(args[1:], stdout, stderr)
	case "list":
		return runProjectList(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "project: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runProjectAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("project add", flag.ContinueOnError)
	flags.SetOutput(stderr)
	id := flags.String("id", "", "project id")
	name := flags.String("name", "", "project name")
	projectPath := flags.String("path", "", "project source path")
	var techDomains multiValueFlag
	var businessDomains multiValueFlag
	flags.Var(&techDomains, "tech-domain", "tech domain; may be repeated")
	flags.Var(&businessDomains, "business-domain", "business domain; may be repeated")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*id) == "" {
		fmt.Fprintln(stderr, "project add: --id is required")
		return 2
	}
	if strings.TrimSpace(*name) == "" {
		fmt.Fprintln(stderr, "project add: --name is required")
		return 2
	}
	if strings.TrimSpace(*projectPath) == "" {
		fmt.Fprintln(stderr, "project add: --path is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "project add: get current directory: %v\n", err)
		return 1
	}
	project := registry.Project{
		ID:              *id,
		Name:            *name,
		Path:            *projectPath,
		TechDomains:     techDomains,
		BusinessDomains: businessDomains,
	}
	if err := registry.AddProject(root, project); err != nil {
		fmt.Fprintf(stderr, "project add: %v\n", err)
		return projectAddErrorCode(err)
	}
	fmt.Fprintf(stdout, "added project %s\n", strings.TrimSpace(*id))
	return 0
}

func projectAddErrorCode(err error) int {
	message := err.Error()
	for _, validationError := range []string{
		"project already exists:",
		"project id is required",
		"project name is required",
		"project path is required",
		"project path must be relative",
		"project path must stay inside workspace",
		"unknown tech domain:",
		"unknown business domain:",
	} {
		if strings.Contains(message, validationError) {
			return 2
		}
	}
	return 1
}

func runProjectList(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("project list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "project list: --json is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "project list: get current directory: %v\n", err)
		return 1
	}
	projects, err := registry.ListProjects(root)
	if err != nil {
		fmt.Fprintf(stderr, "project list: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, struct {
		Projects []registry.Project `json:"projects"`
	}{Projects: projects})
}

func runKnowledge(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "knowledge: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "design":
		return runKnowledgeDesign(args[1:], stdout, stderr)
	case "check":
		return runKnowledgeCheck(args[1:], stdout, stderr)
	case "publish":
		return runKnowledgePublish(args[1:], stdout, stderr)
	case "find":
		return runKnowledgeFind(args[1:], stdout, stderr)
	case "list":
		return runKnowledgeList(args[1:], stdout, stderr)
	case "read":
		return runKnowledgeRead(args[1:], stdout, stderr)
	case "cite":
		return runKnowledgeCite(args[1:], stdout, stderr)
	case "audit":
		return runKnowledgeAudit(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "knowledge: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runKnowledgeAudit(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge audit", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	project := flags.String("project", "", "project id filter")
	includePublished := flags.Bool("include-published", false, "include healthy published provenance items")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "knowledge audit: --json is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "knowledge audit: get current directory: %v\n", err)
		return 1
	}
	result, err := audit.Knowledge(root, audit.Request{
		Project:          *project,
		IncludePublished: *includePublished,
	})
	if err != nil {
		fmt.Fprintf(stderr, "knowledge audit: %v\n", err)
		return 1
	}
	if code := printJSON(stdout, stderr, result); code != 0 {
		return code
	}
	if result.Result == "problem" {
		return 1
	}
	return 0
}

func runProvenance(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "provenance: subcommand is required")
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "start":
		return runProvenanceStart(args[1:], stdout, stderr)
	case "record-decision":
		return runProvenanceRecordDecision(args[1:], stdout, stderr)
	case "record-check":
		return runProvenanceRecordCheck(args[1:], stdout, stderr)
	case "verify":
		return runProvenanceVerify(args[1:], stdout, stderr)
	case "list":
		return runProvenanceList(args[1:], stdout, stderr)
	case "status":
		return runProvenanceStatus(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "provenance: unknown subcommand %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runProvenanceStart(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance start", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	designPath := flags.String("design", "", "knowledge design JSON path")
	draftPath := flags.String("draft", "", "draft item or package path")
	createdBy := flags.String("created-by", "", "provenance record creator")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance start: --json is required")
		return 2
	}
	if strings.TrimSpace(*designPath) == "" {
		fmt.Fprintln(stderr, "provenance start: --design is required")
		return 2
	}
	if strings.TrimSpace(*draftPath) == "" {
		fmt.Fprintln(stderr, "provenance start: --draft is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance start: get current directory: %v\n", err)
		return 1
	}
	record, err := provenance.Start(root, provenance.StartRequest{
		DesignPath: *designPath,
		DraftPath:  *draftPath,
		CreatedBy:  *createdBy,
	})
	if err != nil {
		fmt.Fprintf(stderr, "provenance start: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, record)
}

func runProvenanceRecordDecision(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance record-decision", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	provenanceID := flags.String("provenance", "", "provenance id or path")
	stage := flags.String("stage", "", "decision stage")
	decisionValue := flags.String("decision", "", "decision value")
	decidedBy := flags.String("decided-by", "", "decision maker")
	role := flags.String("role", "", "decision maker role")
	source := flags.String("source", "", "decision source")
	reason := flags.String("reason", "", "decision reason")
	recordedBy := flags.String("recorded-by", "", "decision recorder")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance record-decision: --json is required")
		return 2
	}
	if strings.TrimSpace(*provenanceID) == "" {
		fmt.Fprintln(stderr, "provenance record-decision: --provenance is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance record-decision: get current directory: %v\n", err)
		return 1
	}
	decision, err := provenance.RecordDecision(root, *provenanceID, provenance.DecisionInput{
		Stage:      *stage,
		Decision:   *decisionValue,
		DecidedBy:  *decidedBy,
		Role:       *role,
		Source:     *source,
		Reason:     *reason,
		RecordedBy: *recordedBy,
	})
	if err != nil {
		fmt.Fprintf(stderr, "provenance record-decision: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, decision)
}

func runProvenanceRecordCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance record-check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	provenanceID := flags.String("provenance", "", "provenance id or path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance record-check: --json is required")
		return 2
	}
	if strings.TrimSpace(*provenanceID) == "" {
		fmt.Fprintln(stderr, "provenance record-check: --provenance is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance record-check: get current directory: %v\n", err)
		return 1
	}
	check, err := provenance.RecordCheck(root, *provenanceID)
	if err != nil {
		fmt.Fprintf(stderr, "provenance record-check: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, check)
}

func runProvenanceVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	provenanceID := flags.String("provenance", "", "provenance id or path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance verify: --json is required")
		return 2
	}
	if strings.TrimSpace(*provenanceID) == "" {
		fmt.Fprintln(stderr, "provenance verify: --provenance is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance verify: get current directory: %v\n", err)
		return 1
	}
	result, err := provenance.Verify(root, *provenanceID)
	if err != nil {
		fmt.Fprintf(stderr, "provenance verify: %v\n", err)
		return 1
	}
	if code := printJSON(stdout, stderr, result); code != 0 {
		return code
	}
	if result.Result != "pass" {
		return 1
	}
	return 0
}

func runProvenanceList(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	state := flags.String("state", "all", "provenance state filter")
	project := flags.String("project", "", "project id filter")
	knowledgeID := flags.String("knowledge-id", "", "knowledge id filter")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance list: --json is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance list: get current directory: %v\n", err)
		return 1
	}
	result, err := provenance.List(root, provenance.ListFilter{
		State:       *state,
		Project:     *project,
		KnowledgeID: *knowledgeID,
	})
	if err != nil {
		fmt.Fprintf(stderr, "provenance list: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}

func runProvenanceStatus(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("provenance status", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	provenanceID := flags.String("provenance", "", "provenance id or path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "provenance status: --json is required")
		return 2
	}
	if strings.TrimSpace(*provenanceID) == "" {
		fmt.Fprintln(stderr, "provenance status: --provenance is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "provenance status: get current directory: %v\n", err)
		return 1
	}
	result, err := provenance.Status(root, *provenanceID)
	if err != nil {
		fmt.Fprintf(stderr, "provenance status: %v\n", err)
		return 1
	}
	if code := printJSON(stdout, stderr, result); code != 0 {
		return code
	}
	if result.Result == "problem" {
		return 1
	}
	return 0
}

func runKnowledgeDesign(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge design", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	project := flags.String("project", "", "project id")
	intent := flags.String("intent", "", "knowledge intent")
	futureTask := flags.String("future-task", "", "future task this knowledge should support")
	phase := flags.String("phase", "", "workflow phase")
	queryText := flags.String("query", "", "search query")
	draftPath := flags.String("draft-path", "", "draft knowledge path")
	var files multiValueFlag
	var domains multiValueFlag
	var tags multiValueFlag
	flags.Var(&files, "files", "file path relevant to the knowledge intent; may be repeated")
	flags.Var(&domains, "domains", "domain relevant to the knowledge intent; may be repeated")
	flags.Var(&tags, "tags", "tag relevant to the knowledge intent; may be repeated")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "knowledge design: --json is required")
		return 2
	}
	if strings.TrimSpace(*project) == "" {
		fmt.Fprintln(stderr, "knowledge design: --project is required")
		return 2
	}
	if strings.TrimSpace(*intent) == "" {
		fmt.Fprintln(stderr, "knowledge design: --intent is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "knowledge design: get current directory: %v\n", err)
		return 1
	}
	result, err := knowledgewrite.Design(root, knowledgewrite.DesignRequest{
		Project:    *project,
		Intent:     *intent,
		FutureTask: *futureTask,
		Phase:      *phase,
		Query:      *queryText,
		Files:      files,
		Domains:    domains,
		Tags:       tags,
		DraftPath:  *draftPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "knowledge design: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}

func runKnowledgeCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print JSON output")
	designPath := flags.String("design", "", "knowledge design JSON path")
	draftPath := flags.String("draft", "", "draft item or package path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*jsonOut {
		fmt.Fprintln(stderr, "knowledge check: --json is required")
		return 2
	}
	if strings.TrimSpace(*designPath) == "" {
		fmt.Fprintln(stderr, "knowledge check: --design is required")
		return 2
	}
	if strings.TrimSpace(*draftPath) == "" {
		fmt.Fprintln(stderr, "knowledge check: --draft is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "knowledge check: get current directory: %v\n", err)
		return 1
	}
	result, err := knowledgewrite.Check(root, knowledgewrite.CheckRequest{
		DesignPath: *designPath,
		DraftPath:  *draftPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "knowledge check: %v\n", err)
		return 1
	}
	return printJSON(stdout, stderr, result)
}

func runKnowledgePublish(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("knowledge publish", flag.ContinueOnError)
	flags.SetOutput(stderr)
	provenancePath := flags.String("provenance", "", "provenance id or path")
	publishedBy := flags.String("published-by", "", "publisher identity")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*provenancePath) == "" {
		fmt.Fprintln(stderr, "knowledge publish: --provenance is required")
		return 2
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "knowledge publish: get current directory: %v\n", err)
		return 1
	}
	loaded, err := provenance.Load(root, *provenancePath)
	if err != nil {
		fmt.Fprintf(stderr, "knowledge publish: %v\n", err)
		return 1
	}
	target, err := publishDraftWithSideEffect(root, loaded.Record.Subject.DraftPath, stderr, func(target string) (func() error, error) {
		if filepath.ToSlash(target) != filepath.ToSlash(loaded.Record.Subject.OfficialPath) {
			return nil, fmt.Errorf("publish target %s does not match provenance official path %s", filepath.ToSlash(target), filepath.ToSlash(loaded.Record.Subject.OfficialPath))
		}
		_, rollback, err := provenance.PreparePublishMove(root, *provenancePath, *publishedBy)
		return rollback, err
	})
	if err != nil {
		fmt.Fprintf(stderr, "knowledge publish: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "published %s\n", target)
	fmt.Fprintln(stdout, "run argos index to refresh query results")
	return 0
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
		return mcp.NewServerWithRoot(root, nil), func() {}, false
	}

	store, err := index.Open(dbPath)
	if err != nil {
		return mcp.NewServerWithRoot(root, nil), func() {}, false
	}
	if err := store.CheckSchema(); err != nil {
		_ = store.Close()
		return mcp.NewServerWithRoot(root, nil), func() {}, false
	}
	return mcp.NewServerWithRoot(root, store), func() {
		_ = store.Close()
	}, true
}

type validationScope struct {
	Inbox bool
	Path  string
}

type knowledgeStorageScope string

const (
	knowledgeStorageOfficial knowledgeStorageScope = "official"
	knowledgeStorageInbox    knowledgeStorageScope = "inbox"
	knowledgeStorageUnknown  knowledgeStorageScope = "unknown"
)

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
	errs = append(errs, validateKnowledgeStorageScope(items, storageScopeForValidation(scope))...)
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

func storageScopeForValidation(scope validationScope) knowledgeStorageScope {
	if strings.TrimSpace(scope.Path) != "" {
		return storageScopeForPath(scope.Path)
	}
	if scope.Inbox {
		return knowledgeStorageInbox
	}
	return knowledgeStorageOfficial
}

func storageScopeForPath(path string) knowledgeStorageScope {
	slash := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if slash == "." || slash == "" {
		return knowledgeStorageUnknown
	}
	if slash == "knowledge/.inbox/items" ||
		slash == "knowledge/.inbox/packages" ||
		strings.HasPrefix(slash, "knowledge/.inbox/items/") ||
		strings.HasPrefix(slash, "knowledge/.inbox/packages/") {
		return knowledgeStorageInbox
	}
	if slash == "knowledge/items" ||
		slash == "knowledge/packages" ||
		strings.HasPrefix(slash, "knowledge/items/") ||
		strings.HasPrefix(slash, "knowledge/packages/") {
		return knowledgeStorageOfficial
	}
	return knowledgeStorageUnknown
}

func validateKnowledgeStorageScope(items []knowledge.Item, scope knowledgeStorageScope) []error {
	var errs []error
	for _, item := range items {
		status := strings.TrimSpace(item.Status)
		switch scope {
		case knowledgeStorageInbox:
			if status != "draft" {
				errs = append(errs, fmt.Errorf("%s: inbox knowledge must use status: draft; set the draft back to status: draft before check or publish", item.Path))
			}
		case knowledgeStorageOfficial:
			if status == "draft" {
				errs = append(errs, fmt.Errorf("%s: official knowledge must not use status: draft; publish from inbox or set status: active after review", item.Path))
			}
		}
	}
	return errs
}

type publishSideEffect func(target string) (func() error, error)

func publishDraft(root string, relPath string, stderr io.Writer) (string, error) {
	return publishDraftWithSideEffect(root, relPath, stderr, nil)
}

func publishDraftWithSideEffect(root string, relPath string, stderr io.Writer, sideEffect publishSideEffect) (string, error) {
	clean := filepath.Clean(relPath)
	if filepath.IsAbs(relPath) || clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("%s: draft path must be relative and inside workspace", relPath)
	}
	target, err := publishTarget(clean)
	if err != nil {
		return "", err
	}
	draftItems, err := loadAndValidateKnowledge(root, stderr, validationScope{Path: clean})
	if err != nil {
		return "", err
	}
	targetAbs := filepath.Join(root, target)
	if _, err := os.Stat(targetAbs); err == nil {
		return "", fmt.Errorf("target already exists: %s", target)
	} else if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat target %s: %w", target, err)
	}
	if err := validateOfficialKnowledgeWithDraft(root, clean, target, draftItems, stderr); err != nil {
		return "", fmt.Errorf("official validation failed before publish: %w", err)
	}
	snapshots, err := snapshotKnowledgeFiles(root, draftItems)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
		return "", fmt.Errorf("create target parent: %w", err)
	}
	if err := os.Rename(filepath.Join(root, clean), targetAbs); err != nil {
		return "", fmt.Errorf("move draft: %w", err)
	}
	if err := activatePublishedKnowledge(root, target); err != nil {
		if rollbackErr := rollbackPublishedKnowledge(root, target, clean, snapshots); rollbackErr != nil {
			return "", fmt.Errorf("activate published knowledge: %w; %v", err, rollbackErr)
		}
		return "", fmt.Errorf("activate published knowledge: %w", err)
	}
	var rollbackSideEffect func() error
	if sideEffect != nil {
		var err error
		rollbackSideEffect, err = sideEffect(target)
		if err != nil {
			if rollbackErr := rollbackPublishedKnowledge(root, target, clean, snapshots); rollbackErr != nil {
				return "", fmt.Errorf("publish side effect failed: %w; %v", err, rollbackErr)
			}
			return "", fmt.Errorf("publish side effect failed: %w", err)
		}
	}
	if _, err := loadAndValidateKnowledge(root, stderr, validationScope{Path: target}); err != nil {
		var rollbackFailures []string
		if rollbackSideEffect != nil {
			if rollbackErr := rollbackSideEffect(); rollbackErr != nil {
				rollbackFailures = append(rollbackFailures, rollbackErr.Error())
			}
		}
		if rollbackErr := rollbackPublishedKnowledge(root, target, clean, snapshots); rollbackErr != nil {
			rollbackFailures = append(rollbackFailures, rollbackErr.Error())
		}
		if len(rollbackFailures) > 0 {
			return "", fmt.Errorf("official validation failed after publish: %w; %s", err, strings.Join(rollbackFailures, "; "))
		}
		return "", fmt.Errorf("official validation failed after publish: %w", err)
	}
	return target, nil
}

type knowledgeFileSnapshot struct {
	Path string
	Data []byte
	Mode os.FileMode
}

func snapshotKnowledgeFiles(root string, items []knowledge.Item) ([]knowledgeFileSnapshot, error) {
	snapshots := make([]knowledgeFileSnapshot, 0, len(items))
	for _, item := range items {
		absPath := filepath.Join(root, item.Path)
		info, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", item.Path, err)
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", item.Path, err)
		}
		snapshots = append(snapshots, knowledgeFileSnapshot{
			Path: item.Path,
			Data: data,
			Mode: info.Mode().Perm(),
		})
	}
	return snapshots, nil
}

func rollbackPublishedKnowledge(root string, target string, draft string, snapshots []knowledgeFileSnapshot) error {
	var failures []string
	targetAbs := filepath.Join(root, target)
	draftAbs := filepath.Join(root, draft)
	if err := os.MkdirAll(filepath.Dir(draftAbs), 0o755); err != nil {
		failures = append(failures, fmt.Sprintf("create draft parent: %v", err))
	} else if err := os.Rename(targetAbs, draftAbs); err != nil {
		failures = append(failures, fmt.Sprintf("move published draft back: %v", err))
	}
	for _, snapshot := range snapshots {
		absPath := filepath.Join(root, snapshot.Path)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			failures = append(failures, fmt.Sprintf("restore %s parent: %v", snapshot.Path, err))
			continue
		}
		if err := os.WriteFile(absPath, snapshot.Data, snapshot.Mode); err != nil {
			failures = append(failures, fmt.Sprintf("restore %s: %v", snapshot.Path, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("rollback publish failed: %s", strings.Join(failures, "; "))
	}
	return nil
}

func activatePublishedKnowledge(root string, relPath string) error {
	items, err := knowledge.LoadPath(root, relPath)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := activateKnowledgeFile(filepath.Join(root, item.Path), item.Path); err != nil {
			return err
		}
	}
	return nil
}

func activateKnowledgeFile(absPath string, relPath string) error {
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", relPath, err)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", relPath, err)
	}
	const opening = "---\n"
	const closing = "\n---\n"
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, opening) {
		return fmt.Errorf("%s: missing frontmatter opening delimiter", relPath)
	}
	end := strings.Index(text[len(opening):], closing)
	if end < 0 {
		return fmt.Errorf("%s: missing frontmatter closing delimiter", relPath)
	}

	frontmatterEnd := len(opening) + end
	frontmatter := text[len(opening):frontmatterEnd]
	body := text[frontmatterEnd+len(closing):]
	lines := strings.Split(frontmatter, "\n")
	statusFound := false
	for i, line := range lines {
		if strings.HasPrefix(line, "status:") {
			lines[i] = "status: active"
			statusFound = true
			break
		}
	}
	if !statusFound {
		return fmt.Errorf("%s: missing status field", relPath)
	}

	next := opening + strings.Join(lines, "\n") + closing + body
	if err := os.WriteFile(absPath, []byte(next), info.Mode().Perm()); err != nil {
		return fmt.Errorf("write %s: %w", relPath, err)
	}
	return nil
}

func validateOfficialKnowledgeWithDraft(root string, draftPath string, target string, draftItems []knowledge.Item, stderr io.Writer) error {
	reg, err := registry.Load(root)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}
	official, err := knowledge.LoadOfficial(root)
	if err != nil {
		return fmt.Errorf("load official knowledge: %w", err)
	}
	items := append([]knowledge.Item{}, official...)
	items = append(items, rebaseKnowledgeItemPaths(draftItems, draftPath, target)...)
	errs := knowledge.ValidateItems(items, reg)
	errs = append(errs, validateKnowledgeStorageScope(official, knowledgeStorageOfficial)...)
	for _, err := range errs {
		fmt.Fprintln(stderr, err)
	}
	if len(errs) > 0 {
		err := fmt.Errorf("validation failed with %d error(s)", len(errs))
		fmt.Fprintln(stderr, err)
		return err
	}
	return nil
}

func rebaseKnowledgeItemPaths(items []knowledge.Item, from string, to string) []knowledge.Item {
	fromSlash := strings.TrimSuffix(filepath.ToSlash(from), "/")
	toSlash := strings.TrimSuffix(filepath.ToSlash(to), "/")
	rebased := make([]knowledge.Item, 0, len(items))
	for _, item := range items {
		item.Path = rebaseKnowledgePath(item.Path, fromSlash, toSlash)
		rebased = append(rebased, item)
	}
	return rebased
}

func rebaseKnowledgePath(path string, from string, to string) string {
	path = filepath.ToSlash(path)
	if path == from {
		return to
	}
	if strings.HasPrefix(path, from+"/") {
		return to + strings.TrimPrefix(path, from)
	}
	return path
}

func publishTarget(clean string) (string, error) {
	slash := filepath.ToSlash(clean)
	const inboxPackages = "knowledge/.inbox/packages/"
	if strings.HasPrefix(slash, inboxPackages) {
		rest := strings.TrimPrefix(slash, inboxPackages)
		if rest == "" || strings.Contains(rest, "../") {
			return "", fmt.Errorf("%s: invalid inbox draft path", clean)
		}
		if rest == "KNOWLEDGE.md" || strings.HasSuffix(rest, "/KNOWLEDGE.md") {
			return "", fmt.Errorf("%s: package publish path must be the package directory, not KNOWLEDGE.md", clean)
		}
		return filepath.FromSlash("knowledge/packages/" + rest), nil
	}
	for _, mapping := range []struct {
		inbox    string
		official string
	}{
		{"knowledge/.inbox/items/", "knowledge/items/"},
	} {
		if strings.HasPrefix(slash, mapping.inbox) {
			rest := strings.TrimPrefix(slash, mapping.inbox)
			if rest == "" || strings.Contains(rest, "../") {
				return "", fmt.Errorf("%s: invalid inbox draft path", clean)
			}
			return filepath.FromSlash(mapping.official + rest), nil
		}
	}
	return "", fmt.Errorf("%s: draft must be under knowledge/.inbox/items or knowledge/.inbox/packages", clean)
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: argos <command> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  help")
	fmt.Fprintln(w, "  version")
	fmt.Fprintln(w, "  init")
	fmt.Fprintln(w, "  validate")
	fmt.Fprintln(w, "  index")
	fmt.Fprintln(w, "  context")
	fmt.Fprintln(w, "  project")
	fmt.Fprintln(w, "  knowledge")
	fmt.Fprintln(w, "  provenance")
	fmt.Fprintln(w, "  dogfood")
	fmt.Fprintln(w, "  mcp")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  argos project add --id <project> --name <name> --path <path> --tech-domain <domain> --business-domain <domain>")
	fmt.Fprintln(w, "  argos project list --json")
	fmt.Fprintln(w, "  argos knowledge design --json --project <project> --intent <intent>")
	fmt.Fprintln(w, "  argos knowledge check --json --design <design.json> --draft <draft>")
	fmt.Fprintln(w, "  argos knowledge publish --provenance <id>")
	fmt.Fprintln(w, "  argos knowledge list --json --project <project>")
	fmt.Fprintln(w, "  argos knowledge find --json --project <project> --task <task>")
	fmt.Fprintln(w, "  argos knowledge read --json <id>")
	fmt.Fprintln(w, "  argos knowledge cite --json <id>...")
	fmt.Fprintln(w, "  argos knowledge audit --json")
	fmt.Fprintln(w, "  argos provenance start --json --design <design.json> --draft <draft>")
	fmt.Fprintln(w, "  argos provenance record-decision --json --provenance <id> --stage <stage> --decision <decision> --decided-by <actor> --role <role> --source <source> --reason <reason> --recorded-by <agent>")
	fmt.Fprintln(w, "  argos provenance record-check --json --provenance <id>")
	fmt.Fprintln(w, "  argos provenance verify --json --provenance <id>")
	fmt.Fprintln(w, "  argos provenance list --json")
	fmt.Fprintln(w, "  argos provenance status --json --provenance <id>")
}
