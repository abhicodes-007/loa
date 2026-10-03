<div align="center">
  <img src="loa.png" alt="Loa Logo" width="400"/>
  <h1>Loa</h1>
  <p><strong>The Autonomous Local Coding Agent</strong></p>
  <p>
    <a href="#why-loa-core-strengths">Core Strengths</a> •
    <a href="#quick-start">Quick Start</a> •
    <a href="#core-concepts">Core Concepts</a> •
    <a href="#faq">FAQ</a> •
    <a href="#contributing">Contributing</a> •
    <a href="LICENSE">License</a>
  </p>
</div>

<br/>

## Summary

Loa is a local-first, stateful coding agent explicitly designed for project-wide architecture and high-level development tasks. To achieve deeper, more reliable reasoning than standard coding models, Loa operates on a strict, continuous cognitive state-machine (Planner -> Executor -> Reflection). It deliberately trades execution time and raw inference count for higher execution quality and verified logic manipulation. It persists state locally per project and optionally executes within an isolated Docker sandbox for system safety.

> [!IMPORTANT]  
> **Cost Warning:** Loa uses a **massive** amount of inferences continuously to achieve it's high-quality reasoning and output. We strongly discourage using cloud-based API providers (like OpenAI or Anthropic) as it will generate massive costs. Loa is strictly designed for **self-hosted, local environments** where there is no per-token cost.

## Why Loa? (Core Strengths)

In the current landscape of AI coding tools, most fall into two categories: simple IDE autocomplete plugins, or cloud-hosted black-box agents. Loa sits in a unique position:

1. **Reliability over Hallucination (State-Machine Architecture):** Most agents rely on unstructured, open-ended loops that frequently hallucinate, get stuck in infinite retry loops, or quietly fail. Loa strictly enforces a Directed Acyclic Graph (DAG) state-machine. It must formulate a plan, execute a step, verify the result, and self-correct explicitly. 
2. **100% Offline & Private:** Loa does not rely on cloud vector databases. Project Memory and working context are serialized locally into a `.loa/` folder inside your project. This guarantees zero cross-project leakage and 100% offline functionality.
3. **Safe Execution (Native Sandboxing):** Because LLMs can hallucinate dangerous shell commands, Loa offers an optional, native Docker container (`loa-sandbox`). When used, the agent operates securely inside this isolated Debian runtime, rather than directly on your host machine.

## Quick Start

### Prerequisites
- **Git**
- **Docker & Docker Compose** (Optional, but highly recommended for Sandboxed execution)
- **Local LLM Endpoint**: Loa expects an OpenAI-compatible API endpoint (e.g., Ollama running a `qwen3.8 27-32b` variant, or vLLM).

### Installation (Official Script)

