package authoringdogfood

import (
	"strings"
)

const (
	ResultPass         = "pass"
	ResultFail         = "fail"
	ResultReviewNeeded = "review-needed"

	reportStatusNotApplicable = "not-applicable"
	reportStatusNotRun        = "not-run"
)

type HumanReviewDecisions struct {
	ProposalApproved           bool `json:"proposal_approved"`
	CandidateWriteApproved     bool `json:"candidate_write_approved"`
	PriorityMustAuthorized     bool `json:"priority_must_authorized"`
	OfficialMutationAuthorized bool `json:"official_mutation_authorized"`
	PromoteAuthorized          bool `json:"promote_authorized"`
}

type Report struct {
	CaseID          string               `json:"case_id"`
	ProposalPath    string               `json:"proposal_path"`
	CandidatePath   string               `json:"candidate_path"`
	VerifyResult    string               `json:"verify_result"`
	HumanReview     HumanReviewDecisions `json:"human_review"`
	Guards          map[string]string    `json:"guards"`
	Result          string               `json:"result"`
	MissingSections []string             `json:"missing_sections"`
	MissingFields   []string             `json:"missing_fields"`
	fieldPresence   map[string]bool
}

var requiredAuthoringReportSections = []struct {
	key     string
	heading string
}{
	{key: "inputs", heading: "Inputs"},
	{key: "tool transcript summary", heading: "Tool Transcript Summary"},
	{key: "artifacts", heading: "Artifacts"},
	{key: "human review decisions", heading: "Human Review Decisions"},
	{key: "guards", heading: "Guards"},
	{key: "result", heading: "Result"},
}

var requiredAuthoringReportFields = []string{
	"case",
	"proposal path",
	"candidate path",
	"author verify result",
	"proposal approved",
	"candidate write approved",
	"priority must authorized",
	"official mutation authorized",
	"promote authorized",
	"result",
}

var requiredAuthoringReportGuards = []string{
	"proposal reviewed before candidate write",
	"source and scope documented",
	"future use documented",
	"candidate stayed in approved area",
	"official knowledge unchanged",
	"promotion not run",
	"verification run",
}

func ParseMarkdownReport(data []byte) (Report, error) {
	text := string(data)
	report := Report{
		CaseID:          parseAuthoringReportCaseID(text),
		Guards:          map[string]string{},
		MissingSections: []string{},
		MissingFields:   []string{},
		fieldPresence:   map[string]bool{},
	}
	if report.CaseID != "" {
		report.fieldPresence["case"] = true
	}

	sections := parseAuthoringReportSections(text)
	for _, section := range requiredAuthoringReportSections {
		if _, ok := sections[section.key]; !ok {
			report.MissingSections = append(report.MissingSections, section.heading)
		}
	}

	parseAuthoringArtifacts(sections["artifacts"], &report)
	parseAuthoringHumanReview(sections["human review decisions"], &report)
	parseAuthoringGuards(sections["guards"], &report)
	report.Result = parseAuthoringResult(sections["result"], &report)
	report.MissingFields = missingAuthoringReportFields(report)

	return report, nil
}

func parseAuthoringReportCaseID(text string) string {
	for _, line := range strings.Split(text, "\n") {
		label, value, ok := splitAuthoringReportLabel(line)
		if ok && strings.EqualFold(label, "case") {
			return cleanAuthoringReportValue(value)
		}
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			return ""
		}
	}
	return ""
}

func parseAuthoringReportSections(text string) map[string]string {
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
			current = normalizeAuthoringReportLabel(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
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

func parseAuthoringArtifacts(section string, report *Report) {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitAuthoringReportBullet(line)
		if !ok {
			continue
		}
		switch authoringArtifactField(label) {
		case "proposal path":
			report.ProposalPath = cleanAuthoringReportPathValue(value)
			markAuthoringReportField(report, "proposal path", value)
		case "candidate path":
			report.CandidatePath = cleanAuthoringReportPathValue(value)
			markAuthoringReportField(report, "candidate path", value)
		case "author verify result":
			report.VerifyResult = cleanAuthoringReportStatus(value)
			markAuthoringReportField(report, "author verify result", value)
		}
	}
}

func parseAuthoringHumanReview(section string, report *Report) {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitAuthoringReportBullet(line)
		if !ok {
			continue
		}
		field := authoringHumanReviewField(label)
		switch field {
		case "proposal approved":
			report.HumanReview.ProposalApproved = parseAuthoringReportBool(value)
		case "candidate write approved":
			report.HumanReview.CandidateWriteApproved = parseAuthoringReportBool(value)
		case "priority must authorized":
			report.HumanReview.PriorityMustAuthorized = parseAuthoringReportBool(value)
		case "official mutation authorized":
			report.HumanReview.OfficialMutationAuthorized = parseAuthoringReportBool(value)
		case "promote authorized":
			report.HumanReview.PromoteAuthorized = parseAuthoringReportBool(value)
		default:
			continue
		}
		markAuthoringReportField(report, field, value)
	}
}

