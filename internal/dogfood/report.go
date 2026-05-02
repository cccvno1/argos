package dogfood

import (
	"regexp"
	"strings"
)

type Report struct {
	CaseID          string            `json:"case_id"`
	ActualSupport   string            `json:"actual_support"`
	UsageGuidance   string            `json:"usage_guidance"`
	SearchStatus    string            `json:"search_status"`
	DiscoveredIDs   []string          `json:"discovered_ids"`
	ReadIDs         []string          `json:"read_ids"`
	CitedIDs        []string          `json:"cited_ids"`
	MissingNeeds    []string          `json:"missing_needs"`
	NextSteps       string            `json:"next_steps"`
	Guards          map[string]string `json:"guards"`
	Result          string            `json:"result"`
	MissingSections []string          `json:"missing_sections"`
	MissingFields   []string          `json:"missing_fields"`
	fieldPresence   map[string]bool
}

var reportIDPattern = regexp.MustCompile(`[a-z][a-z0-9-]*:[a-z0-9._-]+`)

var requiredReportSections = []struct {
	key     string
	heading string
}{
	{key: "inputs", heading: "Inputs"},
	{key: "tool transcript summary", heading: "Tool Transcript Summary"},
	{key: "observed results", heading: "Observed Results"},
	{key: "guards", heading: "Guards"},
	{key: "result", heading: "Result"},
}

var requiredReportFields = []string{
	"actual support",
	"usage guidance",
	"search status",
	"discovered ids",
	"read ids",
	"cited ids",
	"missing needs",
	"argos-backed vs general reasoning",
	"next steps",
}

var requiredReportGuards = []string{
	"progressive reading",
	"weak/none no-overclaim",
	"citation accountability",
	"cited ids subset of read-and-used ids",
	"missing needs not cited",
	"attribution boundary",
	"no discovery-triggered upload/capture",
	"usage guidance followed",
	"context contamination",
}

var knownReportGuards = map[string]string{
	"progressive reading":                   "progressive reading",
	"weak/none no-overclaim":                "weak/none no-overclaim",
	"citation accountability":               "citation accountability",
	"cited ids subset of read-and-used ids": "cited ids subset of read-and-used ids",
	"missing needs not cited":               "missing needs not cited",
	"attribution boundary":                  "attribution boundary",
	"no discovery-triggered upload/capture": "no discovery-triggered upload/capture",
	"usage guidance followed":               "usage guidance followed",
	"context contamination":                 "context contamination",
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
	for _, section := range requiredReportSections {
		if _, ok := sections[section.key]; !ok {
			report.MissingSections = append(report.MissingSections, section.heading)
		}
	}

	parseObservedResults(sections["observed results"], &report)
	parseReportGuards(sections["guards"], &report)
	report.Result = parseReportResult(sections["result"], &report)
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

func parseObservedResults(section string, report *Report) {
	var activeList string
	for _, line := range strings.Split(section, "\n") {
		if activeList != "" {
			if value, ok := splitNestedReportBullet(line); ok {
				appendReportListValue(report, activeList, value)
				continue
			}
		}

		label, value, ok := splitTopLevelReportBullet(line)
		if !ok {
			if isTopLevelReportBullet(line) {
				activeList = ""
			}
			continue
		}
		activeList = ""
		switch observedReportField(label) {
		case "actual support":
			report.ActualSupport = cleanReportSupport(value)
			markReportField(report, "actual support", value)
		case "usage guidance":
			report.UsageGuidance = cleanReportValue(value)
			markReportField(report, "usage guidance", value)
		case "search status":
			report.SearchStatus = cleanReportStatus(value)
			markReportField(report, "search status", value)
		case "discovered ids":
			report.DiscoveredIDs = parseReportList(value)
			markReportListField(report, "discovered ids", value)
			activeList = "discovered ids"
		case "read ids":
			report.ReadIDs = parseReportList(value)
			markReportListField(report, "read ids", value)
			activeList = "read ids"
		case "cited ids":
			report.CitedIDs = parseReportList(value)
			markReportListField(report, "cited ids", value)
			activeList = "cited ids"
		case "missing needs":
			report.MissingNeeds = parseReportList(value)
			markReportListField(report, "missing needs", value)
			activeList = "missing needs"
		case "argos-backed vs general reasoning":
			markReportField(report, "argos-backed vs general reasoning", value)
		case "next steps":
			report.NextSteps = cleanReportValue(value)
			markReportField(report, "next steps", value)
		}
	}
}

func parseReportGuards(section string, report *Report) {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitReportBullet(line)
		if !ok {
			continue
		}
		if guard, ok := knownReportGuards[normalizeReportLabel(label)]; ok {
			report.Guards[guard] = cleanReportLeadingStatus(value)
			markReportField(report, guard, value)
		}
	}
}

