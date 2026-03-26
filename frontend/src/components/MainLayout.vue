<template>
  <div class="flex flex-col h-screen bg-gray-100">
    <HorizontalStepper :current-step="currentStep" :steps="steps" @navigate="navigateToStep" :key="`hstepper-${currentStep}-${steps.map(s=>s.completed).join('')}`" />
    <div class="flex flex-1 overflow-hidden">
      <LeftSidebar 
        v-if="currentStep !== 3"
        :current-step="currentStep" 
        :steps="steps" 
        :project-root="projectRoot"
        :file-tree-nodes="fileTree"
        :use-gitignore="useGitignore"
        :use-custom-ignore="useCustomIgnore"
        :loading-error="loadingError"
        @navigate="navigateToStep"
        @select-folder="selectProjectFolderHandler"
        @toggle-gitignore="toggleGitignoreHandler"
        @toggle-custom-ignore="toggleCustomIgnoreHandler"
        @toggle-exclude="toggleExcludeNode"
        @custom-rules-updated="handleCustomRulesUpdated"
        @add-log="({message, type}) => addLog(message, type)" />
      <CentralPanel
        :current-step="currentStep"
        @auto-context="requestAutoContextSelection"
        ref="centralPanelRef"
      />
    </div>
    <div 
      @mousedown="startResize"
      class="w-full h-2 bg-gray-300 hover:bg-gray-400 cursor-row-resize select-none"
      title="Resize console height"
    >
    </div>
    <BottomConsole :log-messages="logMessages" :height="consoleHeight" ref="bottomConsoleRef" />
    <LlmSettingsModal
      :is-visible="isLlmSettingsModalVisible"
      :initial-settings="llmSettings"
      @close="closeLlmSettingsModal"
      @saved="handleLlmSettingsSaved"
    />
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted, onBeforeUnmount, nextTick } from 'vue';
import { storeToRefs } from 'pinia';
import HorizontalStepper from './HorizontalStepper.vue';
import LeftSidebar from './LeftSidebar.vue';
import CentralPanel from './CentralPanel.vue';
import BottomConsole from './BottomConsole.vue';
import LlmSettingsModal from './LlmSettingsModal.vue';
import { useProjectStore } from '../stores/projectStore';
import { useContextStore } from '../stores/contextStore';
import { useLLMStore } from '../stores/llmStore';
import { useLogStore } from '../stores/logStore';
import { useFileTreeUtils } from '../composables/useFileTreeUtils';
import {
  ListFiles,
  RequestAutoContextSelection,
  RequestShotgunContextGeneration,
  SelectDirectory as SelectDirectoryGo,
  StartFileWatcher,
  StopFileWatcher,
  SetUseGitignore,
  SetUseCustomIgnore,
  GetLlmSettings,
  HasActiveLlmKey,
  GetAutoContextButtonTexture,
} from '../../wailsjs/go/main/App';
import { EventsOn, Environment } from '../../wailsjs/runtime/runtime';

const currentStep = ref(1);
const steps = ref([
  { id: 1, title: 'Prepare Context', completed: false, description: 'Select project folder, review files, and generate the initial project context for the LLM.' },
  { id: 2, title: 'Compose Prompt', completed: false, description: 'Provide a prompt to the LLM based on the project context to generate a code diff.' },
  { id: 3, title: 'Prompt History', completed: false, description: 'Review previously executed prompts and responses.', alwaysAccessible: true },
]);

const projectStore = useProjectStore();
const contextStore = useContextStore();
const llmStore = useLLMStore();
const logStore = useLogStore();

const {
  projectRoot,
  fileTree,
  loadingError,
  useGitignore,
  useCustomIgnore,
  isFileTreeLoading,
  projectFilesChangedPendingReload,
  platform,
} = storeToRefs(projectStore);
const {
  shotgunPromptContext,
  isGeneratingContext,
  generationProgressData,
  userTask,
  rulesContent,
  finalPrompt,
  isAutoContextLoading,
} = storeToRefs(contextStore);
const {
  hasActiveLlmKey,
  isLlmSettingsModalVisible,
  llmSettings,
} = storeToRefs(llmStore);
const { logMessages } = storeToRefs(logStore);

const centralPanelRef = ref(null); 
const bottomConsoleRef = ref(null);
const MIN_CONSOLE_HEIGHT = 50;
const consoleHeight = ref(MIN_CONSOLE_HEIGHT); // Initial height in pixels
let logEntryId = 0;

