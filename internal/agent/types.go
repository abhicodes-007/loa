package agent

import (
	"errors"
	"fmt"
	"strings"

	"github.com/laughingmandev/loa/internal/agent/primitives"
	"github.com/laughingmandev/loa/internal/state"
	"github.com/laughingmandev/loa/internal/tools"
)


type TargetTag struct {
	StepID   int      `json:"step_id"`
	Entities []string `json:"entities"`
}

type PlanTargets struct {
	Tags []TargetTag `json:"tags"`
}

type CriterionCoverage struct {
	CriterionID uint64   `json:"criterion_id"`
	StepIDs     []uint64 `json:"step_ids"`
	Reason      string   `json:"reason"`
}

type PlanEvaluation struct {
	Valid    bool                `json:"valid"`
	Reason   string              `json:"reason"`
	Coverage []CriterionCoverage `json:"coverage"`
	Issues   []string            `json:"issues,omitempty"`
}

type StepEvaluation struct {
	Executable bool   `json:"executable"`
	Reason     string `json:"reason"`
}

type StepDecomposition struct {
	Steps  []primitives.DraftStep `json:"steps"`
	Reason string                 `json:"reason"`
}

type DiscussionDecision struct {
	Action             string         `json:"action"`
	Reason             string         `json:"reason"`
	Query              string         `json:"query,omitempty"`
	AllowedKinds       []string       `json:"allowed_kinds,omitempty"`
	Tool               *tools.Request `json:"tool,omitempty"`
	Answer             string         `json:"answer,omitempty"`
	Question           string         `json:"question,omitempty"`
	ReplanInstructions string         `json:"replan_instructions,omitempty"`
}

type ExecutionDecision struct {
	Action                string         `json:"action"`
	Reason                string         `json:"reason"`
	Query                 string         `json:"query,omitempty"`
	AllowedKinds          []string       `json:"allowed_kinds,omitempty"`
	Tool                  *tools.Request `json:"tool,omitempty"`
	AskUserQuestion       string         `json:"ask_user_question,omitempty"`
	CompletionSummary     string         `json:"completion_summary,omitempty"`
	UpdatedShortTermFacts []string       `json:"updated_short_term_facts"`
}

type ReflectionResult struct {
	Status            string   `json:"status"`
	Reason            string   `json:"reason"`
	Assumptions       []string `json:"assumptions,omitempty"`
	SuggestedRecovery string   `json:"suggested_recovery,omitempty"`
	VerificationNeed  string   `json:"verification_need,omitempty"`
}

type RouteResult struct {
	Action             string `json:"action"`
	Reason             string `json:"reason"`
	MissingInformation string `json:"missing_information,omitempty"`
}

type RecoveryDecision struct {
	Action  string `json:"action"`
	Reason  string `json:"reason"`
	Missing string `json:"missing,omitempty"`
}

type InterventionResult struct {
	Action      string   `json:"action"`
	Reason      string   `json:"reason"`
	Summary     string   `json:"summary"`
	Facts       []string `json:"facts,omitempty"`
	Decisions   []string `json:"decisions,omitempty"`
	Constraints []string `json:"constraints,omitempty"`
	OpenIssues  []string `json:"open_issues,omitempty"`
	Missing     string   `json:"missing,omitempty"`
}

type StepExtractResult struct {
	Summary      string   `json:"summary"`
	FactsLearned []string `json:"facts_learned,omitempty"`
	Decisions    []string `json:"decisions,omitempty"`
}

type MemoryExtraction struct {
	Items []MemoryDraft `json:"items"`
}

type MemoryDraft struct {
	Kind        state.MemoryKind    `json:"kind"`
	Text        string              `json:"text"`
	Participant state.MessageSource `json:"participant"`
}

type SummaryResult struct {
	Summary string `json:"summary"`
}

