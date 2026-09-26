package primitives

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/laughingmandev/loa/internal/config"
	"github.com/laughingmandev/loa/internal/llm"
	"github.com/laughingmandev/loa/internal/state"
	"github.com/laughingmandev/loa/internal/tools"
)

var executePrompt = `Work on exactly the current plan step. You may request one action at a time: a tool call, semantic memory retrieval, the past-task-summary list, clarification from the user, or declare the step complete. Prefer compact symbol/range reads over huge files when practical, but use whole files when appropriate. Do not modify files unless modification is allowed. TASK MODIFICATION AUTHORIZATION in active task state is a hard boundary: may_modify=false means strictly read-only. Furthermore, if the CURRENT STEP mode is "investigate" or "verify", modifying tools are STRICTLY FORBIDDEN even if the task overall allows modification. If you discover a bug during a verify step, do NOT attempt to fix it with a tool; instead, return step_complete with a failed outcome so a repair step can be scheduled.
If encountering an unknown class/file, you MUST return retrieve_memory.
Do not use raw text search or path guessing to understand architecture. You MUST follow the pipeline: Retrieve Semantic Memory -> Identify Targets -> Use find_symbol
For retrieve_memory, you may optionally include "allowed_kinds" (e.g. ["task_summary", "fact", "structural"]) to strictly filter memory results to specific kinds.
Available tool schemas are provided below.
Return exactly one of:
{"action":"tool","reason":"short","tool":{"kind":"execute_process","input":{}}}
{"action":"retrieve_memory","reason":"short","query":"semantic search phrase","allowed_kinds":["task_summary","fact"]}
{"action":"task_summaries","reason":"short"}
{"action":"ask_user","reason":"short","ask_user_question":"what cannot be determined locally"}
{"action":"step_complete","reason":"short","completion_summary":"what was accomplished"}`


type ExecutionDecision struct {
	Action                string         `json:"action"`
	Reason                string         `json:"reason"`
	Query                 string         `json:"query,omitempty"`
	AllowedKinds          []string       `json:"allowed_kinds,omitempty"`
	Tool                  *tools.Request `json:"tool,omitempty"`
	AskUserQuestion       string         `json:"ask_user_question,omitempty"`
	CompletionSummary     string         `json:"completion_summary,omitempty"`
}

type ExecuteStepInput struct {
	Step   state.PlanStep
	Extras []string
}

type ExecuteStepPrimitive struct {
	cfg        config.Config
	llm        llm.Client
	baseSystem string
}

func NewExecuteStepPrimitive(cfg config.Config, llmClient llm.Client, baseSystem string) *ExecuteStepPrimitive {
	return &ExecuteStepPrimitive{
		cfg:        cfg,
		llm:        llmClient,
		baseSystem: baseSystem,
	}
}

func (p *ExecuteStepPrimitive) Execute(ctx context.Context, pc PrimitiveContext, in ExecuteStepInput) (ExecutionDecision, error) {
	stepJSONBytes, _ := json.Marshal(in.Step)
	purpose := fmt.Sprintf("CURRENT STEP:\n%s\n\nAVAILABLE TOOLS:\n%s", string(stepJSONBytes), tools.ToolSchema())

	ctxText := pc.Build(&in.Step, in.Extras, true)
	user := ctxText + "\nPRIMITIVE INSTRUCTIONS:\n" + executePrompt + "\n\nPRIMITIVE INPUT:\n" + purpose

	var out ExecutionDecision
	attempts, err := p.llm.ChatJSON(ctx, p.cfg.ModelExecuting, p.baseSystem, user, &out, func() error {
		return ValidateExecution(&out)
	})

	for i := range attempts {
		if i == 0 {
			pc.RecordInference("ExecuteStep", false)
		} else {
			pc.RecordInference("ExecuteStep", true)
		}
	}

	for _, a := range attempts {
		kind := state.LogLLM
		if a.PromptKind == "repair" {
			kind = state.LogRepair
		}
		pc.Log(kind, "ExecuteStep", "execute step", a.Prompt, a.Raw)
	}

	return out, err
}

func ValidateExecution(d *ExecutionDecision) error {
	if err := NeedReason(d.Reason); err != nil {
		return err
	}
	switch d.Action {
	case "retrieve_memory":
		if d.Query == "" {
			return fmt.Errorf("query required")
		}
	case "task_summaries":
	case "tool":
		FillToolDescription(d.Reason, d.Tool)
		if err := ValidateToolRequest(d.Tool); err != nil {
			return err
		}
	case "ask_user":
		if d.AskUserQuestion == "" {
			return fmt.Errorf("ask_user_question required")
		}
	case "step_complete":
		if d.CompletionSummary == "" {
			return fmt.Errorf("completion_summary required")
		}
	default:
		if err := ToolActionRepairHint(d.Action); err != nil {
			return err
		}
		return fmt.Errorf("invalid execution action %q; allowed values: retrieve_memory, task_summaries, tool, ask_user, step_complete", d.Action)
	}
	return nil
}
