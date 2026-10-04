# Sandbox and Security

Loa can execute code, shell commands, filesystem mutations, and downloaded project dependencies. The security model therefore has several distinct layers: project-root validation for file tools, user approvals, and an optional Docker execution environment.

These controls should not be conflated. In particular, the Docker sandbox protects host resources outside its mounts, while the target project is intentionally mounted read/write so the agent can modify it.

## 1. What `loa-sandbox` Does

The provided sandbox is a Debian-based Docker environment. `run.sh` starts the container with the current working directory mounted at:

```text
/workspace
```

That mount is read/write.

The service publishes Loa on port `8104`.

The current Docker image includes development runtimes/tools for Go, Python, Node.js, Rust, Java, PHP/Composer, compilers/build utilities, Git, curl, jq, and related tooling used by coding tasks.

## 2. Host Mounts

The current Compose configuration exposes several host locations deliberately:

- the current project -> `/workspace` (read/write),
- host `~/.loa` -> the container Loa home data location (read-only),
- host `~/.config/loa` -> global Loa config (read-only) when that host directory exists.

If the host global-config directory does not exist, the launcher provisions sandbox-local writable config storage instead.

Project-local runtime/config files such as `.loa/` and `.loa.json` are inside the `/workspace` mount and are therefore writable from the sandbox.

The sandbox also supplies host-network reachability through `host.docker.internal` so a model server running on the host can be used from the container.

## 3. File-Tool Path Boundary

Loa's registered filesystem tools use project-relative paths and validate the resolved path against the project root. The current implementation also resolves symlinks/real parent paths where appropriate to reject filesystem traversal outside the project root.

This boundary applies to the registered file tools such as read/write/patch/delete/create-directory operations.

It does **not** turn arbitrary child processes into a project-root sandbox.

## 4. `execute_process` and `execute_shell`

`execute_process` executes a binary directly with arguments.

`execute_shell` executes a command through `/bin/sh -c`.

Both start with the project root as their working directory, but neither is constrained by the project-root validation used by the file tools. A command can navigate elsewhere if the operating environment makes those paths available.

Therefore:

- in Host Mode, process/shell commands run with the host user's privileges and host filesystem visibility;
- in `loa-sandbox`, they run inside the container and can see the container filesystem plus explicitly mounted host paths.

Because `/workspace` is a writable host mount, a destructive command inside the sandbox can still destroy or corrupt the mounted project. Container isolation protects host paths that were not mounted; it is not a backup system for the project.

## 5. Permissions

The tool permission modes are:

- `ask_all`: request approval for every tool,
- `ask_selected`: request approval for the tools selected in the permission grid,
- `allow_all`: do not request per-tool approval.

The default configuration is `ask_selected` and includes the mutating/process/shell tools in the approval set.

The permission system is an approval boundary, not a command sanitizer. Loa intentionally does not attempt to fully safety-parse shell scripts before execution.

See [Settings Reference](../reference/settings.md) and [Tooling Reference](../reference/tools.md).

## 6. Host Mode

Host Mode runs the Loa process directly as your user:

```bash
loa .
```

In this mode, arbitrary shell/process execution has the same effective OS permissions as the Loa process.

If you use Host Mode, keep the relevant process/shell tools behind approval unless you explicitly want autonomous host execution.

## 7. Session Uploads and Attachments

Uploaded files are stored under Loa's project-local runtime directory and attached to a session through a working attachment copy.

The current attachment watcher provides two workflow behaviors for manually uploaded files:

- **read-only upload:** if the working attachment is changed, the watcher restores it from the master upload copy;
- **writable upload:** changes to the working attachment are synchronized back to the master upload copy.

Artifacts attached to the prompt are treated as read-only.

This mechanism is for workflow resilience/state handling. It is **not** a hard security boundary against malicious code, because the attachment workspace and master project-local upload storage are both inside the writable project tree and can be reached by sufficiently powerful shell/process execution.

## 8. Platform/Launcher Limitations

The release/setup path currently targets Linux `amd64` and `arm64`.

The sandbox launcher prefers an already-installed compatible Linux ELF `loa` binary. Its current source-build fallback expects a developer checkout at a specific local source path. The launcher should therefore not be documented as a generic automatic source-build fallback for arbitrary macOS/Windows installations.
