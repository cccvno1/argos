package knowledgewrite

const KnowledgeDesignSchemaVersion = "knowledge.design.v1"

type Finding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type DesignRequest struct {
	Project    string   `json:"project"`
	Intent     string   `json:"intent"`
	FutureTask string   `json:"future_task,omitempty"`
	Phase      string   `json:"phase,omitempty"`
	Query      string   `json:"query,omitempty"`
	Files      []string `json:"files,omitempty"`
	Domains    []string `json:"domains,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	DraftPath  string   `json:"draft_path,omitempty"`
}

type DesignResponse struct {
	Project                 string          `json:"project"`
	Intent                  string          `json:"intent"`
	Capabilities            Capabilities    `json:"capabilities"`
	Registry                RegistryFacts   `json:"registry"`
	ExistingKnowledge       ExistingMatches `json:"existing_knowledge"`
	DraftPathRisk           DraftPathRisk   `json:"draft_path_risk"`
	WritePolicy             WritePolicy     `json:"write_policy"`
	WriteGuidance           WriteGuidance   `json:"write_guidance"`
	KnowledgeDesignTemplate KnowledgeDesign `json:"knowledge_design_template"`
	NextSteps               []NextStep      `json:"next_steps"`
}

type Capabilities struct {
	Filesystem string `json:"filesystem"`
	Index      string `json:"index"`
}

type RegistryFacts struct {
	ProjectKnown    bool     `json:"project_known"`
	TechDomains     []string `json:"tech_domains"`
	BusinessDomains []string `json:"business_domains"`
}

type ExistingMatches struct {
	Official []ExistingMatch `json:"official"`
	Inbox    []ExistingMatch `json:"inbox"`
	Index    []ExistingMatch `json:"index"`
}

type ExistingMatch struct {
	Kind    string   `json:"kind"`
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Path    string   `json:"path"`
	Reasons []string `json:"reasons"`
}

type DraftPathRisk struct {
	DraftPath string `json:"draft_path"`
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
}

type WritePolicy struct {
	Write         string `json:"write"`
	OfficialWrite string `json:"official_write"`
	Publish       string `json:"publish"`
	PriorityMust  string `json:"priority_must"`
	AISuggestions string `json:"ai_suggestions"`
}

type NextStep struct {
	Step   string `json:"step"`
	Reason string `json:"reason"`
}

type WriteGuidance struct {
	State           string             `json:"state"`
	NextAction      string             `json:"next_action"`
	Reason          string             `json:"reason"`
	DesignPath      string             `json:"design_path"`
	DraftPath       string             `json:"draft_path,omitempty"`
	DraftAllowed    bool               `json:"draft_allowed"`
	DesignOnly      bool               `json:"design_only"`
	StopConditions  []string           `json:"stop_conditions"`
	DesignFocus     []string           `json:"design_focus"`
	SourceWork      SourceWorkGuidance `json:"source_work"`
	ReviewQuestions []string           `json:"review_questions"`
	Commands        WriteCommands      `json:"commands"`
}

type SourceWorkGuidance struct {
	UserInput     []string `json:"user_input,omitempty"`
	Observed      []string `json:"observed,omitempty"`
	Imported      []string `json:"imported,omitempty"`
	AISuggested   []string `json:"ai_suggested,omitempty"`
	Templates     []string `json:"templates,omitempty"`
	Examples      []string `json:"examples,omitempty"`
	Assumptions   []string `json:"assumptions,omitempty"`
	OpenQuestions []string `json:"open_questions,omitempty"`
}

type WriteCommands struct {
	WriteDesign string `json:"write_design"`
	CheckDraft  string `json:"check_draft,omitempty"`
}

type KnowledgeDesign struct {
	SchemaVersion     string                    `json:"schema_version"`
	UserRequest       string                    `json:"user_request"`
	KnowledgeGoal     string                    `json:"knowledge_goal"`
	Project           string                    `json:"project"`
	Audience          Audience                  `json:"audience"`
	Scope             Scope                     `json:"scope"`
	Sources           Sources                   `json:"sources"`
	DraftOutput       DraftOutput               `json:"draft_output"`
	FutureUse         FutureUse                 `json:"future_use"`
	Applicability     Applicability             `json:"applicability"`
	ExistingKnowledge ExistingKnowledgeDecision `json:"existing_knowledge"`
	WriteBoundary     WriteBoundary             `json:"write_boundary"`
	DraftFiles        []DraftFile               `json:"draft_files"`
	CheckPlan         CheckPlan                 `json:"check_plan"`
	Review            Review                    `json:"review"`
}

type Audience struct {
	Primary               string   `json:"primary"`
	Secondary             []string `json:"secondary,omitempty"`
	Reviewers             []string `json:"reviewers,omitempty"`
	AgentActionsSupported []string `json:"agent_actions_supported"`
}

type Scope struct {
	Projects       []string `json:"projects"`
	Stability      string   `json:"stability"`
	Distribution   string   `json:"distribution"`
	SubjectDomains []string `json:"subject_domains"`
	TechDomains    []string `json:"tech_domains"`
	Files          []string `json:"files"`
	FileGlobs      []string `json:"file_globs"`
	OutOfScope     []string `json:"out_of_scope"`
}

type Sources struct {
	UserInput     []string      `json:"user_input"`
	Observed      []string      `json:"observed"`
	Imported      []string      `json:"imported"`
	AISuggested   []string      `json:"ai_suggested"`
	Templates     []string      `json:"templates"`
	Examples      []string      `json:"examples"`
	Assumptions   []string      `json:"assumptions"`
	OpenQuestions []string      `json:"open_questions"`
	Claims        []SourceClaim `json:"claims"`
}

type SourceClaim struct {
	Claim          string   `json:"claim"`
	Kind           string   `json:"kind"`
	Source         []string `json:"source"`
	Trust          string   `json:"trust"`
	RequiresReview bool     `json:"requires_review"`
}

type DraftOutput struct {
	Kind           string `json:"kind"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	ID             string `json:"id"`
	Path           string `json:"path"`
	Status         string `json:"status"`
	Priority       string `json:"priority"`
	Rationale      string `json:"rationale"`
	EntrypointLoad string `json:"entrypoint_load"`
	DraftState     string `json:"draft_state"`
}

