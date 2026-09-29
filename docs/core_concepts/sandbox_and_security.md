# Sandbox and Security

Loa is designed with the assumption that LLMs will occasionally hallucinate dangerous commands, wipe directories by mistake, or execute arbitrary code returned from web dependencies. To mitigate this, Loa natively integrates with a Dockerized Sandbox.

## What is the Loa Sandbox?

The Sandbox (`loa-sandbox`) is a Docker container built on Debian that mounts your active project directory into an isolated environment. When Loa executes modifying tools (specifically `ToolExecuteProcess` and `ToolExecuteShell`), it routes those commands into the container rather than executing them on your host OS.

### Runtimes Included
The default `loa-sandbox` Dockerfile comes pre-installed with common language runtimes so the agent can build, test, and execute your code natively:
- Go
- Python (with `pip`)
- Node.js (with `npm`/`yarn`)
- PHP (with `composer`)
- Rust (with `cargo`)
- Build tools (`make`, `gcc`, `git`, `curl`, `jq`)

## Volume Mounts and Path Security

When you launch `loa-sandbox` in your terminal:
1. The script identifies your current working directory (e.g., `/home/user/projects/my-app`).
2. It mounts that directory into the container at `/workspace`.
3. The Loa engine restricts all file operations (Read, Write, Delete) to paths *inside* `/workspace`.

### Path Traversal Protection
Because the agent physically runs inside the container, it is impossible for a hallucinated command like `rm -rf /` to destroy your host machine. Running `cat ~/.ssh/id_rsa` will only look for SSH keys inside the ephemeral sandbox user account, not your host machine's private keys.

## User and Permission Mapping

Docker runtimes often suffer from root-permission issues when mounting host directories (files created by the container are owned by `root`). 
To solve this, the `loa-sandbox` startup script dynamically extracts your host `UID` and `GID` and passes them to Docker Compose via an `.env` file. The container executes commands as a user mapped exactly to your host identity, ensuring you never have to `sudo chown` files created by the agent.

## Disabling the Sandbox (Host Execution)

If you have a specialized environment and need the agent to run directly on your host machine without Docker, you can run the binary directly:

```bash
loa -addr 0.0.0.0:8104 .
```

> **[WARNING]**
> Running in Host mode means `execute_shell` commands run with your user privileges. A hallucinated `rm -rf *` in the wrong directory, or a malicious script downloaded by the agent, could compromise your machine. Use Host mode strictly with `Permission Mode: ask_all` enabled in the settings.

## Attachment Sandboxing & Bi-Directional Sync

When you upload external files to a session and attach them via the chat interface, Loa utilizes an ephemeral sandboxing mechanism within the `.loa` directory to manage file state safely.

### The Ephemeral Sandbox
Instead of directly exposing your original uploaded files to the agent, attached files are physically cloned into a temporary `.loa/attachments/<session_id>/` directory. The agent interacts with these clones, believing them to be standard local files in its workspace.

### The `fsnotify` Watcher (Resilience Mechanism) (manually uploaded files only)
To maintain the integrity of your files while allowing dynamic edits, the Loa engine runs a background filesystem watcher on the ephemeral sandbox. 
- **Auto-Heal (Read-Only):** If you mark an upload as `READ_ONLY` and the agent mistakenly alters it, the watcher instantly overwrites the sandbox file using the pristine master copy stored in `.loa/uploads/`. This completely neutralizes the unauthorized change.
- **Auto-Sync (Writable):** If the file is permitted to be altered, the watcher operates in reverse. It takes the agent's sandbox changes and instantly syncs them back to the original `.loa/uploads/` directory. This preserves the agent's work permanently, ensuring changes aren't lost when the file is detached or the session ends.

> **[NOTE]**
> **Resilience, not absolute security:** While the `fsnotify` watch-and-copy trick prevents accidental data loss or hallucinated overwrites during normal execution, it is not an impermeable security boundary. Because the ephemeral sandbox and the master `.loa/uploads/` directory both physically reside inside the mounted `/workspace`, an actively malicious agent (or a malicious downloaded script) could theoretically traverse the file system, discover the hidden `.loa` directory, and modify the master files directly bypassing the watcher. This mechanism is designed for workflow resilience and state preservation, not strict cryptographic isolation.