type FinalCheck struct {
	Complete bool                    `json:"complete"`
	Reason   string                  `json:"reason"`
	Checks   []state.AcceptanceCheck `json:"checks"`
	Missing  []string                `json:"missing,omitempty"`
}

type AskResult struct {
	Question string `json:"question"`
}

type AnswerEvaluation struct {
	Sufficient     bool   `json:"sufficient"`
	Reason         string `json:"reason"`
	FollowupNeeded string `json:"followup_needed,omitempty"`
}


type HypothesisResult struct {
	Hypothesis     string `json:"hypothesis"`
	ExpectedResult string `json:"expected_result"`
}

type CritiqueResult struct {
	Critique        string `json:"critique"`
	FalseAssumption string `json:"false_assumption"`
	NewApproach     string `json:"new_approach"`
}

type RerankResult struct {
	SynthesizedContext    string   `json:"synthesized_context,omitempty"`
	SelectedStructuralIDs []uint64 `json:"selected_structural_ids,omitempty"`
	NeedsUser             bool     `json:"needs_user"`
	ClarificationNeeded   string   `json:"clarification_needed,omitempty"`
	Reason                string   `json:"reason"`
}

func validateToolRequest(r *tools.Request) error {
	if r == nil {
		return errors.New("tool required")
	}
	if !tools.ValidKind(r.Kind) {
		return fmt.Errorf("invalid tool kind %q; allowed values: read_file, read_range, search_path, search_text, find_symbol, list_symbols, read_ast_node, write_file, patch_file, patch_ast_node, create_directory, delete_file, delete_directory, execute_process, execute_shell, git_status, git_diff, read_tool_output, analyze_large_file", r.Kind)
	}
	if len(r.Input) == 0 {
		return errors.New("tool input object is required")
	}
	return nil
}

func fillToolDescription(reason string, r *tools.Request) {
	if r == nil || strings.TrimSpace(r.Description) != "" {
		return
	}
	r.Description = strings.TrimSpace(reason)
}

func needReason(reason string) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("reason is required (needReason)")
	}
	return nil
}

func toolActionRepairHint(action string) error {
	kind := state.ToolKind(action)
	if tools.ValidKind(kind) {
		return fmt.Errorf("action %q is a tool kind, not an action; set action=\"tool\", set tool.kind=%q, and preserve the existing tool.input", action, action)
	}
	return nil
}

func validateIntent(r *state.IntentResult) error {
	if len(r.Intents) == 0 {
		return errors.New("intents must not be empty")
	}
	allowed := map[state.Intent]bool{
		state.IntentStatement:  true,
		state.IntentCorrection: true,
		state.IntentQuestion:   true,
		state.IntentDiscussion: true,
		state.IntentTask:       true,
	}
	for _, v := range r.Intents {
		if !allowed[v] {
			return fmt.Errorf("invalid intent %q; allowed values: statement, correction, question, discussion, task", v)
		}
	}
	hasTask := false
	for _, v := range r.Intents {
		if v == state.IntentTask {
			hasTask = true
			break
		}
	}
	if r.MayModify && !hasTask {
		return errors.New("may_modify requires task intent")
	}
	if r.MayModify && !r.MayInvestigate {
		return errors.New("may_modify requires may_investigate")
	}
	if r.Ambiguous && r.MayModify {
		return errors.New("ambiguous intent cannot authorize modification before clarification")
	}
	return needReason(r.Reason)
}





