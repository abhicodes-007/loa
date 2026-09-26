package primitives

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/laughingmandev/loa/internal/config"
	"github.com/laughingmandev/loa/internal/llm"
	"github.com/laughingmandev/loa/internal/state"
)

type MemoryDraft struct {
	Kind        state.MemoryKind    `json:"kind"`
	Text        string              `json:"text"`
	Participant state.MessageSource `json:"participant"`
}

type MemoryExtraction struct {
	Items []MemoryDraft `json:"items"`
}

type ExtractDurableMemoryInput struct {
	Purpose string
}

type ExtractDurableMemoryPrimitive struct {
	cfg        config.Config
	llm        llm.Client
	baseSystem string
}

func NewExtractDurableMemoryPrimitive(cfg config.Config, llmClient llm.Client, baseSystem string) *ExtractDurableMemoryPrimitive {
	return &ExtractDurableMemoryPrimitive{
		cfg:        cfg,
		llm:        llmClient,
		baseSystem: baseSystem,
	}
}

var ExtractDurableMemoryPrompt = `From the recent interaction, extract only durable facts, decisions, corrections, or user approvals that are likely useful later in this session. Interpret short replies in context (for example "okay" may mean approval of a prior proposal). Do not store pleasantries or transient chatter.
Do NOT extract the user's immediate task instructions or commands (e.g. "analyze this file", "fix this bug"). Only extract persistent user constraints, preferences, or environment details.
Allowed kind values are exactly: fact, decision, approval, step_result.
Return {"items":[{"kind":"fact","text":"self-contained statement"}]}. An empty items list is valid.`

func ValidateMemory(m *MemoryExtraction) error {
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

func (p *ExtractDurableMemoryPrimitive) Execute(ctx context.Context, pc PrimitiveContext, in ExtractDurableMemoryInput) (MemoryExtraction, error) {
	ctxText := pc.Build(nil, nil, false)
	user := ctxText + "\nPRIMITIVE INSTRUCTIONS:\n" + ExtractDurableMemoryPrompt + "\n\nPRIMITIVE INPUT:\n" + in.Purpose

	var out MemoryExtraction
	attempts, err := p.llm.ChatJSON(ctx, p.cfg.ModelCrawling, p.baseSystem, user, &out, func() error {
		return ValidateMemory(&out)
	})

	for i := range attempts {
		if i == 0 {
			pc.RecordInference("ExtractDurableMemory", false)
		} else {
			pc.RecordInference("ExtractDurableMemory", true)
		}
	}

	for _, a := range attempts {
		kind := state.LogLLM
		if a.PromptKind == "repair" {
			kind = state.LogRepair
		}
		pc.Log(kind, "ExtractDurableMemory", "extract durable memory", user, a.Raw)
	}

	return out, err
}
