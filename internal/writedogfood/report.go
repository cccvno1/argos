package writedogfood

import "strings"

const (
	ResultPass         = "pass"
	ResultFail         = "fail"
	ResultReviewNeeded = "review-needed"

	reportStatusNotApplicable = "not-applicable"
	reportStatusNotRun        = "not-run"
)

type ReviewDecisions struct {
	DesignApproved        bool `json:"design_approved"`
	DraftWriteApproved    bool `json:"draft_write_approved"`
	PriorityMustApproved  bool `json:"priority_must_approved"`
	OfficialWriteApproved bool `json:"official_write_approved"`
	PublishApproved       bool `json:"publish_approved"`
}

type Report struct {
	CaseID          string            `json:"case_id"`
	State           string            `json:"state"`
	NextAction      string            `json:"next_action"`
	DesignPath      string            `json:"design_path"`
	DraftPath       string            `json:"draft_path"`
	DraftAllowed    bool              `json:"draft_allowed"`
	DesignOnly      bool              `json:"design_only"`
	CheckResult     string            `json:"check_result"`
	Review          ReviewDecisions   `json:"review"`
	Guards          map[string]string `json:"guards"`
	Result          string            `json:"result"`
	MissingSections []string          `json:"missing_sections"`
	MissingFields   []string          `json:"missing_fields"`
	fieldPresence   map[string]bool
}

var requiredWriteReportSections = []struct {
	key     string
	heading string
}{
	{key: "inputs", heading: "Inputs"},
	{key: "write guidance", heading: "Write Guidance"},
	{key: "artifacts", heading: "Artifacts"},
	{key: "review decisions", heading: "Review Decisions"},
	{key: "guards", heading: "Guards"},
	{key: "result", heading: "Result"},
}

var requiredWriteReportFields = []string{
	"case",
	"state",
	"next action",
	"design path",
	"draft path",
	"draft allowed",
	"design only",
	"check result",
	"design approved",
	"draft write approved",
	"priority must approved",
	"official write approved",
	"publish approved",
	"result",
}

var requiredWriteReportGuards = []string{
	"design reviewed before draft write",
	"sources and scope documented",
	"future use documented",
	"draft stayed in approved area",
	"official knowledge unchanged",
	"publish not run",
	"check run",
}

func ParseMarkdownReport(data []byte) (Report, error) {
	text := string(data)
	report := Report{
		CaseID:          parseReportCaseID(text),
		Guards:          map[string]string{},
		MissingSections: []string{},
		MissingFields:   []string{},
		fieldPresence:   map[string]bool{},
	}
	if report.CaseID != "" {
		report.fieldPresence["case"] = true
	}

	sections := parseReportSections(text)
	for _, section := range requiredWriteReportSections {
		if _, ok := sections[section.key]; !ok {
			report.MissingSections = append(report.MissingSections, section.heading)
		}
	}

	parseWriteGuidance(sections["write guidance"], &report)
	parseArtifacts(sections["artifacts"], &report)
	parseReviewDecisions(sections["review decisions"], &report)
	parseGuards(sections["guards"], &report)
	report.Result = parseResult(sections["result"], &report)
	report.MissingFields = missingReportFields(report)

	return report, nil
}

func parseReportCaseID(text string) string {
	for _, line := range strings.Split(text, "\n") {
		label, value, ok := splitReportLabel(line)
		if ok && strings.EqualFold(label, "case") {
			return cleanReportValue(value)
		}
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			return ""
		}
	}
	return ""
}

func parseReportSections(text string) map[string]string {
	sections := map[string]string{}
	var current string
	var builder strings.Builder
	flush := func() {
		if current == "" {
			return
		}
		sections[current] = strings.TrimSpace(builder.String())
		builder.Reset()
	}

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			flush()
			current = normalizeReportLabel(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
			continue
		}
		if current == "" {
			continue
		}
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
	flush()
	return sections
}

func parseWriteGuidance(section string, report *Report) {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitReportBullet(line)
		if !ok {
			continue
		}
		field := normalizeReportLabel(label)
		switch field {
		case "state":
			report.State = cleanReportValue(value)
		case "next action":
			report.NextAction = cleanReportValue(value)
		case "design path":
			report.DesignPath = cleanReportPathValue(value)
		case "draft path":
			report.DraftPath = cleanReportPathValue(value)
		case "draft allowed":
			report.DraftAllowed = parseReportBool(value)
		case "design only":
			report.DesignOnly = parseReportBool(value)
		case "check result":
			report.CheckResult = cleanReportStatus(value)
		default:
			continue
		}
		markReportField(report, field, value)
	}
}

