<template>
  <div class="p-6 flex flex-col h-full">
    <div class="flex justify-between items-center mb-4">
      <h2 class="text-xl font-semibold text-gray-800">Step 4: Execute with CLI</h2>
      <!-- Executor Selector -->
      <div class="flex items-center p-1 bg-gray-200 rounded-lg">
        <button
          @click="selectedExecutor = 'codex'"
          :class="['px-4 py-1 text-sm font-medium rounded-md', selectedExecutor === 'codex' ? 'bg-white text-blue-600 shadow' : 'text-gray-600 hover:bg-gray-300']"
        >
          Codex
        </button>
        <button
          @click="selectedExecutor = 'claude'"
          :class="['px-4 py-1 text-sm font-medium rounded-md', selectedExecutor === 'claude' ? 'bg-white text-blue-600 shadow' : 'text-gray-600 hover:bg-gray-300']"
        >
          Claude
        </button>
        <button
          @click="selectedExecutor = 'gemini'"
          :class="['px-4 py-1 text-sm font-medium rounded-md', selectedExecutor === 'gemini' ? 'bg-white text-blue-600 shadow' : 'text-gray-600 hover:bg-gray-300']"
        >
          Gemini
        </button>
      </div>
    </div>

    <!-- Checking State -->
    <div v-if="cliStatus === 'checking'" class="flex-grow flex justify-center items-center">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
      <p class="text-gray-600 ml-3">Checking for {{ executorName }} CLI...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="cliStatus === 'error'" class="flex-grow flex justify-center items-center text-center">
      <div class="p-4 border border-red-300 bg-red-50 rounded-md">
        <h3 class="text-lg font-semibold text-red-700 mb-2">An Error Occurred</h3>
        <p v-if="isWslError" class="text-red-600">
          Windows Subsystem for Linux (WSL) not detected or not running.
          <br>Please install WSL and ensure it's operational to use this CLI integration.
        </p>
        <pre v-else class="text-red-600 text-xs whitespace-pre-wrap">{{ cliError }}</pre>
        <button @click="checkCurrentCli" class="mt-4 px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Retry Check</button>
      </div>
    </div>

    <!-- Not Installed State -->
    <div v-else-if="cliStatus === 'not_installed'" class="flex-grow flex flex-col justify-center items-center text-center">
      <!-- Codex Not Installed -->
      <div v-if="selectedExecutor === 'codex'" class="p-6 border border-gray-300 bg-gray-50 rounded-lg shadow-sm max-w-md">
        <h3 class="text-lg font-semibold text-gray-800 mb-2">Codex CLI Not Found</h3>
        <p class="text-gray-600 mb-4 text-sm">
          To execute the prompt automatically, the Codex CLI tool must be installed.
        </p>
        <button
          @click="installCodex"
          :disabled="isInstalling"
          class="px-6 py-2 bg-green-600 text-white font-semibold rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-opacity-50 disabled:bg-gray-400 flex items-center"
        >
          <div v-if="isInstalling" class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
          {{ isInstalling ? 'Installing...' : 'Install Codex CLI (npm)' }}
        </button>
        <p class="text-xs text-gray-500 mt-3">
          Or install manually:
          <code class="block bg-white p-2 rounded mt-2 text-left">npm install -g @openai/codex</code>
        </p>
        <p v-if="installationLog" class="text-xs text-gray-500 mt-3 whitespace-pre-wrap bg-white p-2 border rounded max-h-40 overflow-auto">{{ installationLog }}</p>
      </div>
      <!-- Claude Not Installed -->
      <div v-else-if="selectedExecutor === 'claude'" class="p-6 border border-gray-300 bg-gray-50 rounded-lg shadow-sm max-w-md">
        <h3 class="text-lg font-semibold text-gray-800 mb-2">Claude Code CLI Not Found</h3>
        <p class="text-gray-600 mb-4 text-sm">
          To use Claude, please install the official CLI tool via npm.
        </p>
        <p class="text-xs text-gray-500 mt-3">
          Install manually in your terminal:
          <code class="block bg-white p-2 rounded mt-2 text-left">npm install -g @anthropic-ai/claude-code</code>
        </p>
        <button @click="checkCurrentCli" class="mt-4 px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">I've installed it, check again</button>
      </div>
      <!-- Gemini Not Installed -->
      <div v-else-if="selectedExecutor === 'gemini'" class="p-6 border border-gray-300 bg-gray-50 rounded-lg shadow-sm max-w-md">
        <h3 class="text-lg font-semibold text-gray-800 mb-2">Google Cloud CLI Not Found or Incomplete</h3>
        <p class="text-gray-600 mb-4 text-sm">
          To use Gemini, please install the Google Cloud CLI and the 'gemini' component.
        </p>
        <p class="text-xs text-gray-500 mt-3">
          Follow the official installation guide:
          <a href="https://cloud.google.com/sdk/docs/install" target="_blank" class="text-blue-600 hover:underline">https://cloud.google.com/sdk/docs/install</a>
          <br><br>
          Then, install the component:
          <code class="block bg-white p-2 rounded mt-2 text-left">gcloud components install gemini</code>
        </p>
        <button @click="checkCurrentCli" class="mt-4 px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">I've installed it, check again</button>
      </div>
    </div>

    <!-- Installed but Not Authenticated State -->
    <div v-else-if="cliStatus === 'installed_not_authed'" class="flex-grow flex flex-col justify-center items-center text-center">
      <div class="p-6 border border-yellow-300 bg-yellow-50 rounded-lg shadow-sm max-w-md">
        <h3 class="text-lg font-semibold text-yellow-800 mb-2">{{ executorName }} CLI Not Authenticated</h3>
        <p class="text-yellow-700 mb-4 text-sm">
          {{ executorName }} CLI is installed at: <code class="bg-white p-1 rounded text-xs">{{ cliPath }}</code>
          <br><br>
          You need to authenticate before using it.
        </p>
        <button
          @click="authorizeCli"
          class="px-6 py-2 bg-yellow-600 text-white font-semibold rounded-md hover:bg-yellow-700 focus:outline-none focus:ring-2 focus:ring-yellow-500 focus:ring-opacity-50 disabled:bg-gray-400 flex items-center mx-auto"
        >
          {{ selectedExecutor === 'codex' ? 'Sign in with ChatGPT' : (selectedExecutor === 'claude' ? 'Authenticate via Browser' : 'Login with Google') }}
        </button>
        <p class="text-xs text-yellow-600 mt-3">
          Or authenticate manually:
          <code class="block bg-white p-2 rounded mt-2 text-left">{{ selectedExecutor === 'codex' ? 'codex' : (selectedExecutor === 'claude' ? 'claude setup-token' : 'gcloud auth application-default login') }}</code>
        </p>
        <p class="text-xs text-yellow-600 mt-3 border-t border-yellow-200 pt-3">
          <b>Hint:</b> Check the console below for the full output from the CLI to help diagnose authentication issues.
        </p>
      </div>
    </div>

    <!-- Waiting for Auth Confirmation State -->
    <div v-else-if="cliStatus === 'waiting_for_auth_confirmation'" class="flex-grow flex flex-col justify-center items-center text-center">
      <!-- Codex or Gemini waiting state -->
      <div v-if="selectedExecutor === 'codex' || selectedExecutor === 'gemini'" class="p-6 border border-blue-300 bg-blue-50 rounded-lg shadow-sm max-w-md">
        <h3 class="text-lg font-semibold text-blue-800 mb-2">Terminal Opened for Authentication</h3>
        <p class="text-blue-700 mb-4 text-sm">
          A terminal window has been opened for you to complete the sign-in process.
          <br><br>
          Please complete the authentication in the terminal, then click the button below to verify your login status.
        </p>
        <button @click="checkAuthStatus" :disabled="isCheckingAuthStatus" class="px-6 py-2 bg-blue-600 text-white font-semibold rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50 disabled:bg-gray-400 flex items-center mx-auto">
            <div v-if="isCheckingAuthStatus" class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
            {{ isCheckingAuthStatus ? 'Checking...' : 'Check Login Status' }}
          </button>
        <p class="text-xs text-blue-600 mt-3">
          If the terminal didn't open, run manually:
          <code class="block bg-white p-2 rounded mt-2 text-left">{{ selectedExecutor === 'codex' ? 'codex' : 'gcloud auth application-default login' }}</code>
        </p>
      </div>
      <!-- Claude waiting state -->
      <div v-else-if="selectedExecutor === 'claude'" class="p-6 border border-blue-300 bg-blue-50 rounded-lg shadow-sm max-w-md">
        <h3 class="text-lg font-semibold text-blue-800 mb-2">Awaiting Authentication</h3>
        <div class="flex items-center justify-center my-4">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
        <p class="text-blue-700 mb-4 text-sm">
          Process started. Please complete the login in your browser.
          <br><br>
          The application is now waiting for the authentication token...
        </p>
        <p class="text-xs text-blue-600 mt-3">
          This may take a few moments after you complete the browser login.
        </p>
      </div>
    </div>

    <!-- Installed and Authenticated State -->
    <div v-else-if="cliStatus === 'installed_and_authed'" class="flex-grow flex flex-col space-y-4">
      <div>
        <h3 class="text-lg font-medium text-gray-800">{{ executorName }} CLI is Ready</h3>
        <p class="text-sm text-green-600 font-semibold">✓ Installed and authenticated at: <code class="bg-gray-100 p-1 rounded text-xs">{{ cliPath }}</code></p>
      </div>
      <div class="flex flex-col">
        <label for="final-prompt-input" class="block text-sm font-medium text-gray-700 mb-1">Final Prompt:</label>
        <textarea
          id="final-prompt-input"
          :value="finalPrompt"
          @input="$emit('update:finalPrompt', $event.target.value)"
          rows="8"
          class="w-full p-2 border border-gray-300 rounded-md shadow-sm bg-white font-mono text-xs"
          placeholder="Paste the final prompt here, or use the one from Step 3."
        ></textarea>
      </div>
      <div class="flex-grow flex flex-col">
        <label for="cli-output" class="block text-sm font-medium text-gray-700 mb-1">Execution Output:</label>
        <textarea
          id="cli-output"
          :value="executionOutput"
          rows="15"
          readonly
          class="w-full p-2 border border-gray-300 rounded-md shadow-sm bg-gray-50 font-mono text-xs flex-grow"
          :placeholder="`Output from ${selectedExecutor} will appear here...`"
        ></textarea>
      </div>
      <div class="flex-shrink-0">
        <button
          @click="executeCli"
          :disabled="isExecuting || !finalPrompt"
          class="px-6 py-2 bg-blue-600 text-white font-semibold rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50 disabled:bg-gray-400 flex items-center"
        >
          <div v-if="isExecuting" class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
          {{ isExecuting ? 'Executing...' : `Execute with ${executorName}` }}
        </button>
        <p v-if="!finalPrompt" class="text-xs text-red-500 mt-1">
          The final prompt is empty. Please compose a prompt in Step 2 or enter it manually above.
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, computed } from 'vue';
import { CheckCodexCli, InstallCodexCli, AuthorizeCodexCli, ExecuteCodexCli, CheckClaudeCli, AuthorizeClaudeCli, ExecuteClaudeCli, CheckGeminiCli, AuthorizeGeminiCli, ExecuteGeminiCli } from '../../../wailsjs/go/main/App';
import { LogError as LogErrorRuntime, LogInfo as LogInfoRuntime, EventsOn } from '../../../wailsjs/runtime/runtime';

