# Shotgun App

![Shotgun App Banner](https://github.com/user-attachments/assets/6dd15389-4ad9-493a-a0e7-9813eb143e38)

**Tired of Cursor cutting off context, missing your files, and spitting out empty responses?**

**Shotgun** is the bridge between your local codebase and the world's most powerful LLMs.
It doesn't just copy files; it **intelligently packages your project context** and can **execute prompts directly** against OpenAI (GPT-4o/GPT-5), Google Gemini, or OpenRouter.

> **Stop copy-pasting 50 files manually.**
> 1. Select your repo.
> 2. Let AI pick the relevant files (Auto-Context).
> 3. Blast the payload directly to the model or copy it for use in Cursor/Windsurf.

---

## 1. What Shotgun Does
Shotgun is a desktop power-tool that **explodes your project into a structured payload** designed for AI reasoning.

It has evolved from a simple "context dumper" into a full-fledged **LLM Client for Codebases**:
*   **Smart Selection:** Uses AI ("Auto-Context") to analyze your task and automatically select only the relevant files from your tree.
*   **Direct Execution:** Configurable API integration with **OpenAI**, **Gemini**, and **OpenRouter**.
*   **Prompt Engineering:** Built-in templates for different roles (Developer, Architect, Bug Hunter).
*   **History & Audit:** Keeps a full log of every prompt sent and response received.

---

## 2. Key Features

### 🧠 AI-Powered Context
*   **Auto-Context:** Don't know which files are needed for a bug fix? Type your task, and Shotgun uses an LLM to scan your tree and select the relevant files for you.
*   **Repo Scan:** supplement context retrieval with a `shotgun_reposcan.md` summary of your architecture to give the LLM high-level awareness before diving into code.

### ⚡ Workflow Speed
*   **Fast Tree Scan:** Go + Wails backend scans thousands of files in milliseconds.
*   **Interactive Tree:** Manually toggle files/folders or use `.gitignore` and custom rule sets to filter noise.
*   **One-Click Blast:** Generate a massive context payload instantly.

### 🔌 Direct Integrations
*   **OpenAI:** Support for GPT-4o and experimental support for **GPT-5** family models.
*   **Google Gemini:** Native integration for Gemini 2.5/3 Pro & Flash.
*   **OpenRouter:** Access hundreds of LLM's via a unified API.
*   **Ollama (Local):** Local OpenAI-compatible endpoint support (`http://localhost:11434/v1` by default).
*   **LM Studio (Local):** Local OpenAI-compatible endpoint support (`http://localhost:1234/v1` by default).

### 🛠 Developer Experience
*   **Prompt Templates:** Switch modes easily (e.g., "Find Bug" vs "Refactor" vs "Write Docs").
*   **History Tracking:** Never lose a generated patch. Browse past prompts, responses, and raw API payloads.
*   **Privacy Focused:** Your code goes only to the API provider you choose. No intermediate servers.

---

## 3. The Workflow

Shotgun guides you through a 3-step process:

### Step 1: Prepare Context
*   **Select Project:** Open your local repository.
*   **Filter:** Use the checkbox tree, `.gitignore`, or the **Auto-Context** button to define the scope.
*   **Repo Scan:** Edit or load the high-level repository summary for better AI grounding.
*   **Result:** A structured XML-like dump of your selected codebase.

### Step 2: Compose & Execute
*   **Define Task:** Describe what you need (e.g., "Refactor the auth middleware to use JWT").
*   **Select Template:** Choose a persona (Dev, Architect, QA).
*   **Live Updates:** If files change in your repo while you are on Step 2, Shotgun updates context in the background and shows an "Updating context..." indicator.
*   **Execute:** Click **"Execute Prompt"** to send it to the configured LLM API immediately, OR copy the full payload to your clipboard for use in external tools like ChatGPT or Cursor.

### Step 3: History & Apply
*   **Review:** View the AI's response alongside your original prompt.
*   **Diffs:** The AI output is optimized for `diff` generation.
*   **Audit:** Inspect raw API calls for debugging or token usage analysis.

---

## 4. Installation

### Prerequisites
*   **Go ≥ 1.20**
*   **Node.js LTS**
*   **Wails CLI:** `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Clone & Build
```bash
git clone https://github.com/glebkudr/shotgun_code
cd shotgun_code

# Install frontend dependencies
cd frontend
npm install
cd ..

# Run in Development Mode (Hot Reload)
wails dev

# Build Production Binary
wails build
```
*Binaries will be located in `build/bin/`.*

---

## 5. Configuration

### LLM Setup
Click the **Settings** (gear icon) in the app to configure providers:
1.  **Provider:** Select OpenAI, Gemini, OpenRouter, Ollama, or LM Studio.
2.  **API Key:** Paste your key (stored locally). For Ollama/LM Studio, key is optional.
3.  **Model:** Select your preferred model (e.g., `gpt-4o`, `gemini-2.5-pro`, `claude-3.5-sonnet`).

### Streaming Events & Token Estimation
The backend now exposes prompt streaming and token estimation APIs:

* `ExecuteLLMPromptStream(userTask, finalPrompt) -> requestId`
* `CancelLLMPromptStream(requestId)`
* `EstimateTokens(providerName, model, text)`

Stream lifecycle events emitted to frontend:

* `llmPromptStreamStart`
* `llmPromptStreamChunk`
* `llmPromptStreamEnd`
* `llmPromptStreamError`

Feature flag:

* `LLM_STREAM_ENABLED=true` (default). Set to `false`/`0` to disable streaming and use synchronous fallback.

### Custom Rules
You can define global excludes (like `node_modules`, `dist`, `.git`) and custom prompt instructions that are appended to every request.

### Multiple App Instances
Shotgun is safe to run in multiple independent OS processes (for different repositories). Settings and prompt history writes use cross-process file locks.

*   **Windows/Linux:** Launch Shotgun again (second process).
*   **macOS:** `open -n -a "Shotgun Code"` to force a new app instance.

---

## 6. Output Format

Shotgun generates context optimized for LLM parsing:

```xml
<file path="backend/main.go">
package main
...
</file>

<file path="frontend/src/App.vue">
<template>
...
</template>
</file>
```

This format allows models to understand file boundaries perfectly, enabling accurate multi-file refactoring suggestions.

For binary/non-text files, Shotgun keeps the file entry but replaces content with:

```text
[Binary file content omitted]
```

---

## 7. ⚖️ License & Usage

My name is Gleb Curly, and I am an indie developer making software for a living.

Shotgun is developed and maintained by **Curly's Technology Tmi**. 

This project uses a **Community License** model:

### 1. Free for Small Teams & Non-Commercial Use
You can use Shotgun for free (including modification and internal use) if:
- Your company/team generates **less than $1M USD** in annual revenue.
- You do **not** use the code to build a competing public product.

### 2. Commercial License (Enterprise)
If your annual revenue exceeds **$1M USD**, you are required to purchase a commercial license with a pretty reasonable price.

Please contact me at **glebkudr@gmail.com** for pricing.

See [LICENSE.md](LICENSE.md) for the full legal text.

---

*Shotgun – Load, Aim, Blast your code into the future.*
