<template>
  <div class="p-6 flex flex-col h-full">
    <h2 class="text-xl font-semibold text-gray-800 mb-4">Step 4: Execute with Codex CLI</h2>
    
    <!-- Checking State -->
    <div v-if="cliStatus === 'checking'" class="flex-grow flex justify-center items-center">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
      <p class="text-gray-600 ml-3">Checking for Codex CLI...</p>
    </div>
    
    <!-- Error State -->
    <div v-else-if="cliStatus === 'error'" class="flex-grow flex justify-center items-center text-center">
      <div class="p-4 border border-red-300 bg-red-50 rounded-md">
        <h3 class="text-lg font-semibold text-red-700 mb-2">An Error Occurred</h3>
        <p v-if="isWslError" class="text-red-600">
          Windows Subsystem for Linux (WSL) not detected or not running.
          <br>Please install WSL and ensure it's operational to use the Codex CLI integration.
        </p>
        <pre v-else class="text-red-600 text-xs whitespace-pre-wrap">{{ cliError }}</pre>
        <button @click="checkCli" class="mt-4 px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Retry Check</button>
      </div>
    </div>
    
    <!-- Not Installed State -->
    <div v-else-if="cliStatus === 'not_installed'" class="flex-grow flex flex-col justify-center items-center text-center">
      <div class="p-6 border border-gray-300 bg-gray-50 rounded-lg shadow-sm max-w-md">
        <h3 class="text-lg font-semibold text-gray-800 mb-2">Codex CLI Not Found</h3>
        <p class="text-gray-600 mb-4 text-sm">
          To execute the prompt automatically, the Codex CLI tool must be installed.
        </p>
        <button
          @click="installCli"
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
    </div>

    <!-- Installed but Not Authenticated State -->
    <div v-else-if="cliStatus === 'installed_not_authed'" class="flex-grow flex flex-col justify-center items-center text-center">
      <div class="p-6 border border-yellow-300 bg-yellow-50 rounded-lg shadow-sm max-w-md">
        <h3 class="text-lg font-semibold text-yellow-800 mb-2">Codex CLI Not Authenticated</h3>
        <p class="text-yellow-700 mb-4 text-sm">
          Codex CLI is installed at: <code class="bg-white p-1 rounded text-xs">{{ cliPath }}</code>
          <br><br>
          You need to authenticate before using it.
        </p>
        <button
          @click="authorizeCli"
          class="px-6 py-2 bg-yellow-600 text-white font-semibold rounded-md hover:bg-yellow-700 focus:outline-none focus:ring-2 focus:ring-yellow-500 focus:ring-opacity-50 disabled:bg-gray-400 flex items-center mx-auto"
        >
          Sign in with ChatGPT
        </button>
        <p class="text-xs text-yellow-600 mt-3">
          Or authenticate manually:
          <code class="block bg-white p-2 rounded mt-2 text-left">codex</code>
        </p>
      </div>
    </div>

    <!-- Waiting for Auth Confirmation State -->
    <div v-else-if="cliStatus === 'waiting_for_auth_confirmation'" class="flex-grow flex flex-col justify-center items-center text-center">
      <div class="p-6 border border-blue-300 bg-blue-50 rounded-lg shadow-sm max-w-md">
        <h3 class="text-lg font-semibold text-blue-800 mb-2">Terminal Opened for Authentication</h3>
        <p class="text-blue-700 mb-4 text-sm">
          A terminal window has been opened for you to sign in with your ChatGPT account.
          <br><br>
          Please complete the authentication process in the terminal window, then click the button below to verify your login status.
        </p>
        <button
          @click="checkAuthStatus"
          :disabled="isCheckingAuthStatus"
          class="px-6 py-2 bg-blue-600 text-white font-semibold rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50 disabled:bg-gray-400 flex items-center mx-auto"
        >
          <div v-if="isCheckingAuthStatus" class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
          {{ isCheckingAuthStatus ? 'Checking...' : 'Check Login Status' }}
        </button>
        <p class="text-xs text-blue-600 mt-3">
          If the terminal didn't open, run manually:
          <code class="block bg-white p-2 rounded mt-2 text-left">codex</code>
        </p>
      </div>
    </div>

    <!-- Installed and Authenticated State -->
    <div v-else-if="cliStatus === 'installed_and_authed'" class="flex-grow flex flex-col space-y-4">
      <div>
        <h3 class="text-lg font-medium text-gray-800">Codex CLI is Ready</h3>
        <p class="text-sm text-green-600 font-semibold">✓ Installed and authenticated at: <code class="bg-gray-100 p-1 rounded text-xs">{{ cliPath }}</code></p>
      </div>
      <div class="flex-grow flex flex-col">
        <label for="cli-output" class="block text-sm font-medium text-gray-700 mb-1">Execution Output:</label>
        <textarea
          id="cli-output"
          :value="executionOutput"
          rows="15"
          readonly
          class="w-full p-2 border border-gray-300 rounded-md shadow-sm bg-gray-50 font-mono text-xs flex-grow"
          placeholder="Output from codex will appear here..."
        ></textarea>
      </div>
      <div class="flex-shrink-0">
        <button
          @click="executeCli"
          :disabled="isExecuting || !finalPrompt"
          class="px-6 py-2 bg-blue-600 text-white font-semibold rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50 disabled:bg-gray-400 flex items-center"
        >
          <div v-if="isExecuting" class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
          {{ isExecuting ? 'Executing...' : 'Execute with Final Prompt' }}
        </button>
        <p v-if="!finalPrompt" class="text-xs text-red-500 mt-1">
          The final prompt is empty. Please compose a prompt in Step 2.
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue';
import { CheckCodexCli, InstallCodexCli, AuthorizeCodexCli, ExecuteCodexCli } from '../../../wailsjs/go/main/App';
import { LogError as LogErrorRuntime, LogInfo as LogInfoRuntime } from '../../../wailsjs/runtime/runtime';

