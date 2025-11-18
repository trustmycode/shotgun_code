<template>
  <div class="p-4 h-full flex flex-col">
    <!-- Loading State: Always Progress Bar -->
    <div v-if="isLoadingContext" class="flex-grow flex justify-center items-center">
      <div class="text-center">
        <div class="w-64 mx-auto">
          <p class="text-gray-600 mb-1 text-sm">Generating project context...</p>
          <div class="w-full bg-gray-200 rounded-full h-2.5 dark:bg-gray-700">
            <div class="bg-blue-600 h-2.5 rounded-full" :style="{ width: progressBarWidth }"></div>
          </div>
          <p class="text-gray-500 mt-1 text-xs">
            {{ generationProgress.current }} / {{ generationProgress.total > 0 ? generationProgress.total : 'calculating...' }} items
          </p>
        </div>
      </div>
    </div>

    <!-- Content Area (Textarea + Copy Button OR Error Message OR Placeholder) -->
    <div v-else-if="projectRoot" class="mt-0 flex-grow flex flex-col space-y-4">
      <div>
        <label for="user-task-ai-step1" class="block text-sm font-medium text-gray-700 mb-1">Your task for AI:</label>
        <textarea
          id="user-task-ai-step1"
          v-model="localUserTask"
          rows="6"
          class="w-full p-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 text-sm"
          placeholder="Describe what the AI should do..."
        ></textarea>
      </div>

      <div v-if="generatedContext && !generatedContext.startsWith('Error:')" class="flex-grow flex flex-col">
        <div class="flex items-center justify-between mb-2">
          <h3 class="text-md font-medium text-gray-700">Generated Project Context:</h3>
          <div class="flex items-center space-x-2">
            <button
              class="auto-context-button"
              :class="isAutoContextEnabled ? 'auto-context-button--enabled' : 'auto-context-button--disabled'"
              :disabled="!isAutoContextEnabled"
              data-testid="auto-context-btn"
              @click="emit('auto-context')"
            >
              <span>
                {{ props.isAutoContextLoading ? 'Auto selecting…' : 'Auto context' }}
              </span>
            </button>
            <button
              class="text-xs text-blue-600 hover:underline"
              type="button"
              data-testid="setup-api-key-link"
              @click="emit('open-llm-settings')"
            >
              Setup model
            </button>
          </div>
        </div>
        <div
          class="flex items-center justify-end mb-2 text-xs text-gray-600"
          title="Repo scan is attached to your context extraction prompt to better understand the repository structure and extract the right context. Create it on shotgunpro.dev"
        >
          <label class="flex items-center space-x-2 cursor-pointer">
            <input
              type="checkbox"
              v-model="includeRepoScan"
              class="h-3 w-3 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <span>Use repo scan:</span>
            <span class="font-mono">
              {{ repoScanTokensLabel }}
            </span>
            <button
              type="button"
              class="text-blue-600 hover:underline ml-1"
              @click="openRepoScanEditor"
            >
              edit
            </button>
          </label>
        </div>
        <textarea
          :value="generatedContext"
          rows="10"
          readonly
          class="w-full p-2 border border-gray-300 rounded-md shadow-sm bg-gray-50 font-mono text-xs flex-grow"
          placeholder="Context will appear here. If empty, ensure files are selected and not all excluded."
          style="min-height: 150px;"
        ></textarea>
        <button
          v-if="generatedContext"
          @click="copyGeneratedContextToClipboard"
          class="mt-2 px-4 py-1 bg-gray-200 text-gray-700 font-semibold rounded-md hover:bg-gray-300 focus:outline-none focus:ring-2 focus:ring-gray-400 focus:ring-opacity-50 self-start"
        >
          {{ copyButtonText }}
        </button>
      </div>
      <div v-else-if="generatedContext && generatedContext.startsWith('Error:')" class="text-red-500 p-3 border border-red-300 rounded bg-red-50 flex-grow flex flex-col justify-center items-center">
        <h4 class="font-semibold mb-1">Error Generating Context:</h4>
        <pre class="text-xs whitespace-pre-wrap text-left w-full bg-white p-2 border border-red-200 rounded max-h-60 overflow-auto">{{ generatedContext.substring(6).trim() }}</pre>
      </div>
      <p v-else class="text-xs text-gray-500 mt-2 flex-grow flex justify-center items-center">
        Project context will be generated automatically. If empty after generation, ensure files are selected and not all excluded.
      </p>
    </div>

    <!-- Initial message when no project is selected -->
    <p v-else class="text-xs text-gray-500 mt-2 flex-grow flex justify-center items-center">
      Select a project folder to begin.
    </p>

    <RepoScanModal
      :isVisible="isRepoScanModalVisible"
      :initialScan="repoScanContent"
      @save="handleSaveRepoScan"
      @cancel="isRepoScanModalVisible = false"
    />
  </div>
</template>


<script setup>
import { defineProps, ref, computed, defineEmits, watch } from 'vue';
import { ClipboardSetText as WailsClipboardSetText, BrowserOpenURL } from '../../../wailsjs/runtime/runtime';
import { SaveRepoScan, LoadRepoScan } from '../../../wailsjs/go/main/App';
import RepoScanModal from '../RepoScanModal.vue';