const props = defineProps({
  platform: { type: String, default: 'unknown' },
  finalPrompt: { type: String, default: '' },
  projectRoot: { type: String, default: '' },
});

const emit = defineEmits(['action', 'update:finalPrompt', 'add-log']);

const selectedExecutor = ref('codex'); // 'codex', 'claude', or 'gemini'

const codexState = ref({ status: 'checking', path: '', error: '' });
const claudeState = ref({ status: 'checking', path: '', error: '' });
const geminiState = ref({ status: 'checking', path: '', error: '' });

// Shared state
const isInstalling = ref(false); // Only for Codex
const installationLog = ref(''); // Only for Codex
const isCheckingAuthStatus = ref(false);
const isExecuting = ref(false);
const executionOutput = ref('');

let unlistenClaudeAuthSuccess;
let unlistenClaudeAuthFailed;

// Computed properties to drive the UI from the selected executor's state
const currentExecutorState = computed(() => {
  if (selectedExecutor.value === 'codex') return codexState.value;
  if (selectedExecutor.value === 'claude') return claudeState.value;
  return geminiState.value;
});

const cliStatus = computed(() => currentExecutorState.value.status);
const cliPath = computed(() => currentExecutorState.value.path);
const cliError = computed(() => currentExecutorState.value.error);
const executorName = computed(() => {
  if (selectedExecutor.value === 'codex') return 'Codex';
  if (selectedExecutor.value === 'claude') return 'Claude';
  return 'Gemini';
});