function addLog(message, type = 'info', targetConsole = 'bottom') {
  const logEntry = {
    id: ++logEntryId,
    message,
    type,
    timestamp: new Date().toLocaleTimeString()
  };

  if (targetConsole === 'bottom' || targetConsole === 'both') {
    logStore.add(logEntry);
  }
  if (targetConsole === 'step' || targetConsole === 'both') {
    if (centralPanelRef.value && currentStep.value === 3 && centralPanelRef.value.addLogToStep3Console) {
      centralPanelRef.value.addLogToStep3Console(message, type);
    }
  }
}

const manuallyToggledNodes = reactive(new Map());
const autoContextButtonTexture = ref('');
let debounceTimer = null;
let unlistenProjectFilesChanged = null;
let unlistenShotgunContextGenerated = null;
let unlistenShotgunContextError = null;
let unlistenShotgunContextGenerationProgress = null;
let unlistenAutoContextError = null;
const {
  mapDataToTreeRecursive,
  toggleExcludeNode,
  updateAllNodesExcludedState,
  buildExcludedPathsPayload,
  buildIgnoredPathsPayloadForAutoContext,
  normalizeRelPath,
} = useFileTreeUtils({
  useGitignore,
  useCustomIgnore,
  manuallyToggledNodes,
  fileTree,
  addLog,
});

async function selectProjectFolderHandler() {
  isFileTreeLoading.value = true;
  try {
    shotgunPromptContext.value = '';
    isGeneratingContext.value = false;
    const selectedDir = await SelectDirectoryGo(); 
    if (selectedDir) {
      projectRoot.value = selectedDir;
      loadingError.value = '';
      manuallyToggledNodes.clear();
      fileTree.value = [];
      
      await loadFileTree(selectedDir);

      if (!isFileTreeLoading.value && projectRoot.value) {
         debouncedTriggerShotgunContextGeneration();
      }

      steps.value.forEach(s => s.completed = false);
      currentStep.value = 1;
      addLog(`Project folder selected: ${selectedDir}`, 'info', 'bottom');
    } else {
      isFileTreeLoading.value = false;
    }
  } catch (err) {
    console.error("Error selecting directory:", err);
    const errorMsg = "Failed to select directory: " + (err.message || err);
    loadingError.value = errorMsg;
    addLog(errorMsg, 'error', 'bottom');
    isFileTreeLoading.value = false;
  }
}

async function loadFileTree(dirPath) {
  isFileTreeLoading.value = true;
  loadingError.value = '';
  addLog(`Loading file tree for: ${dirPath}`, 'info', 'bottom');
  try {
    const treeData = await ListFiles(dirPath);
    fileTree.value = mapDataToTreeRecursive(treeData, null);
    addLog(`File tree loaded successfully. Root items: ${fileTree.value.length}`, 'info', 'bottom');
  } catch (err) {
    console.error("Error listing files:", err);
    const errorMsg = "Failed to load file tree: " + (err.message || err);
    loadingError.value = errorMsg;
    addLog(errorMsg, 'error', 'bottom');
    fileTree.value = [];
  } finally {
    isFileTreeLoading.value = false;
    checkAndProcessPendingFileTreeReload();
  }
}

function toggleGitignoreHandler(value) {
  useGitignore.value = value;
  addLog(`.gitignore usage changed to: ${value}. Updating tree and watcher...`, 'info', 'bottom');
  SetUseGitignore(value)
    .then(() => addLog(`Watchman instructed to use .gitignore: ${value}`, 'debug'))
    .catch(err => addLog(`Error setting useGitignore in backend: ${err}`, 'error'));
  // Context regeneration is handled by the watch on [fileTree, useGitignore, useCustomIgnore]
  // which calls updateAllNodesExcludedState and debouncedTriggerShotgunContextGeneration.
}

function toggleCustomIgnoreHandler(value) {
  useCustomIgnore.value = value;
  addLog(`Custom ignore rules usage changed to: ${value}. Updating tree and watcher...`, 'info', 'bottom');
  SetUseCustomIgnore(value)
    .then(() => addLog(`Watchman instructed to use custom ignores: ${value}`, 'debug'))
    .catch(err => addLog(`Error setting useCustomIgnore in backend: ${err}`, 'error'));
}

