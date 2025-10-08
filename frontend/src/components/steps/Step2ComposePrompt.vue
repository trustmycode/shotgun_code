<template>
  <div class="p-4 h-full flex flex-col">
    <p class="text-gray-700 mb-4 text-center text-sm">
      Write the task for the LLM in the central column and copy the final prompt
    </p>

    <CustomRulesModal
      :is-visible="isPromptRulesModalVisible"
      :initial-rules="currentPromptRulesForModal"
      title="Edit Custom Prompt Rules"
      ruleType="prompt"
      @save="handleSavePromptRules"
      @cancel="handleCancelPromptRules"
    />

    <div class="flex-grow flex flex-row space-x-4 overflow-hidden">
      <div :class="['flex flex-col space-y-3 overflow-y-auto p-2 border border-gray-200 rounded-md bg-gray-50', isPromptVisible ? 'w-1/2' : 'w-full']">
        <div>
          <label for="user-task-ai" class="block text-sm font-medium text-gray-700 mb-1">Your task for AI:</label>
          <textarea
            id="user-task-ai"
            v-model="localUserTask"
            rows="15"
            class="w-full p-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 text-sm"
            placeholder="Describe what the AI should do..."
          ></textarea>
        </div>
        <div class="flex items-center space-x-3 pt-2">
          <select
            v-model="selectedPromptTemplateKey"
            class="p-1 border border-gray-300 rounded-md text-sm focus:ring-blue-500 focus:border-blue-500"
            :disabled="isLoadingFinalPrompt"
            title="Select prompt template"
          >
            <option v-for="(template, key) in promptTemplates" :key="key" :value="key">
              {{ template.name }}
            </option>
          </select>
          <div class="flex items-center space-x-2">
            <button
              @click="handleCopy"
              :disabled="!localUserTask.trim()"
              class="px-3 py-1 bg-gray-200 text-gray-800 text-sm font-semibold rounded-md hover:bg-gray-300 focus:outline-none focus:ring-2 focus:ring-gray-400 focus:ring-opacity-50 disabled:bg-gray-300"
            >
              {{ copyButtonText }}
            </button>
            <button
              @click="handleNext"
              :disabled="!localUserTask.trim() || isProcessingNext"
              class="px-3 py-1 bg-blue-500 text-white text-sm font-semibold rounded-md hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50 disabled:bg-gray-300"
            >
              <span v-if="isProcessingNext">Processing...</span>
              <span v-else>Next</span>
            </button>
          </div>
        </div>

        <div>
          <label for="rules-content" class="block text-sm font-medium text-gray-700 mb-1 flex items-center">
            Custom rules:
            <button @click="openPromptRulesModal" title="Edit custom prompt rules" class="ml-2 p-0.5 hover:bg-gray-200 rounded text-xs">⚙️</button>
          </label>
          <textarea
            id="rules-content"
            :value="rulesContent"
            @input="e => emit('update:rulesContent', e.target.value)"
            rows="8"
            class="w-full p-2 border border-gray-300 rounded-md shadow-sm bg-gray-100 text-sm font-mono"
            placeholder="Rules for AI..."
          ></textarea>
        </div>

        <div>
          <label for="file-list-context" class="block text-sm font-medium text-gray-700 mb-1">Files to include:</label>
          <textarea
            id="file-list-context"
            :value="props.fileListContext"
            rows="20"
            readonly
            class="w-full p-2 border border-gray-300 rounded-md shadow-sm bg-gray-100 font-mono text-xs"
            placeholder="File list from Step 1 (Prepare Context) will appear here..."
            style="min-height: 150px;"
          ></textarea>
        </div>
      </div>

      <div v-if="isPromptVisible" class="w-1/2 flex flex-col overflow-y-auto p-2 border border-gray-200 rounded-md bg-white">
        <div class="flex justify-between items-center mb-2">
          <h3 class="text-md font-medium text-gray-700">Final Prompt:</h3>
          <div class="flex items-center space-x-3">
            <span v-show="!isLoadingFinalPrompt" :class="['text-xs font-medium', charCountColorClass]" :title="tooltipText">
              ~{{ approximateTokens }} tokens
            </span>
            <button @click="isPromptVisible = false" class="px-3 py-1 bg-gray-200 text-gray-700 text-xs font-semibold rounded-md hover:bg-gray-300">Hide</button>
          </div>
        </div>

        <div v-if="isLoadingFinalPrompt" class="flex-grow flex justify-center items-center">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
          <p class="text-gray-500 ml-2">Updating prompt...</p>
        </div>

        <textarea
          v-else
          :value="props.finalPrompt"
          @input="e => emit('update:finalPrompt', e.target.value)"
          rows="20"
          class="w-full p-2 border border-gray-300 rounded-md shadow-sm font-mono text-xs flex-grow"
          placeholder="The final prompt will be generated here..."
          style="min-height: 300px;"
        ></textarea>
         <p class="text-xs text-gray-500 mt-1">
            The prompt updates automatically. Manual changes to this field may be overwritten when source data (task, rules, file list) is updated.
        </p>
      </div>
      <div v-else class="flex justify-center items-center p-2">
        <button @click="isPromptVisible = true" class="px-4 py-2 bg-gray-200 text-gray-800 font-semibold rounded-md hover:bg-gray-300">
          Show Final Prompt
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, computed } from 'vue';
import { GetCustomPromptRules, SetCustomPromptRules, AssembleFinalPrompt } from '../../../wailsjs/go/main/App'; // <--- 1. ИМПОРТИРУЕМ НОВУЮ ФУНКЦИЮ
import { LogInfo as LogInfoRuntime, LogError as LogErrorRuntime } from '../../../wailsjs/runtime/runtime';
import CustomRulesModal from '../CustomRulesModal.vue';