const isWslError = computed(() => {
  return props.platform === 'windows' && cliError.value.toLowerCase().includes('wsl');
});

async function checkCodex() {
  codexState.value = { status: 'checking', path: '', error: '' };
  try {
    const result = await CheckCodexCli();
    codexState.value.status = result.status;
    if (result.path) {
      codexState.value.path = result.path;
    }
    LogInfoRuntime(`Codex CLI status: ${result.status}`);
  } catch (err) {
    codexState.value.status = 'error';
    codexState.value.error = err.message || String(err);
    LogErrorRuntime(`Error checking for Codex CLI: ${codexState.value.error}`);
  }
}

async function checkClaude() {
  claudeState.value = { status: 'checking', path: '', error: '' };
  try {
    const result = await CheckClaudeCli();
    claudeState.value.status = result.status;
    if (result.path) {
      claudeState.value.path = result.path;
    }
    LogInfoRuntime(`Claude CLI status: ${result.status}`);
  } catch (err) {
    claudeState.value.status = 'error';
    claudeState.value.error = err.message || String(err);
    LogErrorRuntime(`Error checking for Claude CLI: ${claudeState.value.error}`);
  }
}

async function checkGemini() {
  geminiState.value = { status: 'checking', path: '', error: '' };
  try {
    const result = await CheckGeminiCli();
    geminiState.value.status = result.status;
    if (result.path) {
      geminiState.value.path = result.path;
    }
    LogInfoRuntime(`Gemini CLI status: ${result.status}`);
  } catch (err) {
    geminiState.value.status = 'error';
    geminiState.value.error = err.message || String(err);
    LogErrorRuntime(`Error checking for Gemini CLI: ${geminiState.value.error}`);
  }
}