The setup script automatically downloads the binary and will print the required terminal aliases for you to manually add to your `.bashrc` or `.zshrc` (we intentionally don't mess with your environment files automatically). It will also optionally prompt you to install the `loa-sandbox` (which requires a target directory to store the relevant sandbox docker files).

During setup, you will also be prompted to automatically download a GGUF embedding model (e.g., `nomic-embed-text` or `mxbai-embed-large`). It is highly recommended to do this so Loa can perform high-speed, local CPU embeddings for semantic search. Alternatively you can configure on first start the openapi compatible api to be used for embedding calls - tho the cpu variant is recommended.

```bash
curl -fsSL https://raw.githubusercontent.com/laughingmandev/loa/main/scripts/setup.sh | bash
```

**Installing from Source (Local Build)**
If you clone the repository, you can run the setup routine to compile the binary from source instead of downloading the pre-compiled release:
```bash
git clone https://github.com/laughingmandev/loa.git
cd loa
bash scripts/setup.sh --local
```

> **📖 First Time User?** Before running the agent, please read the [Usage Guide & Setup Instructions](docs/getting_started/usage_guide.md) to learn how to configure your LLM, allocate context budget based on your hardware, and navigate the initial setup wizards.

### Running the Agent

1. Start your local LLM server (e.g. `ollama serve`).
2. Navigate to your target project directory.
3. Launch the agent. You have two execution modes:

**Sandboxed Mode (Recommended):**
Runs the agent inside the Docker container, mounting your project directory safely. 
*Note: The sandbox automatically detects if you have a compatible Linux host binary and uses it for instant boot times. For macOS/Windows users, it gracefully falls back to compiling the Linux binary from source inside the container.*
```bash
cd /path/to/your/project
loa-sandbox
```

**Host Mode (Direct Execution):**
Runs the agent natively on your host machine without Docker. 
> [!CAUTION]  
> **Danger:** This is inherently dangerous if the agent decides to execute destructive shell commands (like `rm -rf /`). Loa features a built-in Permission System in the UI's Settings menu. If you must run in Host Mode, you should **always** configure the permissions to explicitly require manual human approval for shell/process execution tools before the agent is allowed to run them.
```bash
cd /path/to/your/project
loa -addr 0.0.0.0:8104 .
```

4. Open the UI in your browser at `http://localhost:8104`.

## Documentation

For a detailed technical breakdown of Loa's internals, see the `docs/` directory:

### Getting Started
- [Usage Guide](docs/getting_started/usage_guide.md): The UI, Prompt Engineering, and Managing Paused States.

### Core Concepts
- [Design Philosophy](docs/core_concepts/design_philosophy.md): Why Loa exists and the principles behind its control system.
- [Architecture & Pipeline](docs/core_concepts/architecture.md): The underlying DAG state-machine.
- [Memory System](docs/core_concepts/memory_system.md): Context sliding, token budgets, and long-term project memory.
- [Sandbox & Security](docs/core_concepts/sandbox_and_security.md): Isolation mechanisms and permissions.

### Reference
- [Primitives Reference](docs/reference/primitives.md): The atomic internal states of the engine.
- [Tooling Reference](docs/reference/tools.md): The native tools available to the LLM.
- [Settings Reference](docs/reference/settings.md): Configuration variables in `.loa/config.json`.

## Future Enhancement Plans

*Note: The following is a living list of ideas and thoughts for the future of Loa. The order does not reflect priority or a strict roadmap. Some of these may be reprioritized, altered, or dropped entirely based on need and bandwidth.*

- **Chat File Provisioning:** Adding the ability to drag-and-drop files directly into the chat interface. This will allow the agent to read and analyze external files (e.g., a massive log dump or an external API spec) within the scope of the active session, without those files needing to be formally added to your project's codebase.
- **Multi-language UI & Chat Support:** An auto-translation layer for the web interface. While Loa's internal cognitive engine and prompts will strictly remain "English thinking" to maintain parser stability, this feature would allow users to converse in the chat and read UI elements in their native language.
- **Expanded Architecture Support:** Broadening native support for different operating systems (like macOS). Due to current hardware limitations, this will only be tackled if requested by users willing to actively provide test environments and feedback data.
- **UI Enhancements:** Loa is currently in early beta. The core execution engine is solid, but the Web UI has some quirks and rough edges that will be polished over time to improve the overall user experience.

## FAQ

**Q: Why did you build Loa?**  
A: I built Loa because existing local-first agents lacked the reliability I needed. Most other agents fall short in three areas: (1) They lack true observability into what the agent is actually doing. (2) They perform poorly with smaller local models. (3) They don't allow for human steering once execution begins. Furthermore, based on my separate research into "dynamic thought compositors," I learned that using a mutating DAG (where the plan adapts as execution unfolds) yields significantly higher quality results than blindly forcing an LLM to follow an initially drafted, static plan.

**Q: Why does Loa use so many inferences?**  
A: This stems directly from the "dynamic thought compositor" research. In Loa, a single inference is not supposed to be a complex chain-of-thought solving an entire problem. Instead, a single inference represents a *single sequential logical step* in a thought. By decomposing the work into tiny, focused steps, the model isn't diluted by trying to solve too many problems at once. Yes, this results in a massive amount of API calls, but it dramatically increases the quality of reasoning and execution.

**Q: Which local LLM works best with Loa?**  
A: Currently, I highly recommend a model around the 27B-32B parameter range—specifically **Qwen3.8 (27-32B)** or similar variants. While larger models (70B+) can certainly perform better, this range is the "minimum" size where I've observed consistently excellent results on complex tasks. *Note: Due to hardware limitations, I haven't been able to run exhaustive tests on all models yet. I will update these recommendations as I collect more data.*

**Q: Is Loa compatible with any API?**  
A: Yes, Loa uses standard OpenAI-compatible API formatting. As long as your inference server exposes an OpenAI-compatible endpoint, it will work.

**Q: Why not just use an IDE plugin like GitHub Copilot?**  
A: IDE plugins excel at autocomplete and micro-edits. Loa is designed for macro-level architectural work. You give Loa a high-level goal, and it will autonomously read the codebase, generate a plan, edit multiple files, and verify the build.

**Q: Why doesn't Loa persist Session memories into Long-Term Storage (LTS)?**  
A: Because session memories are highly state-specific. Codebases mutate rapidly—sometimes across multiple branches or parallel sessions at once. Pushing transient session knowledge into LTS would create conflicting, outdated memory that requires a massively complex consolidation pipeline. Instead, Loa strictly reserves LTS for "global truths," such as the structural index of the project (which automatically updates on file changes) and any explicit architectural definitions added by the developer. 

**Q: Why doesn't Loa include trendy features like MCP, RAG, or Multi-Modal media handling?**  
A: Loa is designed to solve a few very specific problems exceptionally well, rather than doing everything mediocrely. It is not meant to be the ultimate, all-encompassing agent. Features may be added if there is a genuine need, but I will actively deny feature requests that dilute Loa's core scope and target philosophy. 

**Q: What is the policy on Contributions and Feature Requests?**  
A: If you want to contribute, please create an issue *before* writing any code. Because Loa has a very strict target scope, you must ensure your proposed feature aligns with the project's philosophy. This prevents you from wasting time writing code that will ultimately be rejected.

**Q: Why is the target architecture support so small?**  
A: Loa is built by a single developer. While I don't own a Mac or Windows environment to properly test native binaries, the `loa-sandbox` container solves this by providing a guaranteed Linux runtime that works flawlessly across macOS and Windows via Docker. Support for native OS architectures will likely come if requested by users willing to actively provide test environments and feedback.

## Contributing
Loa is actively developed. If you want to contribute, please refer to the architecture documentation to understand the execution pipeline before submitting PRs, and ensure you open an issue to discuss feature scope first!