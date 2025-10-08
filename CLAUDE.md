# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Shotgun is a desktop application built with Wails (Go backend + Vue 3 frontend) that helps prepare large codebases for LLM consumption. It generates structured "shotgun" outputs combining file trees and file contents, enables iterative prompt refinement through streaming chat with Google Gemini, and can execute prompts via OpenAI Codex CLI.

## Development Commands

### Initial Setup
```bash
go mod tidy                    # Install Go dependencies
cd frontend && npm install     # Install frontend dependencies
```

### Development
```bash
wails dev                      # Run app in dev mode with hot-reload for Vue
                               # Restart required for Go changes
```

### Building
```bash
wails build                    # Build production binaries to build/bin/
```

### Frontend Only
```bash
cd frontend
npm run dev                    # Run Vite dev server
npm run build                  # Build frontend
```

## Architecture

### Backend (Go)

**Core Files:**
- `main.go`: Wails app initialization, menu setup (macOS app/edit menus)
- `app.go`: Primary application logic (~1340 lines)
- `split_diff.go`: Diff parsing utilities

**Key Components in app.go:**

1. **App struct**: Main application state
   - Context management
   - Settings (custom ignore/prompt rules, API key)
   - File watching (Watchman)
   - Context generation (ContextGenerator)
   - Gitignore patterns (both project .gitignore and custom ignore.glob)

2. **ContextGenerator**: Asynchronous shotgun output generation
   - Handles cancellation of previous jobs when new generation starts
   - Emits progress events to frontend (`shotgunContextGenerationProgress`)
   - Size limit: 10MB (`maxOutputSizeBytes`)
   - Output format: Tree view + XML-like `<file path="...">content</file>` blocks
   - Uses goroutines with token-based cancellation

3. **Watchman**: File system monitoring with fsnotify
   - Recursively watches directories respecting ignore patterns
   - Dynamically adds/removes paths as dirs are created/deleted
   - Emits `projectFilesChanged` events
   - Ignores `.git` directories and patterns from .gitignore/ignore.glob
   - Can be refreshed when ignore settings change

4. **Google Gemini Integration**:
   - `CommunicateWithGoogleAI()`: Streaming chat with gemini-2.5-pro
   - API key stored securely in backend (never sent to frontend)
   - Events: `newChunk`, `streamEnd`, `streamError`
   - Config location: `~/.config/shotgun-code/settings.json` (via xdg)

5. **Codex CLI Management**:
   - `CheckCodexCli()`: Checks for `codex` in PATH and verifies authentication via `~/.codex/config.toml`
   - `InstallCodexCli()`: Installs via `npm install -g @openai/codex`
   - `AuthorizeCodexCli()`: Opens terminal window for interactive login via `codex` command
   - `ExecuteCodexCli()`: Runs `codex exec "<prompt>"` in project directory
   - Windows: All operations go through WSL (path conversion via `toWSLPath()`)

### Frontend (Vue 3 + Vite + Tailwind CSS)

**Component Structure:**
```
App.vue
└── MainLayout.vue
    ├── HorizontalStepper.vue (top navigation)
    ├── LeftSidebar.vue (step list + file tree)
    ├── CentralPanel.vue (step content router)
    │   ├── Step1PrepareContext.vue (folder selection)
    │   ├── Step2ComposePrompt.vue (task input, role selection)
    │   ├── Step3Chat.vue (streaming Gemini chat)
    │   └── Step4ApplyPatch.vue (Codex CLI execution)
    ├── BottomConsole.vue (logs)
    ├── CustomRulesModal.vue
    └── FileTree.vue (recursive tree component)
```

**State Management**: Reactive state in MainLayout.vue coordinates all steps.

**Wails Bindings**: Go methods called via `window.go.main.App.*` from frontend.

**Key Frontend Patterns:**
- Event listeners for backend events: `runtime.EventsOn("eventName", handler)`
- File tree uses recursive exclusion checkboxes
- Step navigation supports both linear and non-linear flows
- Role-based workflow: Architect/FindBug roles end at Step 3, Dev/Project continue to Step 4

### Workflow (4 Steps)

1. **Step 1: Prepare Context**
   - Select project folder
   - Display file tree with exclusion checkboxes
   - Respects .gitignore and custom ignore.glob patterns
   - Automatically generates context (tree + file contents)
   - File watcher monitors changes and notifies frontend