type FutureUse struct {
	TriggerRequests  []string `json:"trigger_requests"`
	NegativeTriggers []string `json:"negative_triggers"`
	Phases           []string `json:"phases"`
	QueryPhrases     []string `json:"query_phrases"`
	ExpectedUse      string   `json:"expected_use"`
	CitationPolicy   string   `json:"citation_policy"`
	MissingNeeds     []string `json:"missing_needs"`
}

type Applicability struct {
	WhenToUse    []string `json:"when_to_use"`
	WhenNotToUse []string `json:"when_not_to_use"`
	Tradeoffs    []string `json:"tradeoffs"`
}

type ExistingKnowledgeDecision struct {
	OfficialMatches      []string `json:"official_matches"`
	InboxMatches         []string `json:"inbox_matches"`
	PossibleMatches      []string `json:"possible_matches"`
	Decision             string   `json:"decision"`
	Reason               string   `json:"reason"`
	ReviewChoiceRequired bool     `json:"review_choice_required"`
}

type WriteBoundary struct {
	Path                        string `json:"path"`
	WriteRequiresReviewApproval bool   `json:"write_requires_review_approval"`
	PriorityMustApproved        bool   `json:"priority_must_approved"`
	PublishApproved             bool   `json:"publish_approved"`
	OfficialWriteApproved       bool   `json:"official_write_approved"`
	ReviewPacketRequired        bool   `json:"review_packet_required"`
}

type DraftFile struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
	Load    string `json:"load"`
}

type CheckPlan struct {
	ValidatePath      string                     `json:"validate_path"`
	FindabilityChecks []FindabilityCheckScenario `json:"findability_checks"`
}

type FindabilityCheckScenario struct {
	Project string   `json:"project"`
	Phase   string   `json:"phase"`
	Task    string   `json:"task"`
	Query   string   `json:"query"`
	Files   []string `json:"files"`
}

type Review struct {
	Questions             []string `json:"questions"`
	DesignApproved        bool     `json:"design_approved"`
	DraftWriteApproved    bool     `json:"draft_write_approved"`
	PriorityMustApproved  bool     `json:"priority_must_approved"`
	OfficialWriteApproved bool     `json:"official_write_approved"`
	PublishApproved       bool     `json:"publish_approved"`
	RequestedEdits        []string `json:"requested_edits,omitempty"`
	UnresolvedBlockers    []string `json:"unresolved_blockers,omitempty"`
}

type CheckRequest struct {
	DesignPath string `json:"design"`
	DraftPath  string `json:"draft"`
}

type CheckResponse struct {
	Result      string            `json:"result"`
	Design      DesignStatus      `json:"design"`
	Draft       DraftStatus       `json:"draft"`
	Policy      PolicyStatus      `json:"policy"`
	Findability FindabilityStatus `json:"findability"`
	Findings    []Finding         `json:"findings,omitempty"`
}

type DesignStatus struct {
	Path       string `json:"path"`
	Validation string `json:"validation"`
}

type DraftStatus struct {
	Path       string   `json:"path"`
	Validation string   `json:"validation"`
	IDs        []string `json:"ids,omitempty"`
}

type PolicyStatus struct {
	Result string `json:"result"`
}

type FindabilityStatus struct {
	Result string             `json:"result"`
	Checks []FindabilityCheck `json:"checks,omitempty"`
}

type FindabilityCheck struct {
	Scenario     FindabilityCheckScenario `json:"scenario"`
	Result       string                   `json:"result"`
	DraftID      string                   `json:"draft_id,omitempty"`
	MatchedDraft bool                     `json:"matched_draft"`
	MatchedIDs   []string                 `json:"matched_ids,omitempty"`
	Support      string                   `json:"support"`
}
