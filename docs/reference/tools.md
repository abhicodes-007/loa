# Tooling Reference

Loa interacts with the host environment exclusively through a registered set of tools managed by the Tool Manager (`internal/tools/manager.go`). The LLM cannot execute native Go functions or arbitary syscalls directly; it requests execution via structured JSON payloads.

## Tool Categories

### 1. Read & Analysis Tools (Non-Mutating)
These tools are used to traverse the filesystem, read files, and parse code semantics. They are safe to run at any time and do not require user approval.

- **`read_file`**: Reads the entire contents of a file.
  - *Input*: `{"path": "main.go"}`
- **`read_range`**: Reads specific line ranges to conserve tokens. Ideal for inspecting massive files.
  - *Input*: `{"path": "main.go", "start": 10, "end": 20}`
- **`search_path`**: Locates files by name or glob pattern.
  - *Input*: `{"query": "*.go", "limit": 50}`
- **`search_text`**: Performs fast regex or literal string matching across the codebase.
  - *Input*: `{"pattern": "TODO:", "regex": false, "limit": 50}`
- **`find_symbol`**: Uses `ctags` to locate where classes, functions, or structs are defined.
  - *Input*: `{"query": "Manager", "limit": 10}`
- **`list_symbols`**: Lists all available symbols (functions, structs, interfaces) in a given file.
  - *Input*: `{"path": "internal/tools/manager.go"}`
- **`read_ast_node`**: Uses `ast-grep` to extract semantic logic blocks based on AST rules.
  - *Input*: `{"path": "main.go", "rule": "{ kind: function_declaration }"}`
- **`analyze_large_file`**: Specialized tool allowing the agent to parse massive files without blowing out the token limit by chunking.

### 2. Modifying Tools (Mutating)
These tools alter the state of the workspace. They trigger the `IsMutating` flag in the engine, requiring user permissions depending on settings.

- **`write_file`**: Overwrites or creates a new file.
  - *Input*: `{"path": "main.go", "content": "package main..."}`
- **`patch_file`**: Applies a unified diff/patch to an existing file.
  - *Input*: `{"path": "main.go", "old": "fmt.Println(a)", "new": "fmt.Println(b)", "replace_all": false}`
- **`patch_ast_node`**: Semantically replaces a specific AST node.
  - *Input*: `{"path": "main.go", "rule": "...", "rewrite": "..."}`
- **`create_directory`**: Provisions a new directory tree.
  - *Input*: `{"path": "internal/newpkg"}`

### 3. Execution Tools (Mutating / Dangerous)
These tools spawn sub-processes. Inside the `loa-sandbox`, they execute securely against the container's isolated OS. On Host execution, they run natively against your real OS.

- **`execute_process`**: Executes a specific binary directly with arguments (avoids shell injection risks).
  - *Input*: `{"command": "go", "args": ["build", "./..."], "dir": "."}`
- **`execute_shell`**: Spawns a raw bash sub-process. Capable of complex piping and scripting.
  - *Input*: `{"script": "npm install && npm run build", "dir": "."}`

### 4. Destructive Tools (High Risk)
Tools that permanently remove data. They trigger the `IsDestructive` flag and are the most tightly restricted tools.

- **`delete_file`**: Deletes a specific file.
  - *Input*: `{"path": "temp.txt"}`
- **`delete_directory`**: Recursively removes a directory tree (`rm -rf`).
  - *Input*: `{"path": "build/"}`

### 5. Version Control Tools
Native wrappers for workspace differential analysis.
- **`git_status`**: Returns the current git workspace status.
- **`git_diff`**: Returns the uncommitted diffs to analyze what the agent has changed.

### 6. Agent Introspection & Artifact Tools
Tools that allow the agent to manage internal documentation, scratchpads, and execution history.

- **`search_task_steps`**: Searches previous steps in the DAG to recall past findings.
- **`read_task_step`**: Reads the detailed payload of a specific past step.
- **`read_tool_output`**: Reads the raw, un-truncated output of a previously executed tool.
- **`artifact_list`**: Lists internal markdown artifacts.
- **`artifact_read`**: Reads an internal artifact.
- **`artifact_write`**: Creates or overwrites an internal artifact.
- **`artifact_append`**: Appends text to an internal artifact.
- **`artifact_patch`**: Patches text in an internal artifact.

## Output Truncation

To protect the LLM context window from collapsing, the Tool Manager enforces a hard output limit (defaulting to 4MB) for all tool execution outputs. 
If a tool (like `execute_shell "cat large.log"`) exceeds this limit, the engine truncates the output string, appending a `Truncated: true` flag in the `ToolResult`. The LLM is instructed to recognize this flag and follow up with `read_range` or `analyze_large_file` tools to parse the data sequentially.