function debouncedTriggerShotgunContextGeneration() {
  if (!projectRoot.value) {
    // Clear context and stop loading if no project root
    shotgunPromptContext.value = ''; // Clear previous context
    generationProgressData.value = { current: 0, total: 0 }; // Reset progress
    // isGeneratingContext will be set to false by the return or by the timeout if it runs
    isGeneratingContext.value = false;
    return;
  }

  if (isFileTreeLoading.value) {
    addLog("Debounced trigger skipped: file tree is loading.", 'debug', 'bottom');
    isGeneratingContext.value = false;
    return;
  }

  if (!isGeneratingContext.value) nextTick(() => isGeneratingContext.value = true);

  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    if (!projectRoot.value) { 
        isGeneratingContext.value = false;
        return;
    }
    if (isFileTreeLoading.value) {
        addLog("Debounced execution skipped: file tree became loading.", 'debug', 'bottom');
        isGeneratingContext.value = false;
        return;
    }

    addLog("Debounced trigger: Requesting shotgun context generation...", 'info');
    
    updateAllNodesExcludedState(fileTree.value);
    generationProgressData.value = { current: 0, total: 0 }; // Reset progress before new request

    const excludedPathsArray = buildExcludedPathsPayload();
 
     RequestShotgunContextGeneration(projectRoot.value, excludedPathsArray)
       .catch(err => {
        const errorMsg = "Error calling RequestShotgunContextGeneration: " + (err.message || err);
        addLog(errorMsg, 'error');
        shotgunPromptContext.value = "Error: " + errorMsg; 
      })
      .finally(() => {
         // isGeneratingContext.value = false;
      });
  }, 750); 
}

function navigateToStep(stepId) {
  const targetStep = steps.value.find(s => s.id === stepId);
  if (!targetStep) return;

  if (stepId === currentStep.value) {
    currentStep.value = stepId;
    return;
  }

  if (targetStep.alwaysAccessible) {
    currentStep.value = stepId;
    return;
  }

  if (targetStep.completed) {
    currentStep.value = stepId;
    return;
  }

  const firstUncompletedStep = steps.value.find(s => !s.completed);
  if (!firstUncompletedStep || stepId === firstUncompletedStep.id) {
    currentStep.value = stepId;
  } else {
    addLog(`Cannot navigate to step ${stepId} yet. Please complete step ${firstUncompletedStep.id}.`, 'warn');
  }
}

const isResizing = ref(false);

function startResize(event) {
  isResizing.value = true;
  document.addEventListener('mousemove', doResize);
  document.addEventListener('mouseup', stopResize);
  event.preventDefault(); 
}

function doResize(event) {
  if (!isResizing.value) return;
  const newHeight = window.innerHeight - event.clientY;
  const minHeight = MIN_CONSOLE_HEIGHT;
  const maxHeight = window.innerHeight * 0.7;
  consoleHeight.value = Math.max(minHeight, Math.min(newHeight, maxHeight));
}

function stopResize() {
  isResizing.value = false;
  document.removeEventListener('mousemove', doResize);
  document.removeEventListener('mouseup', stopResize);
}

onMounted(() => {
  unlistenShotgunContextGenerated = EventsOn("shotgunContextGenerated", (output) => {
    addLog("Wails event: shotgunContextGenerated RECEIVED", 'debug', 'bottom');
    
    if (shotgunPromptContext.value !== output) {
      shotgunPromptContext.value = output;
      // Context changed. If we are NOT on Step 2 (which handles live updates),
      // clear the stale finalPrompt so it regenerates when Step 2 mounts.
      if (currentStep.value !== 2) {
        finalPrompt.value = '';
      }
    }

    isGeneratingContext.value = false;
    addLog(`Shotgun context updated (${output.length} chars).`, 'success');
    const step1 = steps.value.find(s => s.id === 1);
    if (step1 && !step1.completed) {
        step1.completed = true;
    }
    if (currentStep.value === 1 && centralPanelRef.value?.updateStep2ShotgunContext) {
        centralPanelRef.value.updateStep2ShotgunContext(output);
    }
    checkAndProcessPendingFileTreeReload(); // Check after context generation
  });

  unlistenShotgunContextError = EventsOn("shotgunContextError", (errorMsg) => {
    addLog(`Wails event: shotgunContextError RECEIVED: ${errorMsg}`, 'debug', 'bottom');
    shotgunPromptContext.value = "Error: " + errorMsg;
    isGeneratingContext.value = false;
    addLog(`Error generating context: ${errorMsg}`, 'error');
    checkAndProcessPendingFileTreeReload(); // Check after context generation error
  });

  unlistenShotgunContextGenerationProgress = EventsOn("shotgunContextGenerationProgress", (progress) => {
    // console.log("FE: Progress event:", progress); // For debugging in Browser console
    generationProgressData.value = progress;
  });
  unlistenAutoContextError = EventsOn("autoContextError", (message) => {
    isAutoContextLoading.value = false;
    addLog(`Auto context error: ${message}`, 'error', 'bottom');
  });

  // Get platform information
  (async () => {
    try {
      const envInfo = await Environment();
      platform.value = envInfo.platform;
      addLog(`Platform detected: ${platform.value}`, 'debug');
    } catch (err) {
      addLog(`Error getting platform: ${err}`, 'error');
      // platform.value remains 'unknown' as fallback
    }
  })();
  refreshLlmSettingsState();

  (async () => {
    try {
      const texture = await GetAutoContextButtonTexture();
      if (texture && typeof texture === 'string' && texture.length > 0) {
        autoContextButtonTexture.value = texture;
        document.documentElement.style.setProperty(
          '--auto-context-button-bg',
          `url(${texture})`
        );
      }
    } catch (err) {
      addLog(`Failed to load auto-context button texture: ${err?.message || err}`, 'error', 'bottom');
    }
  })();

  unlistenProjectFilesChanged = EventsOn("projectFilesChanged", (changedRootDir) => {
    if (changedRootDir !== projectRoot.value) {
      addLog(`Watchman: Ignoring event for ${changedRootDir}, current root is ${projectRoot.value}`, 'debug');
      return;
    }
    if (isFileTreeLoading.value || isGeneratingContext.value) {
      const wasPending = projectFilesChangedPendingReload.value;
      projectFilesChangedPendingReload.value = true;
      if (!wasPending) {
        addLog("Watchman: File change detected, reload queued as system is busy.", 'info');
      }
    } else {
      addLog("Watchman: File change detected, reloading tree immediately.", 'info');
      loadFileTree(projectRoot.value); // This will set isFileTreeLoading = true
      // debouncedTriggerShotgunContextGeneration will be called by the watcher on fileTree if projectRoot is set
    }
  });
});