func validatePlanEvaluation(v *PlanEvaluation, criteria []state.AcceptanceCriterion, plan state.Plan) error {
	if err := needReason(v.Reason); err != nil {
		return err
	}

	criteriaByID := make(map[uint64]state.AcceptanceCriterion, len(criteria))
	for _, criterion := range criteria {
		criteriaByID[criterion.ID] = criterion
	}
	stepsByID := make(map[uint64]state.PlanStep, len(plan.Steps))
	for _, step := range plan.Steps {
		stepsByID[step.ID] = step
	}

	seenCriteria := make(map[uint64]bool, len(v.Coverage))
	for i, coverage := range v.Coverage {
		if _, ok := criteriaByID[coverage.CriterionID]; !ok {
			return fmt.Errorf("coverage entry %d references unknown acceptance criterion id %d", i+1, coverage.CriterionID)
		}
		if seenCriteria[coverage.CriterionID] {
			return fmt.Errorf("acceptance criterion id %d appears more than once in coverage", coverage.CriterionID)
		}
		seenCriteria[coverage.CriterionID] = true
		if err := needReason(coverage.Reason); err != nil {
			return fmt.Errorf("coverage for acceptance criterion id %d requires a reason", coverage.CriterionID)
		}

		seenSteps := map[uint64]bool{}
		for _, stepID := range coverage.StepIDs {
			if _, ok := stepsByID[stepID]; !ok {
				return fmt.Errorf("coverage for acceptance criterion id %d references unknown plan step id %d", coverage.CriterionID, stepID)
			}
			if seenSteps[stepID] {
				return fmt.Errorf("coverage for acceptance criterion id %d repeats plan step id %d", coverage.CriterionID, stepID)
			}
			seenSteps[stepID] = true
		}
	}

	for _, criterion := range criteria {
		var mapped *CriterionCoverage
		for i := range v.Coverage {
			if v.Coverage[i].CriterionID == criterion.ID {
				mapped = &v.Coverage[i]
				break
			}
		}
		if mapped == nil {
			return fmt.Errorf("plan evaluation is missing coverage for acceptance criterion id %d; include every supplied criterion, using an empty step_ids array when it is uncovered", criterion.ID)
		}
		if v.Valid && len(mapped.StepIDs) == 0 {
			return fmt.Errorf("valid plan leaves acceptance criterion id %d uncovered; provide one or more actual plan step ids or set valid=false", criterion.ID)
		}
	}
	return nil
}
func validateStepEvaluation(v *StepEvaluation) error { return needReason(v.Reason) }

func validateDiscussion(d *DiscussionDecision) error {
	if err := needReason(d.Reason); err != nil {
		return err
	}
	switch d.Action {
	case "retrieve_memory":
		if strings.TrimSpace(d.Query) == "" {
			return errors.New("query required")
		}
	case "task_summaries":
	case "tool":
		fillToolDescription(d.Reason, d.Tool)
		if err := validateToolRequest(d.Tool); err != nil {
			return err
		}
	case "ask_user":
		if strings.TrimSpace(d.Question) == "" {
			return errors.New("question required")
		}
	case "final":
		if strings.TrimSpace(d.Answer) == "" {
			return errors.New("answer required")
		}
	default:
		if err := toolActionRepairHint(d.Action); err != nil {
			return err
		}
		return fmt.Errorf("invalid discussion action %q; allowed values: retrieve_memory, task_summaries, tool, ask_user, final", d.Action)
	}
	return nil
}

func validateNegotiation(d *DiscussionDecision) error {
	if d.Action == "approve" {
		return needReason(d.Reason)
	}
	if d.Action == "replan" {
		if err := needReason(d.Reason); err != nil {
			return err
		}
		if strings.TrimSpace(d.ReplanInstructions) == "" {
			return errors.New("replan_instructions required for replan action")
		}
		return nil
	}
	err := validateDiscussion(d)
	if err != nil && strings.Contains(err.Error(), "invalid discussion action") {
		return fmt.Errorf("invalid negotiation action %q; allowed values: retrieve_memory, task_summaries, tool, ask_user, final, approve, replan", d.Action)
	}
	return err
}

