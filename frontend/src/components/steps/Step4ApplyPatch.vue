<template>
  <div class="p-6 flex flex-col h-full">
    <h2 class="text-xl font-semibold text-gray-800 mb-4">Step 4: Execute with Cursor CLI</h2>
    
    <!-- Checking State -->
    <div v-if="cliStatus === 'checking'" class="flex-grow flex justify-center items-center">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
      <p class="text-gray-600 ml-3">Checking for Cursor CLI...</p>
    </div>
    
    <!-- Error State -->
    <div v-else-if="cliStatus === 'error'" class="flex-grow flex justify-center items-center text-center">
      <div class="p-4 border border-red-300 bg-red-50 rounded-md">
        <h3 class="text-lg font-semibold text-red-700 mb-2">An Error Occurred</h3>
        <p v-if="isWslError" class="text-red-600">
          Windows Subsystem for Linux (WSL) not detected or not running.
          <br>Please install WSL and ensure it's operational to use the Cursor CLI integration.
        </p>
        <pre v-else class="text-red-600 text-xs whitespace-pre-wrap">{{ cliError }}</pre>
        <button @click="checkCli" class="mt-4 px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600">Retry Check</button>
      </div>
    </div>
    
    <!-- Not Installed State -->
    <div v-else-if="cliStatus === 'not_installed'" class="flex-grow flex flex-col justify-center items-center text-center">
      <div class="p-6 border border-gray-300 bg-gray-50 rounded-lg shadow-sm max-w-md">
        <h3 class="text-lg font-semibold text-gray-800 mb-2">Cursor CLI Not Found</h3>
        <p class="text-gray-600 mb-4 text-sm">
          To execute the prompt automatically, the Cursor CLI tool (`cursor-agent`) is required.
        </p>
        <button
          @click="installCli"
          :disabled="isInstalling"
          class="px-6 py-2 bg-green-600 text-white font-semibold rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-opacity-50 disabled:bg-gray-400 flex items-center"
        >
          <div v-if="isInstalling" class="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
          {{ isInstalling ? 'Installing...' : 'Install Cursor CLI' }}
        </button>
        <p v-if="installationLog" class="text-xs text-gray-500 mt-3 whitespace-pre-wrap bg-white p-2 border rounded max-h-40 overflow-auto">{{ installationLog }}</p>
      </div>
    </div>
    
    <!-- Installed State -->
    <div v-else-if="cliStatus === 'installed'" class="flex-grow flex flex-col space-y-4">
      <div>
        <h3 class="text-lg font-medium text-gray-800">Cursor CLI is Ready</h3>
        <p class="text-sm text-green-600 font-semibold">✓ Installed at: <code class="bg-gray-100 p-1 rounded text-xs">{{ cliPath }}</code></p>
      </div>
      <div class="flex-grow flex flex-col">
        <label for="cli-output" class="block text-sm font-medium text-gray-700 mb-1">Execution Output:</label>
        <textarea
          id="cli-output"
          :value="executionOutput"
          rows="15"
          readonly
          class="w-full p-2 border border-gray-300 rounded-md shadow-sm bg-gray-50 font-mono text-xs flex-grow"
          placeholder="Output from cursor-agent will appear here..."
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
import { CheckCursorInstallation, InstallCursorCli, ExecuteCliTool } from '../../../wailsjs/go/main/App';
import { LogError as LogErrorRuntime, LogInfo as LogInfoRuntime } from '../../../wailsjs/runtime/runtime';

const props = defineProps({
  platform: { type: String, default: 'unknown' },
  finalPrompt: { type: String, default: '' },
  projectRoot: { type: String, default: '' },
});

const emit = defineEmits(['action']);

const cliStatus = ref('checking'); // 'checking', 'installed', 'not_installed', 'error'
const cliPath = ref('');
const cliError = ref('');
const isInstalling = ref(false);
const installationLog = ref('');
const isExecuting = ref(false);
const executionOutput = ref('');

const isWslError = computed(() => {
  return props.platform === 'windows' && cliError.value.toLowerCase().includes('wsl');
});

async function checkCli() {
  cliStatus.value = 'checking';
  cliError.value = '';
  try {
    const result = await CheckCursorInstallation();
    cliStatus.value = result.status;
    if (result.status === 'installed') {
      cliPath.value = result.path;
      LogInfoRuntime(`Cursor CLI found at: ${result.path}`);
    } else {
      LogInfoRuntime('Cursor CLI not found.');
    }
  } catch (err) {
    cliStatus.value = 'error';
    cliError.value = err.message || String(err);
    LogErrorRuntime(`Error checking for Cursor CLI: ${cliError.value}`);
  }
}

async function installCli() {
  if (!window.confirm('This will download and execute the official installation script from cursor.com to install the `cursor-agent` CLI tool. Do you want to continue?')) {
    installationLog.value = 'Installation cancelled by user.';
    return;
  }
  isInstalling.value = true;
  installationLog.value = 'Starting installation...';
  try {
    await InstallCursorCli();
    installationLog.value = 'Installation successful! Re-checking status...';
    LogInfoRuntime('Cursor CLI installation completed.');
    await checkCli(); // Re-check to update UI
  } catch (err) {
    installationLog.value = `Installation failed:\n${err.message || String(err)}`;
    LogErrorRuntime(`Cursor CLI installation failed: ${installationLog.value}`);
  } finally {
    isInstalling.value = false;
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
    const output = await ExecuteCliTool(props.finalPrompt, props.projectRoot, cliPath.value);
    executionOutput.value = output;
    LogInfoRuntime('Cursor CLI execution successful.');
    emit('action', 'cliExecutionSuccess');
  } catch (err) {
    executionOutput.value = `Execution failed:\n${err.message || String(err)}`;
    LogErrorRuntime(`Cursor CLI execution failed: ${executionOutput.value}`);
  } finally {
    isExecuting.value = false;
  }
}

onMounted(() => {
  checkCli();
});
</script>