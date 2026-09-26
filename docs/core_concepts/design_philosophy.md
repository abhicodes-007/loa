# Design Philosophy

The strongest case for Loa is not simply "it uses an LLM to edit code." It is the **control system** built *around* the model. 

Loa is designed for architect-driven engineering rather than autonomous product taste. You define the architectural intent and invariants; Loa performs the expensive implementation, research, and verification work underneath that direction.

Below is a breakdown of the core philosophies that differentiate Loa from standard API wrappers.

---

## 1. State & Planning (The Mutable DAG)

- **Dynamic mutable DAG instead of a fixed plan.** Loa starts with a plan, but the plan is expected to change as the agent learns the codebase. Steps can be added, removed, rewired, retried, or decomposed rather than treating the initial plan as truth.
- **Dependencies are also context topology.** A DAG edge does not merely mean “step B waits for step A.” It determines which prior results become context for later reasoning. Changing the DAG therefore changes both execution order and information flow.
- **Bounded cognitive operations.** A step is not supposed to solve an entire engineering problem in one huge inference. Loa breaks work into narrower reasoning/actions and allows many sequential iterations around one objective.
- **Sequential reasoning where state matters.** Loa deliberately does not optimize for maximum parallelism. When interpretation depends on what was learned in the previous thought, it keeps inference sequential. 
- **Plan approval before mutation.** Architectural discussion does not automatically turn into code changes. The flow is: understand intent → form plan → user approves → execute.

## 2. Execution, Verification & Recovery

- **Explicit hypothesis → action → evaluation loops.** Before risky or meaningful mutations, Loa can formulate a hypothesis and expected result, perform the action, and independently evaluate whether the result actually supports completion.
- **It distinguishes “worked” from “verified.”** Build success, test success, and semantic correctness are not treated as the same thing. Loa can return `partial` or `verification_required` even after code compiles.
- **It can say “I’m not sure yet.”** The control layer explicitly permits `partial`, `needs_more_information`, and similar outcomes rather than strongly incentivizing a premature “done.”
- **Failure generates new knowledge rather than merely consuming retries.** A failed test can trigger a hypothesis, investigation, architectural discovery, and a plan mutation.
- **Retry versus investigate versus repair versus replan.** Failure is not a single generic loop. Loa decides whether the problem needs another attempt, more information, a local repair, or a structural change to the remaining DAG.
- **Structured-output repair is built into the control loop.** Malformed JSON first gets deterministic repair where possible, then bounded model-based repair. A malformed orchestration response does not kill a long run.

## 3. Context & Memory Systems

- **Context is reconstructed, not accumulated.** Loa does not drag the entire conversation history forever. It surgically composes the context needed for the current inference from relevant original material.
- **Temporary context expansion and contraction.** Difficult steps can grow large context windows, but when the step finishes, the context drops back down rather than continuing upward forever.
- **Original evidence is preferred over lossy context summarization.** Instead of repeatedly “minifying” the whole context through an LLM and risking information loss or hallucinated summaries, Loa keeps source material intact and selects relevant pieces. 
- **Layered memory rather than one global RAG bucket.** Working memory, session memory, and project memory have different purposes and lifetimes.
- **Semantic promotion at task boundaries.** Working memory is not simply dumped wholesale into longer-term memory. Loa consolidates useful semantic state and discards episodic execution noise.
- **Codebase freshness is preferred over memory convenience.** Loa is relatively reluctant to rely on memory when it can inspect the actual source. For mutable software, this is usually the safer bias.
- **Artifacts as external reasoning storage.** Intermediate analyses, inventories, and reports can be stored as artifacts rather than bloating the active model context.

## 4. Autonomy & Observability

- **Long-running autonomy is a first-class design target.** It is meant to run for hours, potentially unattended, rather than being optimized around five-minute interactive coding exchanges.
- **Pause/resume and manual-intervention state.** An unrecoverable operational problem does not have to destroy a 10-hour run. Loa can pause with the DAG, memories, tool state, and artifacts preserved, let you fix the environment, and then resume.
- **User steering during execution.** You can inject information or constraints while a long task is running. Loa interprets the message in the context of the current task and can mutate the DAG accordingly.
- **Tool use is deliberately observable.** The UI exposes DAG state, inference inputs/outputs, context size, memories, tool calls, results, and execution state. You can inspect why the system behaved the way it did instead of receiving only a final diff.
- **Past DAG work remains queryable.** It can explicitly search and reopen previous task-step results instead of requiring all prior reasoning to remain in-context.
- **Long-run state is inspectable enough for external review.** The debug logs provide a forensic record of what the system believed, attempted, verified, repaired, and concluded.

## 5. Security & Economics

- **Local-first economics.** Hundreds of inferences are viable because the model runs locally. Loa can spend inference budget on checking, reconsidering, and verifying rather than minimizing API calls for cost reasons.
- **Explicit trust boundaries and permissions.** Read-only, selected approval, ask-every-time, and allow-all modes let the same agent operate safely on a host or autonomously inside a sandbox.
- **Optional Docker execution sandbox.** The sandbox makes unattended execution practical while keeping the target project mounted dynamically and the dangerous execution boundary obvious.
- **Model-agnostic control layer.** A stronger model should improve hypothesis quality and coding, but the durable DAG/memory/tool structure remains outside the model. The architecture does not depend on one frontier provider.