func validateExecution(d *ExecutionDecision) error {
	if err := needReason(d.Reason); err != nil {
		return err
	}
	switch d.Action {
	case "retrieve_memory":
		if strings.TrimSpace(d.Query) == "" {
			return errors.New("query required")
		}
	case "task_summaries":
	case "tool":
		fillToolDescription(d.Reason, d.Tool)
		if err := validateToolRequest(d.Tool); err != nil {
			return err
		}
	case "ask_user":
		if strings.TrimSpace(d.AskUserQuestion) == "" {
			return errors.New("ask_user_question required")
		}
	case "step_complete":
		if strings.TrimSpace(d.CompletionSummary) == "" {
			return errors.New("completion_summary required")
		}
	default:
		if err := toolActionRepairHint(d.Action); err != nil {
			return err
		}
		return fmt.Errorf("invalid execution action %q; allowed values: retrieve_memory, task_summaries, tool, ask_user, step_complete", d.Action)
	}
	if d.UpdatedShortTermFacts == nil {
		return errors.New("updated_short_term_facts array is required (may be empty)")
	}
	return nil
}

func validateReflection(r *ReflectionResult) error {
	if err := needReason(r.Reason); err != nil {
		return err
	}
	switch r.Status {
	case "success", "partial", "failed", "needs_more_information", "plan_invalid", "verification_required":
	default:
		return fmt.Errorf("invalid reflection status %q; allowed values: success, partial, failed, needs_more_information, plan_invalid, verification_required", r.Status)
	}
	if r.SuggestedRecovery != "" {
		switch r.SuggestedRecovery {
		case "retry", "investigate", "repair", "replan", "ask_user":
		default:
			return fmt.Errorf("invalid suggested_recovery %q; allowed values: retry, investigate, repair, replan, ask_user, or empty string", r.SuggestedRecovery)
		}
	}
	return nil
}

func validateRoute(r *RouteResult) error {
	if err := needReason(r.Reason); err != nil {
		return err
	}
	switch r.Action {
	case "retry", "investigate", "repair", "replan", "rollback", "ask_user":
	default:
		return fmt.Errorf("invalid route action %q; allowed values: retry, investigate, repair, replan, rollback, ask_user", r.Action)
	}
	return nil
}

func validateIntervention(v *InterventionResult) error {
	if err := needReason(v.Reason); err != nil {
		return err
	}
	if strings.TrimSpace(v.Summary) == "" {
		return errors.New("summary is required")
	}
	switch v.Action {
	case "context_only", "reconsider_current_step", "replan_remaining", "ask_user":
	default:
		return fmt.Errorf("invalid intervention action %q; allowed values: context_only, reconsider_current_step, replan_remaining, ask_user", v.Action)
	}
	if v.Action == "ask_user" && strings.TrimSpace(v.Missing) == "" {
		return errors.New("missing is required when intervention action is ask_user")
	}
	for _, group := range [][]string{v.Facts, v.Decisions, v.Constraints, v.OpenIssues} {
		for _, item := range group {
			if strings.TrimSpace(item) == "" {
				return errors.New("intervention context updates must not contain empty entries")
			}
		}
	}
	return nil
}

func validateRecovery(r *RecoveryDecision) error {
	if err := needReason(r.Reason); err != nil {
		return err
	}
	switch r.Action {
	case "continue", "retry", "replan", "ask_user", "start_new":
	default:
		return fmt.Errorf("invalid recovery action %q; allowed values: continue, retry, replan, ask_user, start_new", r.Action)
	}
	if r.Action == "ask_user" && strings.TrimSpace(r.Missing) == "" {
		return errors.New("missing is required when recovery action is ask_user")
	}
	return nil
}

func validateMemory(m *MemoryExtraction) error {
	for i := range m.Items {
		switch m.Items[i].Kind {
		case state.MemoryFact, state.MemoryDecision, state.MemoryApproval, state.MemoryStepResult, state.MemoryStructural, state.MemoryGlobalConcept:
		default:
			return fmt.Errorf("invalid memory kind %q; allowed values: fact, decision, approval, step_result, structural, global_concept", m.Items[i].Kind)
		}
		if strings.TrimSpace(m.Items[i].Text) == "" {
			return errors.New("memory text required")
		}
	}
	return nil
}