import devTemplateContentFromFile from '../../../../design/prompts/prompt_makeDiffGitFormat.md?raw';
import architectTemplateContentFromFile from '../../../../design/prompts/prompt_makePlan.md?raw';
import findBugTemplateContentFromFile from '../../../../design/prompts/prompt_analyzeBug.md?raw';
import projectManagerTemplateContentFromFile from '../../../../design/prompts/prompt_projectManager.md?raw';

const props = defineProps({
  fileListContext: {
    type: String,
    default: ''
  },
  platform: { // To know if we are on macOS
    type: String,
    default: 'unknown'
  },
  userTask: {
    type: String,
    default: ''
  },
  rulesContent: {
    type: String,
    default: ''
  },
  finalPrompt: {
    type: String,
    default: ''
  }
});

const emit = defineEmits(['update:finalPrompt', 'update:userTask', 'update:rulesContent', 'action']);

const promptTemplates = {
  dev: { name: 'Dev', content: devTemplateContentFromFile },
  architect: { name: 'Architect', content: architectTemplateContentFromFile },
  findBug: { name: 'Find Bug', content: findBugTemplateContentFromFile },
  projectManager: { name: 'Project: Update Tasks', content: projectManagerTemplateContentFromFile },
};

const isPromptVisible = ref(true);
const selectedPromptTemplateKey = ref('dev');
const isLoadingFinalPrompt = ref(false);
let userTaskInputDebounceTimer = null;
const isPromptRulesModalVisible = ref(false);
const currentPromptRulesForModal = ref('');
const isFirstMount = ref(true);
const localUserTask = ref(props.userTask);
const DEFAULT_RULES = `no additional rules`;
const isProcessingNext = ref(false);
const copyButtonText = ref('Copy');

const charCount = computed(() => (props.finalPrompt || '').length);
const approximateTokens = computed(() => Math.round(charCount.value / 3).toString().replace(/\B(?=(\d{3})+(?!\d))/g, " "));
const charCountColorClass = computed(() => {
  const count = charCount.value;
  if (count < 1000000) return 'text-green-600';
  if (count <= 4000000) return 'text-yellow-500';
  return 'text-red-600';
});
const tooltipText = computed(() => {
  if (isLoadingFinalPrompt.value) return 'Calculating...';
  const tokens = Math.round(charCount.value / 3);
  return `Your text contains ${charCount.value} symbols which is roughly equivalent to ${tokens} tokens`;
});


// Функция для обновления предпросмотра, теперь вызывает Go
async function updateFinalPrompt() {
  isLoadingFinalPrompt.value = true;
  await new Promise(resolve => setTimeout(resolve, 50));
  
  try {
    const populatedPrompt = await AssembleFinalPrompt(
      promptTemplates[selectedPromptTemplateKey.value].content,
      localUserTask.value,
      props.rulesContent,
      props.fileListContext
    );
    emit('update:finalPrompt', populatedPrompt);
  } catch (error) {
    console.error("Error assembling prompt via backend:", error);
    LogErrorRuntime(`Error assembling prompt via backend: ${error.message || error}`);
    emit('update:finalPrompt', `Error: Failed to generate prompt. See console for details.`);
  } finally {
    isLoadingFinalPrompt.value = false;
  }
}

