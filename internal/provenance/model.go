package provenance

const (
	SchemaVersion         = "knowledge.provenance.v1"
	DecisionSchemaVersion = "knowledge.provenance.decision.v1"

	StateDraft     = "draft"
	StatePublished = "published"

	StageDesign     = "design"
	StageDraftWrite = "draft_write"
	StagePublish    = "publish"

	DecisionApproved         = "approved"
	DecisionChangesRequested = "changes_requested"
	DecisionRejected         = "rejected"
)

type Record struct {
	SchemaVersion string       `json:"schema_version"`
	ProvenanceID  string       `json:"provenance_id"`
	State         string       `json:"state"`
	Subject       Subject      `json:"subject"`
	Hashes        Hashes       `json:"hashes"`
	LatestCheck   *LatestCheck `json:"latest_check,omitempty"`
	CreatedAt     string       `json:"created_at"`
	CreatedBy     string       `json:"created_by"`
	PublishedAt   string       `json:"published_at,omitempty"`
	PublishedBy   string       `json:"published_by,omitempty"`
	PublishedFrom string       `json:"published_from,omitempty"`
	PublishedTo   string       `json:"published_to,omitempty"`
}

type Subject struct {
	Kind         string `json:"kind"`
	KnowledgeID  string `json:"knowledge_id"`
	Project      string `json:"project"`
	DesignPath   string `json:"design_path"`
	DraftPath    string `json:"draft_path"`
	OfficialPath string `json:"official_path"`
}

type Hashes struct {
	DesignSHA256      string `json:"design_sha256,omitempty"`
	DraftTreeSHA256   string `json:"draft_tree_sha256,omitempty"`
	LatestCheckSHA256 string `json:"latest_check_sha256,omitempty"`
}

type LatestCheck struct {
	Path   string `json:"path,omitempty"`
	Result string `json:"result,omitempty"`
}

type Decision struct {
	SchemaVersion string `json:"schema_version"`
	DecisionID    string `json:"decision_id"`
	Stage         string `json:"stage"`
	Decision      string `json:"decision"`
	DecidedBy     string `json:"decided_by"`
	DeciderRole   string `json:"decider_role"`
	RecordedBy    string `json:"recorded_by"`
	Source        string `json:"source"`
	Reason        string `json:"reason"`
	Hashes        Hashes `json:"hashes"`
	CreatedAt     string `json:"created_at"`
}

type DecisionInput struct {
	Stage      string
	Decision   string
	DecidedBy  string
	Role       string
	Source     string
	Reason     string
	RecordedBy string
}

type VerifyResult struct {
	Result   string   `json:"result"`
	ID       string   `json:"provenance_id"`
	Path     string   `json:"path"`
	Findings []string `json:"findings,omitempty"`
}

type StatusResult struct {
	Result         string          `json:"result"`
	ProvenanceID   string          `json:"provenance_id"`
	State          string          `json:"state"`
	Path           string          `json:"path"`
	Subject        Subject         `json:"subject"`
	Evidence       StatusEvidence  `json:"evidence"`
	ReadyToPublish bool            `json:"ready_to_publish"`
	Actions        []string        `json:"actions,omitempty"`
	Findings       []StatusFinding `json:"findings,omitempty"`
}

type StatusEvidence struct {
	DesignBound        string `json:"design_bound"`
	DraftBound         string `json:"draft_bound"`
	LatestCheck        string `json:"latest_check"`
	DesignDecision     string `json:"design_decision"`
	DraftWriteDecision string `json:"draft_write_decision"`
	PublishDecision    string `json:"publish_decision"`
	OfficialTarget     string `json:"official_target"`
}

type StatusFinding struct {
	Severity string `json:"severity"`
	Category string `json:"category"`
	Message  string `json:"message"`
}

type ListFilter struct {
	State       string
	Project     string
	KnowledgeID string
}

type ListResponse struct {
	Records []ListRecord `json:"records"`
}

type ListRecord struct {
	ProvenanceID      string `json:"provenance_id"`
	State             string `json:"state"`
	Path              string `json:"path"`
	Project           string `json:"project,omitempty"`
	KnowledgeID       string `json:"knowledge_id,omitempty"`
	Kind              string `json:"kind,omitempty"`
	DesignPath        string `json:"design_path,omitempty"`
	DraftPath         string `json:"draft_path,omitempty"`
	OfficialPath      string `json:"official_path,omitempty"`
	LatestCheckResult string `json:"latest_check_result,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	PublishedAt       string `json:"published_at,omitempty"`
}
