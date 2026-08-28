package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/voocel/agentcore/schema"
	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/errs"
	"github.com/voocel/ainovel-cli/internal/llmcontract"
	"github.com/voocel/ainovel-cli/internal/store"
)

// AuditFoundationTool 接收 Architect 对已落盘基础设定的语义审查结论。
// 文学与跨文件语义由模型判断；工具只保证审查版本、结论和状态迁移一致。
type AuditFoundationTool struct {
	store *store.Store
}

func NewAuditFoundationTool(store *store.Store) *AuditFoundationTool {
	return &AuditFoundationTool{store: store}
}

func (t *AuditFoundationTool) Name() string { return "audit_foundation" }
func (t *AuditFoundationTool) Description() string {
	return "Audit whether the persisted book, premise, outline, characters, world_rules and compass are semantically consistent with one another. " +
		"You must call novel_context again first and pass foundation_status.fingerprint through verbatim."
}
func (t *AuditFoundationTool) Label() string                          { return "Audit foundation" }
func (t *AuditFoundationTool) ReadOnly(_ json.RawMessage) bool        { return false }
func (t *AuditFoundationTool) ConcurrencySafe(_ json.RawMessage) bool { return false }
func (t *AuditFoundationTool) StrictSchema() bool                     { return true }

func (t *AuditFoundationTool) Schema() map[string]any {
	issue := schema.Object(
		schema.Property("artifact", schema.String("The artifact with the problem, e.g. book/premise/characters/layered_outline/world_rules/compass")).Required(),
		schema.Property("description", schema.String("The cross-file semantic problem")).Required(),
		schema.Property("evidence", schema.String("Concrete evidence of the conflict, quoted from the persisted content")).Required(),
		schema.Property("suggestion", llmcontract.Nullable(schema.String("Suggested direction for the fix; null when no suggestion is needed"))).Required(),
	)
	return schema.Object(
		schema.Property("fingerprint", schema.String("The foundation_status.fingerprint returned by novel_context")).Required(),
		schema.Property("ready", schema.Bool("Whether the whole foundation is now semantically consistent and writing can begin")).Required(),
		schema.Property("summary", schema.String("Summary of the audit conclusion")).Required(),
		schema.Property("issues", schema.Array("Cross-file semantic problems found; an empty array when ready=true", issue)).Required(),
	)
}

func (t *AuditFoundationTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var audit domain.FoundationAudit
	if err := json.Unmarshal(args, &audit); err != nil {
		return nil, fmt.Errorf("invalid args: %w: %w", errs.ErrToolArgs, err)
	}
	if strings.TrimSpace(audit.Fingerprint) == "" {
		return nil, fmt.Errorf("fingerprint is required: %w", errs.ErrToolArgs)
	}
	if strings.TrimSpace(audit.Summary) == "" {
		return nil, fmt.Errorf("summary is required: %w", errs.ErrToolArgs)
	}

	missing, err := t.store.FoundationMissing()
	if err != nil {
		return nil, fmt.Errorf("load foundation state: %w: %w", errs.ErrStoreRead, err)
	}
	for _, item := range missing {
		if item != "foundation_audit" {
			return nil, fmt.Errorf("the foundation is still missing %s, so it cannot be audited: %w", item, errs.ErrToolPrecondition)
		}
	}
	current, err := t.store.FoundationFingerprint()
	if err != nil {
		return nil, fmt.Errorf("fingerprint foundation: %w: %w", errs.ErrStoreRead, err)
	}
	if audit.Fingerprint != current {
		return nil, fmt.Errorf("the foundation has changed; call novel_context again for the latest fingerprint before auditing: %w", errs.ErrToolConflict)
	}
	if audit.Ready && len(audit.Issues) > 0 {
		return nil, fmt.Errorf("issues must be empty when ready=true: %w", errs.ErrToolArgs)
	}
	if !audit.Ready && len(audit.Issues) == 0 {
		return nil, fmt.Errorf("concrete issues are required when ready=false: %w", errs.ErrToolArgs)
	}
	for i, issue := range audit.Issues {
		if strings.TrimSpace(issue.Artifact) == "" || strings.TrimSpace(issue.Description) == "" || strings.TrimSpace(issue.Evidence) == "" {
			return nil, fmt.Errorf("issues[%d] must contain artifact, description and evidence: %w", i, errs.ErrToolArgs)
		}
	}

	if err := t.store.Outline.SaveFoundationAudit(audit); err != nil {
		return nil, fmt.Errorf("save foundation audit: %w: %w", errs.ErrStoreWrite, err)
	}
	result := map[string]any{
		"foundation_ready": audit.Ready,
		"issues":           audit.Issues,
	}
	if !audit.Ready {
		result["next_action"] = "fix the corresponding foundation artifacts per the issues, then call novel_context again and re-audit"
		return json.Marshal(result)
	}

	if _, err := t.store.Checkpoints.AppendArtifact(domain.GlobalScope(), "foundation_audit", "meta/foundation_audit.json"); err != nil {
		return nil, fmt.Errorf("checkpoint foundation audit: %w: %w", errs.ErrStoreWrite, err)
	}
	if err := t.store.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		return nil, fmt.Errorf("enter writing phase: %w: %w", errs.ErrStoreWrite, err)
	}
	result["phase"] = string(domain.PhaseWriting)
	return json.Marshal(result)
}