func validateStepExtract(v *StepExtractResult) error {
	if strings.TrimSpace(v.Summary) == "" {
		return errors.New("summary required")
	}
	return nil
}

func validateTaskContext(v *state.TaskContext) error {
	for _, group := range [][]string{v.Facts, v.Decisions, v.Constraints, v.OpenIssues} {
		for _, item := range group {
			if strings.TrimSpace(item) == "" {
				return errors.New("task context entries must not be empty")
			}
		}
	}
	return nil
}

func validateFinalCheck(v *FinalCheck, criteria []state.AcceptanceCriterion) error {
	if err := needReason(v.Reason); err != nil {
		return err
	}
	if len(v.Checks) != len(criteria) {
		return fmt.Errorf("checks must contain exactly one entry for each acceptance criterion; expected %d, got %d", len(criteria), len(v.Checks))
	}
	valid := map[uint64]bool{}
	for _, criterion := range criteria {
		valid[criterion.ID] = true
	}
	seen := map[uint64]bool{}
	allVerified := true
	for _, check := range v.Checks {
		if !valid[check.CriterionID] {
			return fmt.Errorf("unknown criterion_id %d", check.CriterionID)
		}
		if seen[check.CriterionID] {
			return fmt.Errorf("duplicate criterion_id %d", check.CriterionID)
		}
		seen[check.CriterionID] = true
		switch check.Status {
		case state.AcceptanceVerified:
			if len(check.Evidence) == 0 {
				return fmt.Errorf("criterion %d marked verified without evidence", check.CriterionID)
			}
		case state.AcceptancePartial, state.AcceptanceUnverified:
			allVerified = false
			if strings.TrimSpace(check.Missing) == "" {
				return fmt.Errorf("criterion %d status %q requires missing explanation", check.CriterionID, check.Status)
			}
		default:
			return fmt.Errorf("invalid acceptance status %q; allowed values: verified, partial, unverified", check.Status)
		}
	}
	if v.Complete && !allVerified {
		return errors.New("complete=true requires every acceptance criterion to be verified")
	}
	if !v.Complete && allVerified {
		return errors.New("complete=false conflicts with all acceptance criteria being verified")
	}
	return nil
}

func validateAnswerEvaluation(v *AnswerEvaluation) error { return needReason(v.Reason) }


func validateRerank(rr *RerankResult, valid map[uint64]bool, limit int) error {
	if rr.NeedsUser && rr.ClarificationNeeded == "" {
		return errors.New("must provide clarification_needed if needs_user is true")
	}
	if !rr.NeedsUser && rr.SynthesizedContext == "" && len(rr.SelectedStructuralIDs) == 0 {
		return errors.New("must provide synthesized_context or selected_structural_ids if needs_user is false")
	}
	if len(rr.SelectedStructuralIDs) > limit {
		return errors.New("selected_structural_ids exceeds requested limit")
	}
	for _, id := range rr.SelectedStructuralIDs {
		if !valid[id] {
			return errors.New("selected_structural_ids contains invalid candidate ID")
		}
	}
	return nil
}

func validateHypothesis(v *HypothesisResult) error {
	if strings.TrimSpace(v.Hypothesis) == "" {
		return errors.New("hypothesis required")
	}
	if strings.TrimSpace(v.ExpectedResult) == "" {
		return errors.New("expected_result required")
	}
	return nil
}

func validateCritique(v *CritiqueResult) error {
	if strings.TrimSpace(v.Critique) == "" {
		return errors.New("critique required")
	}
	if strings.TrimSpace(v.FalseAssumption) == "" {
		return errors.New("false_assumption required")
	}
	if strings.TrimSpace(v.NewApproach) == "" {
		return errors.New("new_approach required")
	}
	return nil
}