const props = defineProps({
  generatedContext: {
    type: String,
    default: ''
  },
  projectRoot: {
    type: String,
    default: ''
  },
  isLoadingContext: { // New prop
    type: Boolean,
    default: false
  },
  generationProgress: { // New prop for progress data
    type: Object,
    default: () => ({ current: 0, total: 0 })
  },
  platform: { // To know if we are on macOS
    type: String,
    default: 'unknown'
  },
  hasActiveLlmKey: {
    type: Boolean,
    default: false
  },
  isAutoContextLoading: {
    type: Boolean,
    default: false
  },
  userTask: {
    type: String,
    default: ''
  }
});

const emit = defineEmits(['auto-context', 'open-llm-settings', 'update:userTask']);

const progressBarWidth = computed(() => {
  if (props.generationProgress && props.generationProgress.total > 0) {
    const percentage = (props.generationProgress.current / props.generationProgress.total) * 100;
    return `${Math.min(100, Math.max(0, percentage))}%`;
  }
  return '0%';
});
const copyButtonText = ref('Copy All');
const localUserTask = ref(props.userTask);
let userTaskInputDebounceTimer = null;

const includeRepoScan = ref(false);
const repoScanTokenCount = ref(0);
const repoScanContent = ref('');
const isRepoScanModalVisible = ref(false);

const repoScanTokensLabel = computed(() => {
  if (repoScanTokenCount.value === 0) {
    return 'empty';
  }
  return `${repoScanTokenCount.value} tokens`;
});

const isAutoContextEnabled = computed(() => {
  if (!props.hasActiveLlmKey || props.isAutoContextLoading) {
    return false;
  }
  if (!localUserTask.value) {
    return false;
  }
  return localUserTask.value.trim().length > 0;
});

watch(() => props.userTask, (newValue) => {
  if (newValue !== localUserTask.value) {
    localUserTask.value = newValue;
  }
});

watch(localUserTask, (currentValue) => {
  clearTimeout(userTaskInputDebounceTimer);
  userTaskInputDebounceTimer = setTimeout(() => {
    if (currentValue !== props.userTask) {
      emit('update:userTask', currentValue);
    }
  }, 300);
});

// Watch for project root changes to load repo scan
watch(() => props.projectRoot, async (newRoot) => {
  if (newRoot) {
    try {
      const content = await LoadRepoScan(newRoot);
      if (content) {
        repoScanContent.value = content;
        includeRepoScan.value = true;
        updateTokenCount(content);
      } else {
        repoScanContent.value = '';
        includeRepoScan.value = false;
        repoScanTokenCount.value = 0;
      }
    } catch (err) {
      console.error("Failed to load repo scan:", err);
      repoScanContent.value = '';
      includeRepoScan.value = false;
      repoScanTokenCount.value = 0;
    }
  } else {
    repoScanContent.value = '';
    includeRepoScan.value = false;
    repoScanTokenCount.value = 0;
  }
}, { immediate: true });

function updateTokenCount(text) {
  // Simple estimation: length / 4
  if (!text) {
    repoScanTokenCount.value = 0;
    return;
  }
  repoScanTokenCount.value = Math.ceil(text.length / 4);
}

async function copyGeneratedContextToClipboard() {
  if (!props.generatedContext) return;
  try {
    await navigator.clipboard.writeText(props.generatedContext);
    //if (props.platform === 'darwin') {
    //  await WailsClipboardSetText(props.generatedContext);
    //} else {
    //  await navigator.clipboard.writeText(props.generatedContext);
    //}
    copyButtonText.value = 'Copied!';
    setTimeout(() => {
      copyButtonText.value = 'Copy All';
    }, 2000);
  } catch (err) {
    console.error('Failed to copy context: ', err);
    if (props.platform === 'darwin' && err) {
      console.error('darvin ClipboardSetText failed for context:', err);
    }
    copyButtonText.value = 'Failed!';
    setTimeout(() => {
      copyButtonText.value = 'Copy All';
    }, 2000);
  }
}

function openRepoScanEditor() {
  isRepoScanModalVisible.value = true;
}

async function handleSaveRepoScan(content) {
  repoScanContent.value = content;
  updateTokenCount(content);
  isRepoScanModalVisible.value = false;
  
  if (content && props.projectRoot) {
    try {
      await SaveRepoScan(props.projectRoot, content);
      includeRepoScan.value = true;
    } catch (err) {
      console.error("Failed to save repo scan:", err);
      // Optionally show error to user
    }
  } else if (!content && props.projectRoot) {
     // If content is empty, maybe we should delete the file? 
     // For now, just saving empty string is fine or we can leave it. 
     // The requirement says "if there is a repo scan... it is picked up automatically".
     // If user clears it, we probably should save empty string.
     try {
      await SaveRepoScan(props.projectRoot, "");
      includeRepoScan.value = false;
    } catch (err) {
      console.error("Failed to clear repo scan:", err);
    }
  }
}
</script>