function handleClaudeAuthSuccess() {
  emit('add-log', { message: 'Claude authentication successful! Refreshing status.', type: 'success' });
  checkClaude();
}

function handleClaudeAuthFailed(errorMsg) {
  emit('add-log', { message: `Claude authentication failed: ${errorMsg}`, type: 'error' });
  if (selectedExecutor.value === 'claude') {
    claudeState.value.status = 'installed_not_authed';
  }
}

function checkCurrentCli() {
  if (selectedExecutor.value === 'codex') {
    checkCodex();
  } else if (selectedExecutor.value === 'claude') {
    checkClaude();
  } else {
    checkGemini();
  }
}

async function installCodex() {
  if (!window.confirm('This will run "npm install -g @openai/codex" to install the Codex CLI. Do you want to continue?')) {
    installationLog.value = 'Installation cancelled by user.';
    return;
  }
  isInstalling.value = true;
  installationLog.value = 'Starting installation...';
  try {
    await InstallCodexCli();
    installationLog.value = 'Installation successful! Re-checking status...';
    LogInfoRuntime('Codex CLI installation completed.');
    await checkCodex();
  } catch (err) {
    installationLog.value = `Installation failed:\n${err.message || String(err)}`;
    LogErrorRuntime(`Codex CLI installation failed: ${installationLog.value}`);
  } finally {
    isInstalling.value = false;
  }
}