func parseArtifacts(section string, report *Report) {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitReportBullet(line)
		if !ok {
			continue
		}
		switch artifactField(label) {
		case "design path":
			report.DesignPath = cleanReportPathValue(value)
			markReportField(report, "design path", value)
		case "draft path":
			report.DraftPath = cleanReportPathValue(value)
			markReportField(report, "draft path", value)
		case "check result":
			report.CheckResult = cleanReportStatus(value)
			markReportField(report, "check result", value)
		}
	}
}

func parseReviewDecisions(section string, report *Report) {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitReportBullet(line)
		if !ok {
			continue
		}
		field := normalizeReportLabel(label)
		switch field {
		case "design approved":
			report.Review.DesignApproved = parseReportBool(value)
		case "draft write approved":
			report.Review.DraftWriteApproved = parseReportBool(value)
		case "priority must approved":
			report.Review.PriorityMustApproved = parseReportBool(value)
		case "official write approved":
			report.Review.OfficialWriteApproved = parseReportBool(value)
		case "publish approved":
			report.Review.PublishApproved = parseReportBool(value)
		default:
			continue
		}
		markReportField(report, field, value)
	}
}

func parseGuards(section string, report *Report) {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitReportBullet(line)
		if !ok {
			continue
		}
		guard := normalizeReportLabel(label)
		if guard == "" {
			continue
		}
		report.Guards[guard] = cleanReportStatus(value)
		markReportField(report, guard, value)
	}
}

func parseResult(section string, report *Report) string {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitReportLabel(line)
		if ok && strings.EqualFold(label, "result") {
			markReportField(report, "result", value)
			return cleanReportStatus(value)
		}
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			markReportField(report, "result", trimmed)
			return cleanReportStatus(trimmed)
		}
	}
	return ""
}

func splitReportBullet(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return splitReportLabel(strings.TrimSpace(trimmed[2:]))
	}
	return "", "", false
}

func splitReportLabel(line string) (string, string, bool) {
	before, after, ok := strings.Cut(strings.TrimSpace(line), ":")
	if !ok {
		return "", "", false
	}
	label := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(before), "#"))
	if label == "" {
		return "", "", false
	}
	return label, strings.TrimSpace(after), true
}

func artifactField(label string) string {
	normalized := normalizeReportLabel(label)
	switch normalized {
	case "design", "design artifact", "design file":
		return "design path"
	case "draft", "draft artifact", "draft file":
		return "draft path"
	case "check", "knowledge check", "knowledge check result":
		return "check result"
	default:
		return normalized
	}
}

func missingReportFields(report Report) []string {
	var missing []string
	for _, field := range requiredWriteReportFields {
		if !report.hasField(field) {
			missing = append(missing, field)
		}
	}
	for _, guard := range requiredWriteReportGuards {
		if !report.hasField(guard) {
			missing = append(missing, guard)
		}
	}
	return missing
}

func markReportField(report *Report, field string, rawValue string) {
	if !placeholderReportValue(rawValue) {
		report.fieldPresence[normalizeReportLabel(field)] = true
	}
}

func (report Report) hasField(field string) bool {
	return report.fieldPresence[normalizeReportLabel(field)]
}

func cleanReportPathValue(value string) string {
	clean := cleanReportValue(value)
	if strings.EqualFold(clean, "none") || strings.EqualFold(clean, "not-run") {
		return ""
	}
	return clean
}

func cleanReportStatus(value string) string {
	value = strings.ToLower(cleanReportValue(value))
	for _, suffix := range []string{";", ".", ","} {
		if before, _, ok := strings.Cut(value, suffix); ok {
			value = strings.TrimSpace(before)
		}
	}
	return value
}

func cleanReportValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`")
	value = strings.TrimSpace(value)
	for _, marker := range []string{" #", " //"} {
		if before, _, ok := strings.Cut(value, marker); ok {
			value = strings.TrimSpace(before)
		}
	}
	return strings.Trim(value, "`")
}

func parseReportBool(value string) bool {
	switch strings.ToLower(cleanReportValue(value)) {
	case "true", "yes", "approved", "pass":
		return true
	default:
		return false
	}
}

func placeholderReportValue(value string) bool {
	clean := strings.ToLower(cleanReportValue(value))
	return clean == ""
}

func normalizeReportLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "`", "")
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	return strings.Join(strings.Fields(value), " ")
}