2. **Step 2: Compose Prompt**
   - User enters task description
   - Select role/template (Architect, Dev, FindBug, Project)
   - Custom rules can be edited
   - Prompt assembled from: template + task + rules + file context

3. **Step 3: Chat with AI**
   - Streaming chat interface with Google Gemini
   - Initial prompt from Step 2 sent automatically
   - Iterate to refine output
   - **Conditional workflow:**
     - Architect/FindBug: Process ends here (user copies final prompt)
     - Dev/Project: "Use Last Response & Proceed" continues to Step 4

4. **Step 4: Apply Patch**
   - Check for Codex CLI installation and authentication status
   - Auto-install via npm if missing (with user consent)
   - Authenticate via terminal if not logged in
   - Execute final prompt via `codex exec`
   - Display CLI output

## Important Technical Details

### Ignore Patterns
- **Project .gitignore**: Loaded from project root if present
- **Custom ignore.glob**: App-level patterns embedded in binary, stored in config
- Both compiled using `github.com/sabhiram/go-gitignore`
- Patterns applied during tree building and file watching

### Context Generation
- Recursive directory traversal with progress tracking
- Excludes items based on exclusion list from frontend
- Output format: Tree representation + file contents in XML-like blocks
- Cancellable (new generation cancels previous)
- Size-limited (10MB) with `ErrContextTooLong` error

### File Watching
- Uses fsnotify for native OS events
- Tracks explicitly watched directories in map
- Skips Chmod events (irrelevant for content changes)
- Handles Create/Remove/Rename for dynamic directory watching
- Can be stopped/restarted and refreshed when settings change

### Windows/WSL Handling
- Cursor CLI installation and execution happen inside WSL on Windows
- Path conversion: `C:\path` → `/mnt/c/path`
- All shell scripts run with `wsl bash script.sh`
- Frontend project paths converted to WSL paths before CLI execution

### Settings Persistence
- Config file: `~/.config/shotgun-code/settings.json` (Linux/macOS)
- Contains: CustomIgnoreRules, CustomPromptRules, ApiKey
- Loaded on startup, saved on modification
- Defaults to embedded `ignore.glob` content if config missing

### Wails Integration
- Public Go methods auto-bound to `window.go.main.App.*`
- Event emission: `runtime.EventsEmit(ctx, "eventName", data)`
- Frontend listens: `runtime.EventsOn("eventName", callback)`
- Context passed through `app.ctx` from Wails startup

## Common Development Patterns

### Adding a New Go Method Exposed to Frontend
1. Add public method to `App` struct in `app.go`
2. Method automatically available in frontend as `window.go.main.App.MethodName()`
3. Use `runtime.LogInfo/Error/Debug` for logging visible in console

### Adding a New Event from Backend to Frontend
1. Backend: `runtime.EventsEmit(a.ctx, "myEventName", data)`
2. Frontend: `runtime.EventsOn("myEventName", (data) => { ... })`
3. Cleanup: `runtime.EventsOff("myEventName")` in component unmount

### Modifying File Tree Logic
- Core logic in `buildTreeRecursive()` (app.go:142)
- Returns `[]*FileNode` with `IsGitignored` and `IsCustomIgnored` flags
- Frontend FileTree.vue renders recursively with checkbox exclusions
- Changes trigger `RequestShotgunContextGeneration()` via watcher

### Testing Windows/WSL Features
- Set `goruntime.GOOS == "windows"` checks in code
- Test path conversion with `toWSLPath()` (app.go:1308)
- Verify script execution uses `wsl bash script.sh` pattern

## Architecture Documents

Detailed architecture specs in `architecture/app/`:
- `ARCH-CORE-WORKFLOW-V3.md`: 4-step workflow with role-based conditionals and Codex CLI integration (current)
- `ARCH-BACKEND-GEMINI-PROXY-V1.md`: Streaming API proxy design
- `ARCH-FRONTEND-STREAMING-CHAT-V1.md`: Chat UI patterns

## Task Management

Project uses structured task tracking in `tasks/2025-Q3/`:
- Task IDs: `TASK-2025-XXX`
- Markdown format with metadata, acceptance criteria, DoD
- Reference these when implementing features

## Special Files

- `ignore.glob`: Embedded default ignore patterns (images, videos, binaries, etc.)
- `wails.json`: Wails project configuration
- `appicon.{png,ico,icns}`: Application icons for different platforms
