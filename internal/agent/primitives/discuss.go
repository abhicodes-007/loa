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

var discussPrompt = `Answer or discuss the user's request without making project modifications. You may investigate first using semantic memory, task summaries, or read-oriented tools. Arbitrary command execution can be requested when genuinely useful but is controlled by runtime permissions. Never use file-mutating tools in discussion mode.
Allowed action values are exactly: tool, retrieve_memory, task_summaries, ask_user, final.
Allowed tool.kind values are exactly: ` + ToolKindEnum + `.
For tool.kind use Loa tool kinds only. Executables such as "go", "git", "docker", or "bash" belong inside execute_process/execute_shell input, never in tool.kind.
tool.description is optional. Do not duplicate reason just to fill it; when omitted, Loa will use the action reason as the tool description.
For retrieve_memory, you may optionally include "allowed_kinds" (e.g. ["task_summary", "fact", "structural"]) to strictly filter memory results to specific kinds.
Return exactly one of:
{"action":"tool","reason":"short","tool":{"kind":"read_file","input":{}}}
{"action":"retrieve_memory","reason":"short","query":"semantic search phrase","allowed_kinds":["task_summary","fact"]}
{"action":"task_summaries","reason":"short"}
{"action":"ask_user","reason":"short","question":"clarification"}
{"action":"final","reason":"short","answer":"user-facing answer"}`

type DiscussionResult struct {
	Action             string         `json:"action"`
	Reason             string         `json:"reason"`
	Query              string         `json:"query,omitempty"`
	AllowedKinds       []string       `json:"allowed_kinds,omitempty"`
	Tool               *tools.Request `json:"tool,omitempty"`
	Answer             string         `json:"answer,omitempty"`
	Question           string         `json:"question,omitempty"`
	ReplanInstructions string         `json:"replan_instructions,omitempty"`
}

type DiscussInput struct {
	UserText string
	Intent   state.IntentResult
	Extras   []string
}

type DiscussPrimitive struct {
	cfg        config.Config
	llm        llm.Client
	baseSystem string
}

func NewDiscussPrimitive(cfg config.Config, llmClient llm.Client, baseSystem string) *DiscussPrimitive {
	return &DiscussPrimitive{
		cfg:        cfg,
		llm:        llmClient,
		baseSystem: baseSystem,
	}
}

func (p *DiscussPrimitive) Execute(ctx context.Context, pc PrimitiveContext, in DiscussInput) (DiscussionResult, error) {
	intentJSONBytes, _ := json.Marshal(in.Intent)
	intentJSON := string(intentJSONBytes)
	purpose := fmt.Sprintf("USER REQUEST:\n%s\n\nINTENT:\n%s\n\nAVAILABLE TOOLS:\n%s", in.UserText, intentJSON, tools.ToolSchema())

	ctxText := pc.Build(nil, in.Extras, true)
	user := ctxText + "\nPRIMITIVE INSTRUCTIONS:\n" + discussPrompt + "\n\nPRIMITIVE INPUT:\n" + purpose

	var out DiscussionResult
	attempts, err := p.llm.ChatJSON(ctx, p.cfg.ModelConversation, p.baseSystem, user, &out, func() error {
		return validateDiscussion(&out)
	})

	for i := range attempts {
		if i == 0 {
			pc.RecordInference("Discuss", false)
		} else {
			pc.RecordInference("Discuss", true)
		}
	}

	for _, a := range attempts {
		kind := state.LogLLM
		if a.PromptKind == "repair" {
			kind = state.LogRepair
		}
		pc.Log(kind, "Discuss", "discuss user request", user, a.Raw)
	}

	return out, err
}

func validateDiscussion(d *DiscussionResult) error {
	if d.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	validActions := map[string]bool{
		"tool":            true,
		"retrieve_memory": true,
		"task_summaries":  true,
		"ask_user":        true,
		"final":           true,
		"approve":         true, // Allowed for negotiation
		"replan":          true, // Allowed for negotiation
	}
	if !validActions[d.Action] {
		return fmt.Errorf("invalid action %q", d.Action)
	}
	switch d.Action {
	case "tool":
		if d.Tool == nil || d.Tool.Kind == "" {
			return fmt.Errorf("tool.kind is required for tool action")
		}
	case "retrieve_memory":
		if d.Query == "" {
			return fmt.Errorf("query is required for retrieve_memory action")
		}
	case "ask_user":
		if d.Question == "" {
			return fmt.Errorf("question is required for ask_user action")
		}
	case "final":
		if d.Answer == "" {
			return fmt.Errorf("answer is required for final action")
		}
	case "replan":
		if d.ReplanInstructions == "" {
			return fmt.Errorf("replan_instructions required for replan action")
		}
	}
	return nil
}
