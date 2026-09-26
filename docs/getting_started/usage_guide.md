# Loa Usage Guide

Welcome to Loa. This guide will walk you through the day-to-day workflow of operating the agent, interacting with its UI, and writing effective prompts to ensure reliable execution.

## The Web UI Overview

When you launch Loa (either natively or via `loa-sandbox`), the primary interface is accessible in your browser at `http://localhost:8104`. The UI is divided into several key tabs:

### 1. Chat Tab (The Command Center)
The Chat tab is where you interact directly with the agent.
- **Goal Submission:** Type your request here. The agent will analyze your intent and decide whether to reply conversationally or generate an execution plan to modify the codebase.
- **Interventions:** If the agent is currently executing a long-running plan and goes off track, you can type a new message here to intervene. The agent will pause, assess your new guidance, and optionally update its plan.

### 2. Plan Tab (The Execution State)
If your prompt requires modifying the codebase, Loa will generate a Directed Acyclic Graph (DAG) plan.
- **Visualizing Progress:** You can see the step-by-step breakdown of the task. Steps will highlight as `Active`, `Success`, or `Failed`.
- **Historical Plans:** If you run multiple tasks in a single session, you can use the dropdown at the top of the Plan tab to review how previous tasks were executed.
- **Dependency Graph:** Below the step list, a network graph visualizes the dependencies between execution steps, making it easy to see which tasks can be run in parallel or must wait.

### 3. Context & Log Tabs (Observability)
- **Context Tab:** Displays the current contextual "Working Memory" the agent is using. This includes facts it has learned, architectural decisions it has discovered, and constraints it is adhering to.
- **Execution Log:** A raw, real-time feed of exactly what the LLM is thinking, the tools it is calling, and the raw terminal outputs it is receiving.

---

## Writing Effective Prompts

Because Loa operates as a strict state-machine, it thrives on explicit, clear objectives rather than vague ideas.

### The "Intent" Phase
When you submit a prompt, Loa runs it through an `Intent` classifier. It decides if your message is:
- **Investigatory:** "How does the authentication system work?" (The agent will read code and reply in chat without changing anything).
- **Modifying:** "Add a password reset endpoint to the authentication system." (The agent will generate a DAG plan and request permission to execute it).

### Best Practices for Modifying Prompts
1. **Be specific about the "What":** State the exact feature or bug fix.
2. **Provide Acceptance Criteria:** If possible, tell the agent exactly how you expect to verify the task is complete (e.g., "The task is done when `curl /reset` returns a 200 OK").
3. **Mention specific files (Optional):** If you know where the issue is, mentioning it (e.g., "Check `internal/auth/reset.go`") saves the agent time during the discovery phase.

---

## Managing Paused States & Interventions

Loa features a hard-limit circuit breaker called the **Max Execution Loops** (configurable in Settings). This prevents the agent from infinitely looping if it gets stuck trying to fix a persistent bug.

### What happens when Loa pauses?
If the loop limit is reached, or if the agent encounters a catastrophic error it cannot recover from, it will transition to a `PAUSED` state.
- All background execution will halt.
- The UI will prompt you that the agent requires human intervention.

### How to recover
1. **Review the Execution Log:** Look at the most recent steps to understand *why* the agent got stuck (e.g., a missing dependency, a failing unit test it can't figure out).
2. **Update Settings (If necessary):** While PAUSED, you can freely update the configuration via the Settings modal. If the context window became too small or a request timeout was too low, simply adjust the values and click Save. The new configurations will apply immediately when execution resumes.
3. **Provide Guidance:** Type a message in the Chat tab explaining the solution (e.g., "The test is failing because you forgot to mock the database connection in `setup_test.go`").
4. **Resume Execution:** The agent will process your guidance, optionally update its remaining plan, and resume execution with the newly applied settings.

## Sandbox Volume Management
If you are running `loa-sandbox` (which is highly recommended), remember that the agent operates inside an isolated Docker container.
- It **only** has access to the directory where you ran the `loa-sandbox` command.
- It **cannot** access `/etc/`, `~/.ssh/`, or any files outside your project root on your host machine.
- If you need the agent to utilize a specific host binary that isn't installed in the Debian container, you will need to map it manually or install it during the task.