const props = defineProps({
  platform: { type: String, default: 'unknown' },
  finalPrompt: { type: String, default: '' },
  projectRoot: { type: String, default: '' },
});

const emit = defineEmits(['action']);

const cliStatus = ref('checking'); // 'checking', 'installed_and_authed', 'installed_not_authed', 'waiting_for_auth_confirmation', 'not_installed', 'error'
const cliPath = ref('');
const cliError = ref('');
const isInstalling = ref(false);
const installationLog = ref('');
const isCheckingAuthStatus = ref(false);
const isExecuting = ref(false);
const executionOutput = ref('');

const isWslError = computed(() => {
  return props.platform === 'windows' && cliError.value.toLowerCase().includes('wsl');
});

async function checkCli() {
  cliStatus.value = 'checking';
  cliError.value = '';
  try {
    const result = await CheckCodexCli();
    cliStatus.value = result.status;
    if (result.status === 'installed_and_authed' || result.status === 'installed_not_authed') {
      cliPath.value = result.path;
      LogInfoRuntime(`Codex CLI found at: ${result.path} (status: ${result.status})`);
    } else {
      LogInfoRuntime('Codex CLI not found.');
    }
  } catch (err) {
    cliStatus.value = 'error';
    cliError.value = err.message || String(err);
    LogErrorRuntime(`Error checking for Codex CLI: ${cliError.value}`);
  }
}

async function installCli() {
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
    await checkCli();
  } catch (err) {
    installationLog.value = `Installation failed:\n${err.message || String(err)}`;
    LogErrorRuntime(`Codex CLI installation failed: ${installationLog.value}`);
  } finally {
    isInstalling.value = false;
  }
}

async function authorizeCli() {
  try {
    await AuthorizeCodexCli();
    LogInfoRuntime('Terminal opened for Codex CLI authorization. Please complete authentication in the terminal.');
    // Transition to waiting state
    cliStatus.value = 'waiting_for_auth_confirmation';
  } catch (err) {
    LogErrorRuntime(`Failed to open terminal for authorization: ${err.message || String(err)}`);
    alert(`Failed to open terminal: ${err.message || String(err)}`);
  }
}

async function checkAuthStatus() {
  isCheckingAuthStatus.value = true;
  try {
    const result = await CheckCodexCli();
    if (result.status === 'installed_and_authed') {
      cliStatus.value = 'installed_and_authed';
      cliPath.value = result.path;
      LogInfoRuntime(`Codex CLI authenticated successfully at: ${result.path}`);
    } else if (result.status === 'installed_not_authed') {
      // Still not authenticated, go back to auth prompt
      cliStatus.value = 'installed_not_authed';
      LogInfoRuntime('Authentication not completed yet. Please try again.');
    } else {
      // Unexpected state
      cliStatus.value = result.status;
    }
  } catch (err) {
    LogErrorRuntime(`Error checking auth status: ${err.message || String(err)}`);
    cliStatus.value = 'installed_not_authed';
  } finally {
    isCheckingAuthStatus.value = false;
  }
}

async function executeCli() {
  if (!props.finalPrompt || !props.projectRoot || !cliPath.value) {
    LogErrorRuntime('Cannot execute CLI: missing prompt, project root, or CLI path.');
    executionOutput.value = 'Error: Missing prompt, project root, or CLI path. Cannot execute.';
    return;
  }
  isExecuting.value = true;
  executionOutput.value = 'Executing...';
  try {
    const output = await ExecuteCodexCli(props.finalPrompt, props.projectRoot, cliPath.value);
    executionOutput.value = output;
    LogInfoRuntime('Codex CLI execution successful.');
    emit('action', 'cliExecutionSuccess');
  } catch (err) {
    executionOutput.value = `Execution failed:\n${err.message || String(err)}`;
    LogErrorRuntime(`Codex CLI execution failed: ${executionOutput.value}`);
  } finally {
    isExecuting.value = false;
  }
}

onMounted(() => {
  checkCli();
});
</script>