func parseReportResult(section string, report *Report) string {
	for _, line := range strings.Split(section, "\n") {
		label, value, ok := splitReportLabel(line)
		if ok && strings.EqualFold(label, "result") {
			markReportField(report, "result", value)
			return cleanReportLeadingStatus(value)
		}
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			markReportField(report, "result", trimmed)
			return cleanReportLeadingStatus(trimmed)
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

func splitTopLevelReportBullet(line string) (string, string, bool) {
	if !isTopLevelReportBullet(line) {
		return "", "", false
	}
	return splitReportBullet(line)
}

func isTopLevelReportBullet(line string) bool {
	if strings.TrimLeft(line, " \t") != line {
		return false
	}
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")
}

func splitNestedReportBullet(line string) (string, bool) {
	if strings.TrimLeft(line, " \t") == line {
		return "", false
	}
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return strings.TrimSpace(trimmed[2:]), true
	}
	return "", false
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

func parseReportList(value string) []string {
	cleaned := cleanReportValue(value)
	if isNoneReportValue(cleaned) || cleaned == "" {
		return nil
	}
	ids := reportIDPattern.FindAllString(cleaned, -1)
	if len(ids) > 0 {
		return uniqueStrings(ids)
	}

	var values []string
	for _, part := range strings.Split(cleaned, ",") {
		item := cleanReportValue(part)
		if item != "" && !strings.EqualFold(item, "none") {
			values = append(values, item)
		}
	}
	return uniqueStrings(values)
}

func cleanReportStatus(value string) string {
	return strings.ToLower(cleanReportValue(value))
}

func cleanReportLeadingStatus(value string) string {
	cleaned := cleanReportStatus(value)
	for _, status := range []string{ResultReviewNeeded, "not-applicable", ResultPass, ResultFail, "partial"} {
		if hasReportTokenPrefix(cleaned, status) {
			return status
		}
	}
	for _, synonym := range []string{"followed", "yes", "none", "ok"} {
		if hasReportTokenPrefix(cleaned, synonym) {
			return ResultPass
		}
	}
	return cleaned
}

func cleanReportSupport(value string) string {
	cleaned := cleanReportStatus(value)
	for _, support := range []string{"strong", "partial", "weak", "inventory", "none"} {
		if containsReportToken(cleaned, support) {
			return support
		}
	}
	return cleaned
}

func cleanReportValue(value string) string {
	return strings.Trim(strings.TrimSpace(value), "` \t\r\n.,;:")
}

func normalizeReportLabel(label string) string {
	return strings.ToLower(cleanReportValue(label))
}

func isNoneReportValue(value string) bool {
	return hasReportTokenPrefix(strings.ToLower(cleanReportValue(value)), "none")
}

func hasReportTokenPrefix(value string, token string) bool {
	if !strings.HasPrefix(value, token) {
		return false
	}
	return len(value) == len(token) || isReportTokenBoundary(rune(value[len(token)]))
}

func containsReportToken(value string, token string) bool {
	start := 0
	for {
		index := strings.Index(value[start:], token)
		if index < 0 {
			return false
		}
		index += start
		beforeOK := index == 0 || isReportTokenBoundary(rune(value[index-1]))
		after := index + len(token)
		afterOK := after == len(value) || isReportTokenBoundary(rune(value[after]))
		if beforeOK && afterOK {
			return true
		}
		start = index + len(token)
	}
}

func isReportTokenBoundary(r rune) bool {
	return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-')
}

func observedReportField(label string) string {
	normalized := normalizeReportLabel(label)
	for _, field := range []string{
		"actual support",
		"usage guidance",
		"search status",
		"discovered ids",
		"read ids",
		"cited ids",
		"missing needs",
		"argos-backed vs general reasoning",
		"next steps",
	} {
		if normalized == field || strings.HasPrefix(normalized, field+" ") {
			return field
		}
	}
	return normalized
}

func appendReportListValue(report *Report, field string, value string) {
	values := parseReportList(value)
	markReportListField(report, field, value)
	switch field {
	case "discovered ids":
		report.DiscoveredIDs = uniqueStrings(append(report.DiscoveredIDs, values...))
	case "read ids":
		report.ReadIDs = uniqueStrings(append(report.ReadIDs, values...))
	case "cited ids":
		report.CitedIDs = uniqueStrings(append(report.CitedIDs, values...))
	case "missing needs":
		report.MissingNeeds = uniqueStrings(append(report.MissingNeeds, values...))
	}
}

func missingReportFields(report Report) []string {
	var missing []string
	if !report.hasField("case") {
		missing = append(missing, "case")
	}
	for _, field := range requiredReportFields {
		if !report.hasField(field) {
			missing = append(missing, field)
		}
	}
	if !report.hasField("result") {
		missing = append(missing, "result")
	}
	for _, guard := range requiredReportGuards {
		if !report.hasField(guard) {
			missing = append(missing, guard)
		}
	}
	return missing
}

func markReportField(report *Report, field string, value string) {
	if report.fieldPresence == nil {
		return
	}
	if cleanReportValue(value) != "" {
		report.fieldPresence[field] = true
	}
}

func markReportListField(report *Report, field string, value string) {
	markReportField(report, field, value)
}

func (report Report) hasField(field string) bool {
	if report.fieldPresence == nil {
		return true
	}
	return report.fieldPresence[field]
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var unique []string
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}
