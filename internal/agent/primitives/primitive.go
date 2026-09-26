package primitives

import (
	"context"

	"github.com/laughingmandev/loa/internal/state"
)

// PrimitiveContext provides access to engine-level state, context building, and telemetry.
type PrimitiveContext interface {
	Build(step *state.PlanStep, extras []string, includeInstructions bool) string
	RecordInference(primitive string, isRepair bool)
	Log(kind state.LogKind, primitive, summary, prompt, response string)
}

// Primitive defines a standardized node in the execution pipeline.
type Primitive[I any, O any] interface {
	Execute(ctx context.Context, pc PrimitiveContext, in I) (O, error)
}
