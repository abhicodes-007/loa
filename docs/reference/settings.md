# Settings Reference

Loa configuration is entirely managed locally and saved to your project's `.loa/config.json` file, with global fallbacks available in your user configuration directory. You can adjust these settings at any time using the Settings Modal in the Web UI.

## Model Configuration

- **LLM Base URL** (`ollama_url`): The endpoint for your local OpenAI-compatible inference server (e.g., `http://127.0.0.1:11434/v1` for Ollama).
- **LLM Token** (`api_key`): API key if required by your inference server (can be left blank for local Ollama instances).
- **Model Crawling** (`model_crawling`): The specific model string to use for initial project crawling.
- **Analysis/Background Model** (`model_conversation`): Used for lightweight, analytical background primitives (e.g., assessing blind execution limits, determining intents, synthesizing tasks). 
- **Model Planning** (`model_planning`): The model used for DAG plan generation.
- **Model Executing** (`model_executing`): The primary model used to execute tool calls in the loop.
- **Embedding Model** (`embedding_model`): The model used for vector generation.

## Timeout Configuration

- **Model Timeout Seconds** (`model_timeout_seconds`): Maximum wait time for an LLM response before the engine aborts the request.
- **Command Timeout Seconds** (`command_timeout_seconds`): Maximum wait time for a shell command or process to execute in the sandbox before it is forcefully killed.
- **LLM Polling Timeout Seconds** (`llm_polling_timeout_seconds`): Timeout for LLM polling checks.
- **Server Read Header Timeout** (`server_read_header_timeout_seconds`): HTTP server read header timeout.

## Pipeline & Flow Control

- **Evaluation Mode** (`evaluation_mode`): Determines how strictly Loa evaluates its progress. Set to `strict` (evaluates every single step against the objective) or `dynamic` (allows the agent to execute tightly coupled changes blindly for speed, monitored by a background LLM supervisor).
- **Blind Action Threshold** (`blind_action_threshold`): When in dynamic mode, this defines the maximum number of consecutive blind executions the engine can perform before the background supervisor forces a deep evaluation.
- **Max Execution Loops** (`max_execution_loops`): The maximum number of tool executions/iterations allowed for a single task session before the engine trips the circuit breaker and enters the `PAUSED` state.
- **Max Plan Depth** (`max_plan_depth`): Maximum allowed depth for nested DAG steps during planning/decomposition.
- **JSON Repair Attempts** (`json_repair_attempts`): How many times the engine will automatically prompt the LLM to fix a malformed JSON payload before giving up.
- **Persistent Project Instructions** (`project_instructions`): Custom rules, coding conventions, or architectural guidelines that are permanently injected into the system prompt for every step executed in this project.

## Context & Memory Management

Loa manages the LLM context window explicitly through mathematical token budgets.

- **Context Budget** (`context_budget`): The absolute maximum token limit of your chosen LLM. Loa will aggressively slice memory and tool outputs to never exceed this limit.
- **Output Reserve** (`output_reserve`): The amount of tokens strictly reserved for the LLM's response generation.
- **Code Budget** (`code_budget`): The amount of tokens reserved for code generation.
- **Recent Messages** (`recent_messages`): Number of recent chat messages to retain in context.
- **Memory Top K** (`memory_top_k`): Number of semantic memory chunks to retrieve during a search.
- **Memory Candidate Pool** (`memory_candidate_pool`): Number of candidate chunks to fetch before reranking.
- **Memory Rerank Gates** (`memory_rerank_gates`): Number of gates/passes during memory reranking.
- **Memory Pool Expansion Limit** (`memory_pool_expansion_limit`): Allowed expansions for memory pooling.
- **Context Minimum Buffer** (`context_minimum_buffer`): A safety buffer (e.g., `2000`) subtracted from the max length to account for tokenization discrepancies between Loa's counting and the LLM's internal tokenizer.
- **Context Safe Budget Floor** (`context_minimum_safe_budget`): The minimum allowed token space reserved for historical context. If context sliding drops below this floor, it indicates the task has generated too much unstructured output to continue safely.
- **Code Budget Max Floor** (`context_code_budget_max_floor`): The absolute minimum token budget reserved exclusively for the LLM to write code or generate JSON structures during execution.

## Memory Decay (LTS vs Session)

- **Memory LTS Decay Half Life (Hours)** (`memory_lts_decay_half_life_hours`): Time before Long-Term Storage (Project Memory) decays by half (default `336.0` hours).
- **Memory LTS Max Weight** (`memory_lts_max_weight`): The starting weight of Project Memory (default `0.03`).
- **Memory Session Decay Half Life (Hours)** (`memory_session_decay_half_life_hours`): Time before active session memory decays by half (default `2.0` hours).
- **Memory Session Max Weight** (`memory_session_max_weight`): The starting weight of Session Memory (default `0.20`).

## File System & Tool Limits

- **FS Watcher Debounce (ms)** (`fs_watcher_debounce_ms`): Debounce threshold for the file watcher.
- **Crawler Small File Threshold** (`crawler_small_file_threshold`): Max lines for a file to be processed wholly without chunking during codebase indexing.
- **Crawler Max Chunk Lines** (`crawler_max_chunk_lines`): Max lines per chunk when parsing large files.
- **Max Stored Tool Output (Bytes)** (`max_stored_tool_output_bytes`): Hard limit (default 4MB) for capturing output from shell commands. Outputs exceeding this are truncated in the DB.
- **Max Prompt Tool Output Bytes** (`max_prompt_tool_output_bytes`): Limit (default 30,000 bytes) for injecting tool outputs into the LLM context. Outputs exceeding this are automatically spilled over into a readable artifact file, protecting the prompt budget while still allowing the agent to read the full output safely.

## Permissions

Controls the sandbox boundaries for agent actions.

- **Permission Mode**:
  - `ask_all`: The agent asks for user approval before executing ANY tool.
  - `ask_selected`: The agent asks for user approval only for the tools explicitly checked in the grid.
  - `allow_all`: The agent runs fully autonomously. (Recommended only inside `loa-sandbox`).
- **Tool Grid**: Checkboxes to toggle permissions for individual tools (e.g., `execute_shell`, `write_file`, `delete_directory`).

> **Warning:** Loa intentionally does not strictly "safety parse" shell commands. If `execute_shell` is permitted, the agent has the capability to run commands that could modify your system if you are running in Host Execution mode.