async function authorizeCli() {
  emit('add-log', { message: `Starting authorization process for ${executorName.value}...`, type: 'info' });
  try {
    if (selectedExecutor.value === 'codex') {
      await AuthorizeCodexCli();
      LogInfoRuntime('Terminal opened for Codex CLI authorization.');
    } else if (selectedExecutor.value === 'claude') {
      if (!props.projectRoot) {
        const msg = 'Project root is not selected. Please select a project on Step 1.';
        alert(msg);
        emit('add-log', { message: `Authorization failed: ${msg}`, type: 'error' });
        return;
      }
      await AuthorizeClaudeCli(props.projectRoot);
      LogInfoRuntime('Authorization process for Claude CLI started.');
    } else if (selectedExecutor.value === 'gemini') {
      await AuthorizeGeminiCli();
      LogInfoRuntime('Terminal opened for Gemini CLI (gcloud) authorization.');
    }
    // Transition to waiting state
    currentExecutorState.value.status = 'waiting_for_auth_confirmation';
    emit('add-log', { message: 'Authorization process started successfully. Please follow the instructions.', type: 'success' });
  } catch (err) {
    const errorMsg = `Failed to open terminal for authorization: ${err.message || String(err)}`;
    LogErrorRuntime(errorMsg);
    emit('add-log', { message: errorMsg, type: 'error' });
    alert(errorMsg);
  }
}

async function checkAuthStatus() {
  isCheckingAuthStatus.value = true;
  emit('add-log', { message: `Checking ${executorName.value} CLI login status...`, type: 'info' });
  try {
    let result;
    if (selectedExecutor.value === 'codex') {
      result = await CheckCodexCli();
      codexState.value.status = result.status;
      if (result.path) codexState.value.path = result.path;
      LogInfoRuntime(`Codex auth check status: ${result.status}`);
    } else if (selectedExecutor.value === 'claude') {
      result = await CheckClaudeCli();
      claudeState.value.status = result.status;
      if (result.path) claudeState.value.path = result.path;
      LogInfoRuntime(`Claude auth check status: ${result.status}`);
    } else {
      result = await CheckGeminiCli();
      geminiState.value.status = result.status;
      if (result.path) geminiState.value.path = result.path;
      LogInfoRuntime(`Gemini auth check status: ${result.status}`);
    }
    emit('add-log', { message: `Check complete. New status: ${result.status}`, type: 'success' });
  } catch (err) {
    const errorMsg = `Error checking auth status: ${err.message || String(err)}`;
    LogErrorRuntime(errorMsg);
    emit('add-log', { message: errorMsg, type: 'error' });
    currentExecutorState.value.status = 'installed_not_authed'; // Revert to let user try again
  } finally {
    isCheckingAuthStatus.value = false;
  }
}

async function executeCli() {
  if (!props.finalPrompt || !props.projectRoot || !cliPath.value) {
    const errorMsg = 'Cannot execute CLI: missing prompt, project root, or CLI path.';
    LogErrorRuntime(errorMsg);
    executionOutput.value = `Error: ${errorMsg}`;
    return;
  }
  isExecuting.value = true;
  executionOutput.value = 'Executing...';
  try {
    let output;
    if (selectedExecutor.value === 'codex') {
      output = await ExecuteCodexCli(props.finalPrompt, props.projectRoot, cliPath.value);
      LogInfoRuntime('Codex CLI execution successful.');
    } else if (selectedExecutor.value === 'claude') {
      output = await ExecuteClaudeCli(props.finalPrompt, props.projectRoot, cliPath.value);
      LogInfoRuntime('Claude CLI execution successful.');
    } else {
      output = await ExecuteGeminiCli(props.finalPrompt, props.projectRoot, cliPath.value);
      LogInfoRuntime('Gemini CLI execution successful.');
    }
    executionOutput.value = output;
    emit('action', 'cliExecutionSuccess');
  } catch (err) {
    executionOutput.value = `Execution failed. Check the console below for detailed logs from the CLI.\n\nError: ${err.message || String(err)}`;
    LogErrorRuntime(`${executorName.value} CLI execution failed: ${executionOutput.value}`);
  } finally {
    isExecuting.value = false;
  }
}

onMounted(() => {
  checkCodex();
  checkClaude();
  checkGemini();
  unlistenClaudeAuthSuccess = EventsOn('claudeAuthSuccess', handleClaudeAuthSuccess);
  unlistenClaudeAuthFailed = EventsOn('claudeAuthFailed', handleClaudeAuthFailed);
});

onBeforeUnmount(() => {
  if (unlistenClaudeAuthSuccess) unlistenClaudeAuthSuccess();
  if (unlistenClaudeAuthFailed) unlistenClaudeAuthFailed();
});
</script>
