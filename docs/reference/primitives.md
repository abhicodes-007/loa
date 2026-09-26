# Primitives Reference

Primitives are the fundamental state transitions within the Loa Engine. Every action, whether it is calling the LLM or executing a shell command, is encapsulated within a Primitive in `internal/agent/primitives/`. 

This reference details all core primitives used by the engine.

## 1. Setup & Intent

- **`PrimitiveIntent`**: Analyzes the initial user prompt to classify the user's intent. Output includes boolean flags like `may_investigate` and `may_modify`.
- **`PrimitiveResolveAmbiguity`**: Detects underspecified requirements in the user's prompt before generating a plan.
- **`PrimitiveNarrativeFrame`**: Constructs the persona/context framing for the system prompt based on the project's state.

## 2. Planning & Orchestration

- **`PrimitivePlan`**: Generates the structured Directed Acyclic Graph (DAG) implementation plan consisting of specific, actionable steps.
- **`PrimitiveDecomposeStep`**: Breaks down a single, complex step into smaller, manageable sub-steps if it is deemed too broad for a single LLM execution cycle.
- **`PrimitiveEvaluatePlan`**: Critiques the generated DAG plan to ensure it covers all acceptance criteria before execution begins.
- **`PrimitiveUpdateCriteria`**: Dynamically updates the acceptance criteria of a task based on newly discovered constraints during planning.
- **`PrimitiveCriteriaDependency`**: Maps specific acceptance criteria to the individual plan steps responsible for fulfilling them.

## 3. Execution & Tooling

- **`PrimitiveTagTargets`**: Identifies and tags specific files or symbols that need to be targeted by the upcoming execution, priming the context.
- **`PrimitiveExecuteStep`**: The primary deterministic engine state. Takes the `tool` payload from a previous step, maps it to the Tool Manager, executes the binary or filesystem operation, and captures `stdout`/`stderr`.
- **`PrimitiveGenerateStepArtifact`**: Produces detailed markdown artifacts summarizing the outputs or decisions made during an execution step.

## 4. Evaluation & Reflection

- **`PrimitiveEvaluateStepResult`**: Invoked immediately after `PrimitiveExecuteStep`. Reflects on the `stdout`/`stderr` or file diff to verify if the specific tool call succeeded.
- **`PrimitiveEvaluateStep`**: Assesses whether the overarching goal of a specific plan step has been fully met, allowing the engine to progress to the next step.
- **`PrimitiveFinalTaskEvaluation`**: The final verification gate before marking a complete modifying task as `DONE`.
- **`PrimitiveAssessTaskRecovery`**: Triggered when a step fails continuously. Determines if the task can be salvaged by replanning or if it requires a hard abort.
- **`PrimitiveCritiqueApproach`**: Critiques a hypothesized approach before committing to execution (used when prior approaches failed).
- **`PrimitiveRouteNext`**: Analyzes the current reflection state to route the execution to the most appropriate next primitive (e.g., `replan`, `retry`, `rollback`).
- **`PrimitiveFormulateHypothesis`**: Forces the LLM to state a scientific hypothesis before executing a modifying tool to ensure reasoning precedes action.
- **`PrimitiveIterativeEvaluation`**: Specialized loop for repeatedly evaluating code structure or logic until it passes a specific constraint.

## 5. Memory & Context Management

- **`PrimitiveExtractDurableMemory`**: Scans the working session context to extract overarching rules, bugs, or architectural decisions to persist into long-term Project Memory.
- **`PrimitiveRefreshTaskContext`**: Triggers when the context window exceeds the token limits. Re-summarizes the execution history to slide the context window safely.
- **`PrimitiveExtractKeywords`**: Extracts semantic keywords from a prompt to query the project memory store.
- **`PrimitiveRerankMemory`**: Reranks retrieved memory chunks by relevance to the current step.
- **`PrimitiveExtractStepResult`**: Summarizes the raw outputs and lessons learned from a step to store efficiently in the task history without bloating tokens.
- **`PrimitiveConsolidateContext`**: Merges newly discovered facts and constraints into the working task context.
- **`PrimitiveSummarizeTask`**: Summarizes the entire completed task for historical logging.

## 6. Interactive & Fallback

- **`PrimitiveDiscuss`**: Handles standard conversational chat flows that do not require executing a modifying plan.
- **`PrimitiveAskUser`**: Pauses execution to explicitly ask the user for clarification on an ambiguous requirement or missing information.
- **`PrimitiveEvaluateUserAnswer`**: Interprets the human's response to an `AskUser` prompt.
- **`PrimitiveNegotiation`**: Negotiates the scope of a task with the user if the request is deemed too massive or dangerous.
- **`PrimitiveIntervention`**: The terminal fallback state. Triggered on catastrophic loop failures or when the Max Execution Loops threshold is breached, yielding control back to the UI.