onBeforeUnmount(async () => {
  document.removeEventListener('mousemove', doResize);
  document.removeEventListener('mouseup', stopResize);
  clearTimeout(debounceTimer);
  if (projectRoot.value) {
    await StopFileWatcher().catch(err => console.error("Error stopping file watcher on unmount:", err));
    addLog(`File watcher stopped on component unmount for ${projectRoot.value}`, 'debug');
  }
  if (unlistenProjectFilesChanged) {
    unlistenProjectFilesChanged();
  }
  if (unlistenShotgunContextGenerated) {
    unlistenShotgunContextGenerated();
  }
  if (unlistenShotgunContextError) {
    unlistenShotgunContextError();
  }
  if (unlistenShotgunContextGenerationProgress) {
    unlistenShotgunContextGenerationProgress();
  }
  if (unlistenAutoContextError) {
    unlistenAutoContextError();
  }
});

watch([fileTree, useGitignore, useCustomIgnore], ([newFileTree, newUseGitignore, newUseCustomIgnore], [oldFileTree, oldUseGitignore, oldUseCustomIgnore]) => {
  if (isFileTreeLoading.value) {
    addLog("Watcher triggered during file tree load, generation deferred.", 'debug', 'bottom');
    return;
  }
  
  addLog("Watcher detected changes in fileTree, useGitignore, or useCustomIgnore. Re-evaluating context.", 'debug', 'bottom');
  updateAllNodesExcludedState(fileTree.value);
  debouncedTriggerShotgunContextGeneration();
}, { deep: true });

watch(finalPrompt, (prompt) => {
  if (currentStep.value === 2 && prompt && steps.value[0].completed) {
    const step2 = steps.value.find((s) => s.id === 2);
    if (step2 && !step2.completed) {
      step2.completed = true;
      addLog("Step 2: Prompt composed. Ready to proceed to Step 3.", "success", "bottom");
    }
  }
});

watch(projectRoot, async (newRoot, oldRoot) => {
  if (oldRoot) {
    await StopFileWatcher().catch(err => addLog(`Error stopping watcher for ${oldRoot}: ${err}`, 'error'));
    addLog(`File watcher stopped for ${oldRoot}`, 'debug');
  }
  if (newRoot) {
    // Existing logic to loadFileTree, clear errors, etc., happens in selectProjectFolderHandler
    // which sets projectRoot. Here we just ensure the watcher starts for the new root.
    await StartFileWatcher(newRoot).catch(err => addLog(`Error starting watcher for ${newRoot}: ${err}`, 'error'));
    addLog(`File watcher started for ${newRoot}`, 'debug');
  } else {
    // Project root cleared, ensure watcher is stopped (already handled by oldRoot check if it was set)
    fileTree.value = [];
    shotgunPromptContext.value = '';
    loadingError.value = '';
    manuallyToggledNodes.clear();
    isGeneratingContext.value = false; // Reset generation state
    projectFilesChangedPendingReload.value = false; // Reset pending reload
  }
}, { immediate: false }); // 'immediate: false' to avoid running on initial undefined -> '' or '' -> initial value if set by default

