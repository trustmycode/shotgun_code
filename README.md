![](https://github.com/user-attachments/assets/058bf4a2-9f81-406c-96ea-795cd4eaf118)

**Tired of Cursor cutting off context, missing your files and folders, and spitting out empty responses?**

Save your context with Shotgun!
→ Prepare a truly GIGANTIC prompt
→ Paste it into **Google AI Studio** and receive a massive patch for your code. 25 free queries per day!
→ Drop that patch into Cursor or Windsurf and apply the entire diff in a single request.

**That means you get 25 huge, fully coherent patches per day for your codebase—absolutely free, thanks to complete context transfer.**

Perfect for dynamically-typed languages:

Python
JavaScript

# Shotgun App

*One‑click codebase "blast" for Large‑Language‑Model workflows.*

---

## 1. What Shotgun Does
Shotgun is a tiny desktop tool that **explodes an entire project into a single,
well‑structured text payload** designed for AI assistants.
Think of it as a rapid‑fire alternative to copy‑pasting dozens of files by hand:

*   **Select a folder → get an instant tree + file dump**
    in a predictable delimiter format (`*#*#*...*#*#*begin … *#*#*end*#*#*`).
*   **Tick check‑boxes to exclude noise** (logs, build artifacts, `node_modules`, …).
*   **Paste the result into ChatGPT, Gemini 2.5, Cursor, etc.**
    to ask for multi‑file edits, refactors, bug fixes, reviews, or documentation.
*   **Receive a diff‑style reply** and apply changes with your favourite patch tool.

Shotgun trades surgical, single‑file prompts for a **"whole‑repository blast"** –
hence the name.

---

## 2. Why You Might Need It

| Scenario                 | Pain Point                             | Shotgun Benefit                                           |
|--------------------------|----------------------------------------|-----------------------------------------------------------|
| **Bulk bug fixing**      | "Please fix X across 12 files."        | Generates a complete snapshot so the LLM sees all usages. |
| **Large‑scale refactor** | IDE refactors miss edge cases.         | LLM gets full context and returns a patch set.            |
| **On‑boarding review**   | New joiner must understand legacy code. | Produce a single, searchable text file to discuss in chat.  |
| **Doc generation**       | Want docs/tests for every exported symbol. | LLM can iterate over full source without extra API calls. |
| **Cursor / CodePilot prompts** | Tools accept pasted context but no filesystem. | Shotgun bridges the gap.                                  |

---

## 3. Key Features

*   ⚡ **Fast tree scan** (Go + Wails backend) – thousands of files in milliseconds.
*   ✅ **Interactive exclude list** – skip folders, temporary files, or secrets.
*   📝 **Deterministic delimiters** – easy for LLMs to parse and for you to split.
*   🔄 **Re‑generate anytime** – tweak the excludes and hit *Shotgun* again.
*   🪶 **Lightweight** – no DB, no cloud; a single native executable plus a Vue UI.
*   🖥️ **Cross‑platform** – Windows, macOS, Linux.

---

## 4. How It Works

(This describes the UI flow. The core `GenerateShotgunOutput` Go function remains the primary backend logic for creating the text payload based on the selected root and exclusions.)

1.  **Step 1: Prepare Context**
    -   User selects a project folder.
    -   The file tree is displayed in the `LeftSidebar`.
    -   User can mark files/folders for exclusion.
    -   The application automatically (or via a button) triggers context generation in Go (`GenerateShotgunOutput`).
    -   The resulting context (tree + file contents) is stored in `shotgunPromptContext` and passed to `CentralPanel.vue`, which in turn makes it available to `Step2ComposePrompt.vue`.
2.  **Step 2: Compose Prompt**
    -   `Step2ComposePrompt.vue` is shown.
    -   It displays the `shotgunPromptContext` (read-only).
    -   User types their instructions for the LLM into another textarea (the prompt) and can edit custom rules.
    -   User clicks "Execute Prompt" to run the configured LLM call.
    -   `MainLayout.vue` sends the `shotgunPromptContext` and the user's prompt to the Go backend, which performs the actual LLM request.
3.  **Step 3: Prompt History**
    -   `Step3ExecutePrompt.vue` is shown.
    -   Displays the chronological history of executed prompts, full payloads, responses, and optional API call metadata.
    -   The stepper/sidebar button for Prompt History is always available so you can review or copy past executions without finishing the earlier steps first.

---

## 5. Installation

### 5.1. Prerequisites
*   **Go ≥ 1.20**   `go version`
*   **Node.js LTS**  `node -v`
*   **Wails CLI**    `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### 5.2. Clone & Bootstrap
```bash
git clone https://github.com/glebkudr/shotgun_code
cd shotgun_code
go mod tidy           # backend deps
cd frontend
npm install           # Vue deps
cd ..
```

### 5.3. Run in Dev Mode
```bash
wails dev
```
Hot‑reloads Vue; restart the command for Go code changes.

### 5.4. Build a Release
```bash
wails build           # binaries land in build/bin/
```

---

## 6. Quick‑Start Workflow

1.  Run `wails dev`. The app window will open.
2.  **Step 1: Prepare Context**
    - Click "Select Project Folder" and choose your repository root.
    - In the left pane (`LeftSidebar`), expand folders and un-tick any items you wish to exclude from the context.
    - Click the "Prepare Project Context & Proceed" button (typically in `Step1CopyStructure.vue` or a similar area for Step 1).
    - The generated context (project tree and file contents) will be prepared internally.
3.  **Step 2: Compose Prompt**
    - The view will switch to Step 2 (`Step2ComposePrompt.vue`).
    - The generated project context from Step 1 will be displayed (read-only).
    - Enter your instructions for the LLM in the "Prompt Editor" textarea, adjust custom rules, and click "Execute Prompt".
    - The backend makes the LLM request and the result is shown in a modal plus stored in history.
4.  **Step 3: Prompt History**
    - The Prompt History view (`Step3ExecutePrompt.vue`) can be opened at any time via the stepper/sidebar button.
    - Browse previous executions, copy raw prompts/responses, or inspect saved API call payloads.
5.  You can navigate between completed steps using the top `HorizontalStepper` or the `LeftSidebar` step list, and the Prompt History button is always enabled for quick access.

---

## 7. Shotgun Output Anatomy
```text
app/
├── main.go
├── app.go
└── frontend/
    ├── App.vue
    └── components/
        └── FileTree.vue (example)

<file path="main.go">
package main
...
</file>

<file path="frontend/components/FileTree.vue">
<template>
...
</template>
</file>
```
*   **Tree View** – quick visual map for you & the LLM.
*   **XML-like File Blocks** – <file path="path/to/file">...</file> for easy parsing by models.

---

## 8. Best Practices
*   **Trim the noise** – exclude lock files, vendored libs, generated assets.
    Less tokens → cheaper & more accurate completions.
*   **Ask for diffs, not whole files** – keeps responses concise.
*   **Iterate** – generate → ask → patch → re‑generate if needed.
*   **Watch token limits** – even million‑token models have practical caps.
    Use Shotgun scopes (root folder vs subfolder) to stay under budget.

---

## 9. Troubleshooting

| Symptom                     | Fix                                                          |
|-----------------------------|--------------------------------------------------------------|
| `wails: command not found`  | Ensure `$GOROOT/bin` or `$HOME/go/bin` is on `PATH`.         |
| Blank window on `wails dev` | Check Node version & reinstall frontend deps.              |
| Output too large            | Split Shotgun runs by subdirectory; or exclude binaries/tests. |

---

## 10. Roadmap

- ✅ **Step 1: Prepare Context**  
  Basic ability to select a project, exclude items, and generate a structured text context.

- ✅ **Step 2: Compose Prompt**  
  - ✅ **Watchman to hot-reload TreeView**  
  - ✅ **Custom rules**

- ☐ **Step 3: Prompt History**  
  Dedicated view for browsing, copying, and auditing executed prompts, responses, and API call metadata.

- ☐ **Next improvements**  
  - ☐ Direct API bridge to send output to OpenAI / Gemini without copy-paste  
  - ☐ CLI version for headless pipelines  
  - **Watch token limits** – even million-token models have practical caps. Use Shotgun scopes (root folder vs subfolder) to stay under budget.  

---

## 11. Contributing
PRs and issues are welcome!
Please format Go code with `go fmt` and follow Vue 3 style guidelines.

**Important**: By submitting a Pull Request, you agree to transfer the copyright of your code to Curly's Technology Tmi. See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

## 12. ⚖️ License & Usage

Shotgun is developed and maintained by **Curly's Technology Tmi**.

This project uses a dual-licensing model to ensure sustainable development:

### 1. Free for Small Teams & Non-Commercial Use

You can use Shotgun for free (including modification and internal use) if:

- Your company/team generates **less than $1M USD** in annual revenue.

- You do **not** use the code to build a public product that directly competes with Shotgun (e.g., releasing a "Shotgun Clone").

See the [LICENSE.md](LICENSE.md) file for details.

### 2. Commercial License (Enterprise)

If your annual revenue exceeds **$1M USD**, you are required to purchase a commercial license to use Shotgun in your products or infrastructure.

Please contact us at **glebkudr@gmail.com** for pricing and terms.

### 3. Contributing

We welcome contributions! Please note that by submitting a Pull Request, you agree to transfer the copyright of your code to Curly's Technology Tmi. Check [CONTRIBUTING.md](CONTRIBUTING.md) for more info.

---

Shotgun – load, aim, blast your code straight into the mind of an LLM.
Iterate faster. Ship better. 
