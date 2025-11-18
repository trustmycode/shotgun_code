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
              class="px-3 py-1 text-xs border rounded-md"
              :class="props.hasActiveLlmKey ? 'border-blue-500 text-blue-600 hover:bg-blue-50' : 'border-gray-300 text-gray-400 cursor-not-allowed'"
              :disabled="!props.hasActiveLlmKey || props.isAutoContextLoading"
              data-testid="auto-context-btn"
              @click="emit('auto-context')"
            >
              {{ props.isAutoContextLoading ? 'Auto selecting…' : 'Auto context' }}
            </button>
            <button
              class="text-xs text-blue-600 hover:underline"
              type="button"
              data-testid="setup-api-key-link"
              @click="emit('open-llm-settings')"
            >
              Setup API key
            </button>
          </div>
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
  </div>
</template>

<script setup>
import { defineProps, ref, computed, defineEmits, watch } from 'vue';
import { ClipboardSetText as WailsClipboardSetText } from '../../../wailsjs/runtime/runtime';

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
</script>
