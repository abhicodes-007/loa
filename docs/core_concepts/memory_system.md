# Memory System

Loa employs a strict hierarchical memory architecture, completely contained within your local workspace inside the `.loa/` directory. It does not rely on cloud vector databases or one global RAG bucket, guaranteeing zero cross-project leakage and 100% offline functionality.

## Memory Scopes

Memory is divided into two distinct implementations managed inside `internal/memory/store.go`.

### 1. Session Memory (Working Context)
- **Path:** `.loa/sessions/<session_id>.json`
- **Scope:** Contains the immediate state of the active task, the DAG plan, recent tool outputs, and short-term facts discovered during the current execution.
- **Decay Configuration:** Configured with an aggressive half-life (`2.0 hours`) and a strong context weight (`0.20`).
- **Function:** Ensures the LLM retains immediate, short-term focus on the active execution step. If a session is abandoned, its relevance rapidly decays to prevent polluting future tasks.

### 2. Project Memory (Durable Storage)
- **Path:** `.loa/project_memory.json`
- **Scope:** Persistent key-value and semantic data containing overarching project rules, architectural decisions, discovered bugs, and user preferences.
- **Decay Configuration:** Configured with a slow, Long-Term Support (LTS) decay (`14 days` half-life) and a lighter nudge weight (`0.03`).
- **Function:** Provides subtle, long-term context that nudges the LLM toward project-specific conventions without overwhelming the prompt token limits.

## Memory Extraction & Reflection Lifecycle

Memory does not spontaneously migrate; it follows a strict lifecycle tied to the execution engine. **Context is reconstructed, not accumulated.** Loa does not drag the entire conversation history forever. It surgically composes the context needed for the current inference from relevant original material.

1. **Working Context (Injection):** When a task begins, the engine retrieves relevant nodes from `project_memory.json` using `PrimitiveExtractKeywords` and `PrimitiveRerankMemory` and injects them into the LLM system prompt.
2. **Execution (Mutation):** As `PrimitiveExecuteStep` runs tools, the outputs (stdout, stderr, diffs) are temporarily held in Session Memory.
3. **Reflection (Distillation):** Upon completing a plan step, the engine invokes `PrimitiveEvaluateStepResult` and `PrimitiveExtractStepResult`. These primitives compress massive tool outputs into dense summaries, preferring original evidence over lossy context summarization.
4. **Extraction (Semantic Promotion):** During the transition between steps or at the end of a task, `PrimitiveExtractDurableMemory` scans the session history. Working memory is not simply dumped wholesale into longer-term memory. Loa consolidates useful semantic state (hard architectural constraints or recurring bugs) and discards episodic execution noise, persisting it to `.loa/project_memory.json`.

## Token Limits and Context Sliding

If an execution loop generates excessive output (e.g., an uncontrolled `cat` on a massive log file or a massive test failure), the Session Memory could breach the model's token limits. 

To prevent catastrophic prompt overflow, the engine employs a deterministic technique called **Context Sliding**. 

### How it Works
When preparing the LLM payload, the engine calculates the remaining token budget against the `ContextBudget` defined in your settings. 
If the required context (active task state, tool outputs, project memories) exceeds this limit, the engine does not panic or trigger LLM re-summarization loops. Instead, it deterministically enforces the limit:

1. **Mandatory Context is Preserved:** The Active Task state, current step definition, and explicit user guidance are always preserved.
2. **Chat History is Trimmed:** It drops the oldest chat messages sequentially until it fits within the budget.
3. **Extras are Truncated:** It strictly truncates the length of massive tool outputs or retrieved memory chunks (`extraParts`) while preserving as much of the output as possible within the `CodeBudget`.

By mathematically sliding the context window, the execution can safely continue with a renewed token budget while naturally dropping the verbose history of early, completed steps.