// Helper function to process pending reloads
function checkAndProcessPendingFileTreeReload() {
  if (projectFilesChangedPendingReload.value && !isFileTreeLoading.value && !isGeneratingContext.value) {
    projectFilesChangedPendingReload.value = false;
    addLog("Watchman: Processing queued file tree reload.", 'info');
    // It's important that loadFileTree correctly sets isFileTreeLoading to true at its start
    // and that subsequent context generation is also handled.
    loadFileTree(projectRoot.value);
  }
}

function handleCustomRulesUpdated() {
  addLog("Custom ignore rules updated by user. Reloading file tree.", 'info');
  if (projectRoot.value) {
    // This will call ListFiles in Go, which will use the new custom rules from app.settings.
    // The new tree will have updated IsCustomIgnored flags.
    // The watch on fileTree (and its subsequent call to debouncedTriggerShotgunContextGeneration)
    // will then handle regenerating the context.
    loadFileTree(projectRoot.value);
  }
}

async function refreshLlmSettingsState() {
  try {
    llmSettings.value = await GetLlmSettings();
    hasActiveLlmKey.value = await HasActiveLlmKey();
  } catch (err) {
    addLog(`Failed to load LLM settings: ${err?.message || err}`, 'error', 'bottom');
  }
}

function openLlmSettingsModal() {
  isLlmSettingsModalVisible.value = true;
}

function closeLlmSettingsModal() {
  isLlmSettingsModalVisible.value = false;
}

async function handleLlmSettingsSaved() {
  await refreshLlmSettingsState();
  addLog('LLM settings updated.', 'success', 'bottom');
}

function applyAutoSelection(selectedRelativePaths) {
  if (!Array.isArray(selectedRelativePaths) || selectedRelativePaths.length === 0) {
    addLog('Auto context returned an empty selection.', 'warn', 'bottom');
    return;
  }
  const normalizedSet = new Set(
    selectedRelativePaths.map((path) => normalizeRelPath(path)).filter((path) => path && path !== '.')
  );
  if (normalizedSet.size === 0) {
    addLog('Auto context did not include any valid paths.', 'warn', 'bottom');
    return;
  }

  const markNode = (node) => {
    if (!node) return false;
    const normalized = normalizeRelPath(node.relPath);
    let includeSelf = normalized === '' || normalized === '.' || normalizedSet.has(normalized);
    if (node.children && node.children.length > 0) {
      let childIncluded = false;
      node.children.forEach((child) => {
        if (markNode(child)) {
          childIncluded = true;
        }
      });
      includeSelf = includeSelf || childIncluded;
    }
    node.excluded = !includeSelf;
    manuallyToggledNodes.set(node.relPath, node.excluded);
    return includeSelf;
  };

  manuallyToggledNodes.clear();
  fileTree.value.forEach((node) => markNode(node));
  updateAllNodesExcludedState(fileTree.value);
  addLog(`Auto context selected ${normalizedSet.size} paths.`, 'success', 'bottom');
  debouncedTriggerShotgunContextGeneration();
}

async function requestAutoContextSelection() {
  if (!projectRoot.value) {
    addLog('Select a project folder before running auto context.', 'warn', 'bottom');
    return;
  }
  if (!hasActiveLlmKey.value) {
    addLog('Configure an LLM provider before requesting auto context.', 'warn', 'bottom');
    openLlmSettingsModal();
    return;
  }
  if (isAutoContextLoading.value) {
    return;
  }
  isAutoContextLoading.value = true;
  addLog('Requesting auto context selection…', 'info', 'bottom');
  try {
    // For Auto context we want the project tree that is NOT filtered by user selections,
    // only reduced by .gitignore and ignore.glob rules (when enabled).
    const excludedPathsArray = buildIgnoredPathsPayloadForAutoContext();
    const selection = await RequestAutoContextSelection(
      projectRoot.value,
      excludedPathsArray,
      userTask.value || ''
    );
    if (Array.isArray(selection) && selection.length > 0) {
      applyAutoSelection(selection);
    } else {
      addLog('Auto context call completed but returned no files.', 'warn', 'bottom');
    }
  } catch (err) {
    addLog(`Auto context failed: ${err?.message || err}`, 'error', 'bottom');
  } finally {
    isAutoContextLoading.value = false;
  }
}

</script>

<style scoped>
.flex-1 {
  min-height: 0;
}
</style> 