async function handleCopy() {
  copyButtonText.value = 'Copying...';
  isLoadingFinalPrompt.value = true;
  try {
    const freshPrompt = await AssembleFinalPrompt(
      promptTemplates[selectedPromptTemplateKey.value].content,
      localUserTask.value,
      props.rulesContent,
      props.fileListContext
    );
    emit('update:finalPrompt', freshPrompt);

    await navigator.clipboard.writeText(freshPrompt);

    copyButtonText.value = 'Copied!';
    setTimeout(() => {
      copyButtonText.value = 'Copy';
    }, 2000);
  } catch (err) {
    console.error('Failed to assemble or copy prompt: ', err);
    LogErrorRuntime(`Failed to assemble or copy prompt: ${err.message || err}`);
    emit('update:finalPrompt', `Error: Failed to generate and copy prompt. See console for details.`);
    copyButtonText.value = 'Failed!';
    setTimeout(() => {
      copyButtonText.value = 'Copy';
    }, 2000);
  } finally {
    isLoadingFinalPrompt.value = false;
  }
}

async function handleNext() {
  if (isProcessingNext.value) return;
  isProcessingNext.value = true;
  try {
    const freshPrompt = await AssembleFinalPrompt(
        promptTemplates[selectedPromptTemplateKey.value].content,
        localUserTask.value,
        props.rulesContent,
        props.fileListContext
    );

    emit('action', 'proceedToStep3', {
      role: selectedPromptTemplateKey.value,
      finalPrompt: freshPrompt
    });
  } catch (error) {
    console.error("Error assembling prompt on next:", error);
    LogErrorRuntime(`Error assembling prompt on next: ${error.message || error}`);
  } finally {
    isProcessingNext.value = false;
  }
}

// --- Наблюдатели (Watchers) ---
// Логика наблюдателей остается такой же, как в предыдущем предложении по исправлению.
// Это сохраняет производительность, так как `updateFinalPrompt` не вызывается при каждом нажатии клавиши.

watch(() => props.userTask, (newValue) => {
  if (newValue !== localUserTask.value) { localUserTask.value = newValue; }
});

watch(localUserTask, (currentValue) => {
  clearTimeout(userTaskInputDebounceTimer);
  userTaskInputDebounceTimer = setTimeout(() => {
    if (currentValue !== props.userTask) {
      emit('update:userTask', currentValue);
    }
  }, 300);
});

watch([() => props.rulesContent, () => props.fileListContext, selectedPromptTemplateKey], () => {
  updateFinalPrompt();
}, { immediate: true });

// --- Логика модального окна и монтирования (без изменений) ---
onMounted(async () => {
  try {
    localUserTask.value = props.userTask;
    if (isFirstMount.value) {
      const fetchedRules = await GetCustomPromptRules();
      if (!props.rulesContent) {
        emit('update:rulesContent', fetchedRules);
      }
      isFirstMount.value = false;
    }
  } catch (error) {
    console.error("Failed to load custom prompt rules:", error);
    LogErrorRuntime(`Failed to load custom prompt rules: ${error.message || error}`);
    if (isFirstMount.value && !props.rulesContent) {
      emit('update:rulesContent', DEFAULT_RULES);
    }
    isFirstMount.value = false;
  }
});

async function openPromptRulesModal() {
  try {
    currentPromptRulesForModal.value = await GetCustomPromptRules();
    isPromptRulesModalVisible.value = true;
  } catch (error) {
    console.error("Error fetching prompt rules for modal:", error);
    LogErrorRuntime(`Error fetching prompt rules for modal: ${error.message || error}`);
    currentPromptRulesForModal.value = props.rulesContent || DEFAULT_RULES;
    isPromptRulesModalVisible.value = true;
  }
}

async function handleSavePromptRules(newRules) {
  try {
    await SetCustomPromptRules(newRules);
    emit('update:rulesContent', newRules);
    isPromptRulesModalVisible.value = false;
    LogInfoRuntime('Custom prompt rules saved successfully.');
  } catch (error) {
    console.error("Error saving prompt rules:", error);
    LogErrorRuntime(`Error saving prompt rules: ${error.message || error}`);
  }
}

function handleCancelPromptRules() {
  isPromptRulesModalVisible.value = false;
}

defineExpose({});
</script>
