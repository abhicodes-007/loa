# Loa Usage Guide

Welcome to Loa. This guide will walk you through the day-to-day workflow of operating the agent, interacting with its UI, and writing effective prompts to ensure reliable execution.

## First-Time Setup

Before you can start chatting, you'll encounter two setup wizards to configure your environment.

### 1. The Initial Setup Wizard
*(Configure your LLM endpoints and test connections)*
For the beginning it's recommended to just choose a single model for all inferences and one for embeddings. Make sure to test the configuration before finishing the initial setup. Regarding the "Total Context Budget" you should provide an amount that fits within the buffer of your inference API service, and optimally is supported by the model you plan to choose for inference.

![Initial Setup Wizard](initial_setup.png)

This wizard appears the very first time you launch Loa. Here you define your global API endpoint (e.g., `http://127.0.0.1:11434` for Ollama) and select the models Loa will use.
- **Tip:** You can click the **TEST CONFIGURATION** button at the bottom to ensure Loa can successfully reach your models. This is especially important to verify that you have selected a valid, dedicated embedding model (like `mxbai-embed-large`) for the Embedding Model slot since embedding is used for handling memories and internal searches.

### 2. The Project Setup Wizard
*(Configure project-specific boundaries)*

![Project Setup Wizard](project_setup.png)

This wizard appears whenever Loa detects it is running in a new directory. 
- **Project Purpose:** Briefly explain what this repository is about. This gives the agent immediate baseline context.
- **.loaignore:** Define which directories the crawler should ignore during project indexing (e.g., `node_modules`, `vendor`, `.git`). This prevents massive dependency folders from polluting the search index, though the agent can still manually read specific files inside them if necessary. Loa provides sensible defaults automatically. Choose those carefully because once started loa will at least for once index the whole project and create AST information alongside narrative frames for each file which it later in process can use for cheap lookups.

---

## Hardware & Performance Settings

Depending on your local hardware (specifically VRAM), you may want to tweak Loa's memory physics for optimal performance. You can access these via the **Settings** modal in the UI. To properly configure this you need to know the configured amount of context buffer provided by your inference API and optimally the supported context buffer size of the inference model of your choice.

![Settings Dialog - Context Synthesis](settings_context.png)

- **Low-End Setup (e.g., 16-24GB VRAM):** Keep the **Total Context Budget** around `40000` to `60000` tokens. In the *Context Synthesis* tab, you might want to keep the **Max Context History Steps** low (e.g., 4).
- **Medium Setup (e.g., ~32GB VRAM):** Keep the **Total Context Budget** around `100000` to `200000` tokens. In the *Context Synthesis* tab, you might want to keep the **Max Context History Steps** medium (e.g., 6-8).
- **High-End Setup (e.g., 64GB+ VRAM):** If you are running massive models with huge context windows, you can safely increase the **Total Context Budget** to `300000`+  tokens. You can also increase the **Max Context History Steps** (e.g., ~10) allowing the agent to remember much deeper conversational history during complex executions.

At the end of the day, the optimal values are completely dependent on your environment and use case.

> **⚙️ Advanced Configuration:** Loa features an extensive settings panel (accessible via the top navigation bar) containing 5 distinct tabs to fine-tune its memory physics, indexing behavior, and model parameters. For a deep-dive into how to optimize every single setting for your specific hardware and project size, please read the [Settings Reference Guide](settings_reference.md).

---

## The Web UI Overview

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

## Prompting

Because Loa operates as a strict state-machine, it thrives on explicit, clear objectives rather than vague ideas.

### The "Intent" Phase
When you submit a prompt, Loa runs it through an `Intent` classifier. It decides if your message is:
- **Discuss:** "What do you think about using GraphQL here?" (The agent will answer conversationally. It can use a small subset of read-only tools sequentially to gather context, but will not formulate a complex plan).
- **Investigatory:** "Analyze how the authentication system works and create a report." (The agent will draft a full DAG plan with steps and dependencies using read-only tools. **This executes automatically without requiring approval**).
- **Modifying:** "Add a password reset endpoint to the authentication system." (The agent will draft a full DAG plan with steps, dependencies, and verification criteria. **This requires your manual approval before execution begins**).

### Best Practices for Modifying Prompts
1. **Be specific about the "What":** State the exact feature or bug fix.
2. **Provide Acceptance Criteria:** If possible, tell the agent exactly how you expect to verify the task is complete (e.g., "The task is done when `curl /reset` returns a 200 OK").
3. **Mention specific files (Optional):** If you know where the issue is, mentioning it (e.g., "Check `internal/auth/reset.go`") saves the agent time during the discovery phase.

### Steering During Execution
While Loa is actively executing a complex, long-running plan, you are not locked out! You can send new prompts (steering messages) at any time in the Chat tab.
When Loa receives a steering message during execution, it will:
1. Pause its current background work.
2. Evaluate your new message.
3. Update its current actions or entirely rewrite its remaining DAG plan to accommodate your new instructions.
4. Resume execution seamlessly.
This allows you to dynamically guide the agent if you realize a requirement has changed or if you spot a better architectural approach while watching it work.

---

## Managing Paused States & Interventions

Loa features a hard-limit circuit breaker called the **Max Execution Loops** (configurable in Settings). This prevents the agent from infinitely looping if it gets stuck trying to fix a persistent bug. 

Additionally, because Loa persists its state entirely to your local disk, **you can manually pause execution at any time**. 

### When is pausing useful?
- **Error Recovery:** If the agent hits a loop limit or encounters a catastrophic error, it will automatically pause. The UI will prompt you for human intervention.
- **Resource Management:** If you are running a massive refactoring task and suddenly need your GPU resources for something else, you can manually pause Loa.
- **Session Continuity:** You can pause a long-running plan, completely shut down your computer for the night, and resume the exact same plan the next day right where it left off.

### How to recover from an Error Pause
1. **Review the Execution Log:** Look at the most recent steps to understand *why* the agent got stuck (e.g., a missing dependency, a failing unit test it can't figure out).
2. **Update Settings (If necessary):** While PAUSED, you can freely update the configuration via the Settings modal. If the context window became too small or a request timeout was too low, simply adjust the values and click Save. The new configurations will apply immediately when execution resumes.
3. **Provide Guidance:** Type a message in the Chat tab explaining the solution (e.g., "The test is failing because you forgot to mock the database connection in `setup_test.go`").
4. **Resume Execution:** The agent will process your guidance, optionally update its remaining plan, and resume execution with the newly applied settings.

## Sandbox Volume Management
If you are running `loa-sandbox` (which is highly recommended), remember that the agent operates inside an isolated Docker container.
- It **only** has access to the directory where you ran the `loa-sandbox` command.
- It **cannot** access `/etc/`, `~/.ssh/`, or any files outside your project root on your host machine.
- If you need the agent to utilize a specific host binary that isn't installed in the Debian container, you will need to map it manually or install it during the task.