func parseAuthoringGuards(section string, report *Report) {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitAuthoringReportBullet(line)
		if !ok {
			continue
		}
		guard := normalizeAuthoringReportLabel(label)
		if guard == "" {
			continue
		}
		report.Guards[guard] = cleanAuthoringReportStatus(value)
		markAuthoringReportField(report, guard, value)
	}
}

func parseAuthoringResult(section string, report *Report) string {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitAuthoringReportLabel(line)
		if ok && strings.EqualFold(label, "result") {
			markAuthoringReportField(report, "result", value)
			return cleanAuthoringReportStatus(value)
		}
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			markAuthoringReportField(report, "result", trimmed)
			return cleanAuthoringReportStatus(trimmed)
		}
	}
	return ""
}

func splitAuthoringReportBullet(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return splitAuthoringReportLabel(strings.TrimSpace(trimmed[2:]))
	}
	return "", "", false
}

func splitAuthoringReportLabel(line string) (string, string, bool) {
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

func authoringArtifactField(label string) string {
	normalized := normalizeAuthoringReportLabel(label)
	switch {
	case normalized == "proposal" || normalized == "proposal artifact" || normalized == "proposal file":
		return "proposal path"
	case normalized == "candidate" || normalized == "candidate artifact" || normalized == "candidate file":
		return "candidate path"
	case normalized == "verify result" || normalized == "verification result" || normalized == "author verify":
		return "author verify result"
	default:
		return normalized
	}
}

func authoringHumanReviewField(label string) string {
	normalized := normalizeAuthoringReportLabel(label)
	switch normalized {
	case "promotion authorized":
		return "promote authorized"
	default:
		return normalized
	}
}

func cleanAuthoringReportStatus(value string) string {
	cleaned := strings.ToLower(cleanAuthoringReportValue(value))
	for _, status := range []string{ResultReviewNeeded, reportStatusNotApplicable, reportStatusNotRun, ResultPass, ResultFail} {
		if hasAuthoringReportTokenPrefix(cleaned, status) {
			return status
		}
	}
	for _, synonym := range []string{"passed", "yes", "true", "ok", "followed"} {
		if hasAuthoringReportTokenPrefix(cleaned, synonym) {
			return ResultPass
		}
	}
	for _, synonym := range []string{"failed", "no", "false"} {
		if hasAuthoringReportTokenPrefix(cleaned, synonym) {
			return ResultFail
		}
	}
	return cleaned
}

func cleanAuthoringReportPathValue(value string) string {
	cleaned := cleanAuthoringReportValue(value)
	if isNoneAuthoringReportValue(cleaned) || hasAuthoringReportTokenPrefix(strings.ToLower(cleaned), reportStatusNotRun) {
		return ""
	}
	return cleaned
}

func cleanAuthoringReportValue(value string) string {
	return strings.Trim(strings.TrimSpace(value), "`\"' \t\r\n.,;:")
}

func normalizeAuthoringReportLabel(label string) string {
	return strings.ToLower(cleanAuthoringReportValue(label))
}

func parseAuthoringReportBool(value string) bool {
	cleaned := strings.ToLower(cleanAuthoringReportValue(value))
	for _, token := range []string{"false", "no", "not approved", "denied", "not-authorized", "not authorized"} {
		if hasAuthoringReportTokenPrefix(cleaned, token) {
			return false
		}
	}
	for _, token := range []string{"true", "yes", "approved", "authorized", "pass"} {
		if hasAuthoringReportTokenPrefix(cleaned, token) {
			return true
		}
	}
	return false
}

func missingAuthoringReportFields(report Report) []string {
	var missing []string
	for _, field := range requiredAuthoringReportFields {
		if !report.hasField(field) {
			missing = append(missing, field)
		}
	}
	for _, guard := range requiredAuthoringReportGuards {
		if !report.hasField(guard) {
			missing = append(missing, guard)
		}
	}
	return missing
}

func markAuthoringReportField(report *Report, field string, value string) {
	if report.fieldPresence == nil {
		return
	}
	if cleanAuthoringReportValue(value) != "" {
		report.fieldPresence[field] = true
	}
}

func (report Report) hasField(field string) bool {
	if report.fieldPresence == nil {
		return true
	}
	return report.fieldPresence[field]
}

func isNoneAuthoringReportValue(value string) bool {
	return hasAuthoringReportTokenPrefix(strings.ToLower(cleanAuthoringReportValue(value)), "none")
}

func hasAuthoringReportTokenPrefix(value string, token string) bool {
	if !strings.HasPrefix(value, token) {
		return false
	}
	return len(value) == len(token) || isAuthoringReportTokenBoundary(rune(value[len(token)]))
}

func isAuthoringReportTokenBoundary(r rune) bool {
	return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-')
}
