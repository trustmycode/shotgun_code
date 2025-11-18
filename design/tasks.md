# Shotgun Project Task Log

## Core Implemented Functionality

Below is a list of key features implemented in the Shotgun application.

### 1. Basic application structure
*   **Description:** Setting up the main application skeleton using Go (backend), Wails (bridge), and Vue.js (frontend).
*   **Status:** Implemented.

### 2. Project directory selection
*   **Description:** Implemented the ability for the user to choose the project root directory via the system dialog.
*   **Status:** Implemented.

### 3. Display of file and folder tree
*   **Description:** Dynamic construction and display of the hierarchical structure of the selected project in the `FileTree.vue` component. The root of the tree represents the selected directory.
*   **Status:** Implemented.

### 4. File/folder exclusion mechanism
*   **Description:** Users can mark files and folders in the tree to exclude them from the generated context. Exclusion state is controlled via checkboxes.
*   **Status:** Implemented.

### 5. Generation of project text context ("Shotgun Output")
*   **Description:** The backend forms a single text block that includes the project tree and the contents of non‑excluded files.
*   **Status:** Implemented.

### 6. Multi‑step user interface
*   **Description:** Implemented a UI that guides the user through the process:
    *   **Step 1: Prepare context:** Project selection, exclusion management, automatic context generation.
    *   **Step 2: Compose prompt:** Entering the task for the LLM, using/editing prompt rules, viewing file context, building the final prompt using templates.
    *   **Step 3: Execute prompt:** Placeholder UI with instructions.
    *   **Step 4: Apply patch:** Placeholder UI that imitates a patch editor.
*   **Status:** Implemented (Steps 3 and 4 are placeholders).

### 7. Asynchronous context generation
*   **Description:** Project context is generated asynchronously, with the ability to cancel the previous job when a new request is made, and using debouncing to avoid redundant work.
*   **Status:** Implemented.

### 8. Progress reporting and context size limitation
*   **Description:** A progress bar is displayed while the context is being generated. A maximum size limit for the generated context is implemented to prevent performance and memory issues.
*   **Status:** Implemented.

### 9. UI logging system
*   **Description:** System action, error, and warning messages are shown in the bottom console (`BottomConsole.vue`), whose height can be adjusted by the user.
*   **Status:** Implemented.

### 10. Filesystem watcher (`Watchman`)
*   **Description:** The `Watchman` component tracks changes in the filesystem of the selected project (creation, deletion, renaming of files/folders) using `fsnotify`. When changes are detected, it triggers reloading of the file tree and, as a result, regeneration of the context.
*   **Status:** Implemented.

### 11. Configuration management
*   **Description:**
    *   Application settings (custom ignore rules, prompt rules) are stored in `settings.json` in the user's configuration directory (`xdg`).
    *   Default custom ignore rules are embedded from the `ignore.glob` file. Default prompt rules are defined in code.
    *   Editing of custom rules is implemented via the modal window (`CustomRulesModal.vue`).
    *   Support for rules from the project's `.gitignore` file is integrated.
*   **Status:** Implemented.

### 12. Cross‑platform support and UI/UX improvements
*   **Description:** Basic cross‑platform builds are supported. A system menu for macOS has been added. Platform detection is implemented for platform‑specific behavior (for example, copying to clipboard).
*   **Status:** Implemented.

### 13. Prompt templates
*   **Description:** On Step 2 the user can choose from several predefined prompt templates (Dev, Architect, Find Bug, Project Manager), which are loaded from `.md` files in `design/prompts/`.
*   **Status:** Implemented.

---
This list reflects the main development stages and the current state of the application's functionality.

# Project Tasks

## Phase 1: Initial ignore.glob Implementation (Completed)

- [x] Create `ignore.glob` with default media file patterns.
- [x] Modify `app.go` (`ListFiles`, `buildTree`) to parse `ignore.glob` and combine its rules with `.gitignore`.

## Phase 2: Configurable Ignore Patterns

- [x] Move patterns from `ignore.glob` to application settings so the user can edit them via the application interface. (Custom rules are now stored in app config (`settings.json` via `xdg`), editable via UI modal. Default rules are embedded from the repository's `ignore.glob` file.)
- [x] Ensure saving of custom user patterns between application sessions. (Configuration is saved to `settings.json` using `github.com/adrg/xdg` for persistence across sessions.)

- [x] Add platform detection (using Wails Environment API) in MainLayout.vue and pass it down to CentralPanel and step components.
- [x] In Step1PrepareContext.vue and Step2ComposePrompt.vue, use WailsClipboardSetText for macOS (darwin), otherwise use navigator.clipboard for copying to clipboard.
- [x] Update CentralPanel.vue and MainLayout.vue to forward platform prop.
- [x] Update prop definitions and usages in all affected components.
- [x] Update Go main.go to use os.ReadFile and add menu for macOS.

- [ ] Improve error handling and user feedback in the UI.

### Testing

