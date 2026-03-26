<template>
  <div class="p-4 h-full flex flex-col relative">
    <CustomRulesModal
      :is-visible="isPromptRulesModalVisible"
      :initial-rules="currentPromptRulesForModal"
      title="Edit Custom Prompt Rules"
      ruleType="prompt"
      @save="handleSavePromptRules"
      @cancel="handleCancelPromptRules"
    />

    <!-- Top Bar: Instruction or Execute Button -->
    <div class="flex items-center justify-end mb-4 space-x-4 min-h-[32px]">
      <div class="flex items-center space-x-2">
        <button
          class="auto-context-button"
          :class="executeButtonClass"
          :disabled="!hasExecutePrerequisites"
          @click="handleExecutePrompt"
          title="Execute prompt with configured LLM"
        >
          <span>
            {{ isExecuting ? "Executing..." : "Execute Prompt" }}
          </span>
        </button>
        <button
          class="text-xs text-blue-600 hover:underline"
          type="button"
          @click="openLlmSettings"
        >
          Setup model
        </button>
        <div
          v-if="isGeneratingContext"
          class="text-xs text-amber-700 font-medium"
        >
          Updating context...
        </div>
      </div>
    </div>

    <div class="flex-grow flex flex-row space-x-4 overflow-hidden">
      <!-- Left Column: Task, Rules, Files -->
      <div
        :class="[
          leftColumnClass,
          'flex flex-col space-y-3 overflow-y-auto p-2 border border-gray-200 rounded-md bg-gray-50',
        ]"
      >
        <div>
          <label
            for="user-task-ai"
            class="block text-sm font-medium text-gray-700 mb-1"
            >Your task for AI:</label
          >
          <textarea
            id="user-task-ai"
            v-model="localUserTask"
            rows="15"
            class="w-full p-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 text-sm"
            placeholder="Describe what the AI should do..."
          ></textarea>
          <div class="mt-2 flex flex-wrap items-center gap-3">
            <div class="flex items-center space-x-2">
              <label class="text-sm font-medium text-gray-700"
                >Prompt role:</label
              >
              <select
                v-model="selectedPromptTemplateKey"
                class="p-1 border border-gray-300 rounded-md text-xs focus:ring-blue-500 focus:border-blue-500"
                title="Select prompt template"
              >
                <option
                  v-for="(template, key) in promptTemplates"
                  :key="key"
                  :value="key"
                >
                  {{ template.name }}
                </option>
              </select>
            </div>
            <div
              class="flex items-center space-x-2 text-xs text-gray-600"
              :title="tooltipText"
            >
              <span :class="['font-medium', charCountColorClass]"
                >~{{ approximateTokens }} tokens</span
              >
            </div>
            <button
              @click="copyFinalPromptToClipboard"
              :disabled="!finalPrompt || isLoadingFinalPrompt"
              class="px-3 py-1 bg-blue-500 text-white text-xs font-semibold rounded-md hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50 disabled:bg-gray-300"
            >
              {{ copyButtonText }}
            </button>
            <button
              type="button"
              @click="toggleFinalPromptVisibility"
              class="px-3 py-1 bg-gray-200 text-gray-700 text-xs font-semibold rounded-md hover:bg-gray-300 focus:outline-none focus:ring-2 focus:ring-gray-400 focus:ring-opacity-50"
            >
              {{ isFinalPromptCollapsed ? "Show" : "Hide" }}
            </button>
          </div>
        </div>

        <div>
          <label
            for="rules-content"
            class="block text-sm font-medium text-gray-700 mb-1 flex items-center"
          >
            Custom rules:
            <button
              @click="openPromptRulesModal"
              title="Edit custom prompt rules"
              class="ml-2 p-0.5 hover:bg-gray-200 rounded text-xs"
            >
              ⚙️
            </button>
          </label>
          <textarea
            id="rules-content"
            :value="rulesContent"
            @input="updateRulesContent"
            rows="8"
            class="w-full p-2 border border-gray-300 rounded-md shadow-sm bg-gray-100 text-sm font-mono"
            placeholder="Rules for AI..."
          ></textarea>
        </div>

        <div class="flex items-center gap-2">
          <label class="block text-sm font-medium text-gray-700"
            >Files to include:</label
          >
          <div
            v-if="isGeneratingContext"
            class="animate-spin rounded-full h-3 w-3 border-b-2 border-blue-500"
          ></div>
          <span v-if="isGeneratingContext" class="text-xs text-gray-500"
            >Refreshing...</span
          >
        </div>

        <LargeTextViewer
          :content="fileListContext"
          placeholder="File list from Step 1 (Prepare Context) will appear here..."
          :platform="platform"
          min-height="200px"
          :max-display-length="10000"
          copy-button-label="Copy All"
          :show-header="false"
        />
      </div>

      <!-- Right Column: Final Prompt -->
      <div
        class="w-1/2 flex flex-col overflow-y-auto p-2 border border-gray-200 rounded-md bg-white relative"
      >
        <div class="flex items-center justify-between gap-2 mb-2 flex-shrink-0">
          <div class="flex items-center gap-2 min-w-0">
            <h3 class="text-md font-medium text-gray-700 whitespace-nowrap">
              Prompt:
            </h3>
            <!-- Small Loading Indicator next to title instead of destroying content -->
            <div
              v-if="isLoadingFinalPrompt"
              class="animate-spin rounded-full h-3 w-3 border-b-2 border-blue-500 flex-shrink-0"
            ></div>
            <select
              v-model="selectedPromptTemplateKey"
              class="p-1 border border-gray-300 rounded-md text-xs focus:ring-blue-500 focus:border-blue-500 flex-shrink-0"
              title="Select prompt template"
            >
              <option
                v-for="(template, key) in promptTemplates"
                :key="key"
                :value="key"
              >
                {{ template.name }}
              </option>
            </select>
          </div>
          <button
            @click="copyFinalPromptToClipboard"
            :disabled="!finalPrompt || isLoadingFinalPrompt"
            class="px-3 py-1 bg-blue-500 text-white text-xs font-semibold rounded-md hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-opacity-50 disabled:bg-gray-300 whitespace-nowrap flex-shrink-0"
          >
            {{ copyButtonText }}
          </button>
        </div>
        <!-- 
           MODIFIED: 
           1. Removed v-if/v-else switching.
           2. Added transition classes to LargeTextViewer container.
           3. Added overlay class for loading state.
        -->
        <div class="flex flex-col flex-grow relative min-h-0">
          <div
            class="flex-grow transition-opacity duration-200 ease-in-out min-h-0"
            :class="{ 'opacity-50 grayscale': isLoadingFinalPrompt }"
          >
            <LargeTextViewer
              class="flex-grow h-full"
              :content="finalPrompt"
              :enable-token-estimation="false"
              label="Generated prompt preview"
              placeholder="The final prompt will be generated here..."
              :platform="platform"
              min-height="0px"
              max-height="100%"
              :max-display-length="15000"
              :show-copy-button="false"
            />
            <p class="text-xs text-gray-500 mt-1">
              Preview is truncated for performance. Use Copy All to grab the
              full text.
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- Response Modal -->
    <div
      v-if="isResponseModalVisible"
      class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
    >
      <div
        class="bg-white rounded-lg p-6 w-[90%] h-[90%] flex flex-col shadow-xl"
      >
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold">LLM Response</h3>
          <button
            @click="closeResponseModal"
            class="text-gray-500 hover:text-gray-700 text-2xl"
          >
            &times;
          </button>
        </div>
        <textarea
          readonly
          class="flex-grow p-4 border border-gray-300 rounded-md font-mono text-sm mb-4 resize-none bg-gray-50 focus:outline-none"
          :value="currentResponse"
        ></textarea>
        <div class="flex justify-end space-x-3">
          <button
            @click="copyResponse"
            class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
          >
            {{ copyResponseButtonText }}
          </button>
          <button
            @click="closeResponseModal"
            class="px-4 py-2 bg-gray-200 text-gray-800 rounded hover:bg-gray-300 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, onBeforeUnmount, computed } from "vue";
import { storeToRefs } from "pinia";
import {
  ClipboardSetText as WailsClipboardSetText,
  EventsOn,
  LogInfo as LogInfoRuntime,
  LogError as LogErrorRuntime,
} from "../../../wailsjs/runtime/runtime";
import {
  GetCustomPromptRules,
  SetCustomPromptRules,
  ExecuteLLMPrompt,
  ExecuteLLMPromptStream,
  CancelLLMPromptStream,
} from "../../../wailsjs/go/main/App";
import CustomRulesModal from "../CustomRulesModal.vue";
import LargeTextViewer from "../common/LargeTextViewer.vue";
import { useProjectStore } from "../../stores/projectStore";
import { useContextStore } from "../../stores/contextStore";
import { useLLMStore } from "../../stores/llmStore";
import { useTokenEstimator } from "../../composables/useTokenEstimator";

import devTemplateContentFromFile from "../../../../design/prompts/prompt_makeDiffGitFormat.md?raw";
import architectTemplateContentFromFile from "../../../../design/prompts/prompt_makePlan.md?raw";
import findBugTemplateContentFromFile from "../../../../design/prompts/prompt_analyzeBug.md?raw";
import projectManagerTemplateContentFromFile from "../../../../design/prompts/prompt_projectManager.md?raw";

const projectStore = useProjectStore();
const contextStore = useContextStore();
const llmStore = useLLMStore();

const { platform } = storeToRefs(projectStore);
const {
  shotgunPromptContext: fileListContext,
  isGeneratingContext,
  userTask,
  rulesContent,
  finalPrompt,
} = storeToRefs(contextStore);
const {
  hasActiveLlmKey,
  isLlmSettingsModalVisible,
  activeStreamRequestId,
  currentStreamingResponse,
} = storeToRefs(llmStore);
const { estimateTokensForText } = useTokenEstimator();

const promptTemplates = {
  architect: { name: "Architect", content: architectTemplateContentFromFile },
  findBug: { name: "Test", content: findBugTemplateContentFromFile },
  dev: { name: "Dev", content: devTemplateContentFromFile },
  // architect duplicate removed
  projectManager: {
    name: "Project: Update Tasks",
    content: projectManagerTemplateContentFromFile,
  },
};

const selectedPromptTemplateKey = ref("architect"); // Default template

const isLoadingFinalPrompt = ref(false);
const copyButtonText = ref("Copy All");

let finalPromptDebounceTimer = null;
let userTaskInputDebounceTimer = null;

// Modal state for prompt rules
const isPromptRulesModalVisible = ref(false);
const currentPromptRulesForModal = ref("");

// Response Modal State
const isResponseModalVisible = ref(false);
const currentResponse = computed({
  get: () => currentStreamingResponse.value || "",
  set: (value) => {
    currentStreamingResponse.value = value || "";
  },
});
const isExecuting = ref(false);
const copyResponseButtonText = ref("Copy Response");
const streamReceivedAnyChunk = ref(false);

const isFirstMount = ref(true);
const isFinalPromptCollapsed = ref(false);
const leftColumnClass = computed(() =>
  isFinalPromptCollapsed.value ? "w-full" : "w-1/2",
);

const localUserTask = ref(userTask.value);
const promptTokenCount = ref(0);
const promptTokenMethod = ref("heuristic");
let tokenEstimateDebounceTimer = null;
let unlistenStreamChunk = null;
let unlistenStreamEnd = null;
let unlistenStreamError = null;

const hasExecutePrerequisites = computed(() => {
  if (isGeneratingContext.value || isLoadingFinalPrompt.value) {
    return false;
  }
  if (!hasActiveLlmKey.value) {
    return false;
  }
  if (!localUserTask.value) {
    return false;
  }
  return localUserTask.value.trim().length > 0;
});

const executeButtonClass = computed(() => {
  if (!hasExecutePrerequisites.value) {
    return "auto-context-button--disabled";
  }
  if (isExecuting.value) {
    return "auto-context-button--in-progress";
  }
  return "auto-context-button--enabled";
});

const DEFAULT_RULES = `no additional rules`;
const promptCharCount = computed(() => (finalPrompt.value || "").length);

const approximateTokens = computed(() => {
  if (promptTokenCount.value > 0) {
    return promptTokenCount.value.toString().replace(/\B(?=(\d{3})+(?!\d))/g, " ");
  }
  const fallback = Math.round(promptCharCount.value / 3);
  return fallback.toString().replace(/\B(?=(\d{3})+(?!\d))/g, " ");
});

const tooltipText = computed(() => {
  const method = promptTokenMethod.value || "heuristic";
  return `Estimated with ${method}`;
});

const charCountColorClass = computed(() => {
  const count = promptCharCount.value;
  if (count < 1000000) return "text-green-600";
  if (count <= 4000000) return "text-yellow-500";
  return "text-red-600";
});

onMounted(async () => {
  bindStreamListeners();
  try {
    localUserTask.value = userTask.value;
    // Load rules from the backend only on the first mount
    if (isFirstMount.value) {
      const fetchedRules = await GetCustomPromptRules();
      if (!rulesContent.value) {
        rulesContent.value = fetchedRules;
      }
      isFirstMount.value = false;
    }
  } catch (error) {
    console.error("Failed to load custom prompt rules:", error);
    LogErrorRuntime(
      `Failed to load custom prompt rules: ${error.message || error}`,
    );
    if (isFirstMount.value && !rulesContent.value) {
      rulesContent.value = DEFAULT_RULES;
    }
    isFirstMount.value = false;
  }

  if (!finalPrompt.value && (fileListContext.value || userTask.value)) {
    debouncedUpdateFinalPrompt();
  }
  debouncedEstimatePromptTokens();
});

onBeforeUnmount(() => {
  if (finalPromptDebounceTimer) clearTimeout(finalPromptDebounceTimer);
  if (userTaskInputDebounceTimer) clearTimeout(userTaskInputDebounceTimer);
  if (tokenEstimateDebounceTimer) clearTimeout(tokenEstimateDebounceTimer);
  if (activeStreamRequestId.value) {
    CancelLLMPromptStream(activeStreamRequestId.value).catch(() => {});
    llmStore.clearStream();
  }
  if (unlistenStreamChunk) unlistenStreamChunk();
  if (unlistenStreamEnd) unlistenStreamEnd();
  if (unlistenStreamError) unlistenStreamError();
});

async function updateFinalPrompt() {
  isLoadingFinalPrompt.value = true;

  // MODIFIED: Removed the artificial delay (await new Promise...)
  // to make the update instant and smoother.
  // The debounce on input is enough to prevent performance issues.
  try {
    const currentTemplateContent =
      promptTemplates[selectedPromptTemplateKey.value].content;
    let populatedPrompt = currentTemplateContent;
    populatedPrompt = populatedPrompt.replace(
      "{TASK}",
      userTask.value || "No task provided by the user.",
    );
    populatedPrompt = populatedPrompt.replace("{RULES}", rulesContent.value);
    populatedPrompt = populatedPrompt.replace(
      "{FILE_STRUCTURE}",
      fileListContext.value || "No file structure context provided.",
    );

    // Insert current date in YYYY-MM-DD format
    const now = new Date();
    const yyyy = now.getFullYear();
    const mm = String(now.getMonth() + 1).padStart(2, "0");
    const dd = String(now.getDate()).padStart(2, "0");
    const currentDate = `${yyyy}-${mm}-${dd}`;
    populatedPrompt = populatedPrompt.replaceAll("{CURRENT_DATE}", currentDate);

    finalPrompt.value = populatedPrompt;
    debouncedEstimatePromptTokens();
  } finally {
    isLoadingFinalPrompt.value = false;
  }
}

function debouncedUpdateFinalPrompt() {
  // Set loading to true immediately to show "working" state via opacity opacity/spinner
  isLoadingFinalPrompt.value = true;

  clearTimeout(finalPromptDebounceTimer);
  finalPromptDebounceTimer = setTimeout(() => {
    updateFinalPrompt();
  }, 750);
}

watch(
  userTask,
  (newValue) => {
    if (newValue !== localUserTask.value) {
      localUserTask.value = newValue;
    }
  },
);

watch(localUserTask, (currentValue) => {
  clearTimeout(userTaskInputDebounceTimer);
  userTaskInputDebounceTimer = setTimeout(() => {
    if (currentValue !== userTask.value) {
      userTask.value = currentValue;
    }
  }, 300);
});

watch(
  [
    userTask,
    rulesContent,
    fileListContext,
    selectedPromptTemplateKey,
  ],
  () => {
    debouncedUpdateFinalPrompt();
    debouncedEstimatePromptTokens();
  },
  { deep: true },
);

watch(selectedPromptTemplateKey, () => {
  LogInfoRuntime(
    `Prompt template changed to: ${promptTemplates[selectedPromptTemplateKey.value].name}. Updating final prompt.`,
  );
  debouncedUpdateFinalPrompt();
  debouncedEstimatePromptTokens();
});

watch(
  finalPrompt,
  () => {
    debouncedEstimatePromptTokens();
  },
);

async function copyFinalPromptToClipboard() {
  if (!finalPrompt.value) return;

  // Use navigator.clipboard.writeText as primary (WailsClipboardSetText has UTF-8 encoding issues with box-drawing chars on darwin)
  try {
    await navigator.clipboard.writeText(finalPrompt.value);
    copyButtonText.value = "Copied!";
    resetCopyButtonLabel();
    return;
  } catch (err) {
    console.error("Failed to copy final prompt: ", err);
  }

  // Fallback to Wails clipboard API
  try {
    await WailsClipboardSetText(finalPrompt.value);
    copyButtonText.value = "Copied!";
  } catch (fallbackErr) {
    console.error(
      "Fallback copy attempt for final prompt also failed: ",
      fallbackErr,
    );
    copyButtonText.value = "Failed!";
  } finally {
    resetCopyButtonLabel();
  }
}

function resetCopyButtonLabel() {
  setTimeout(() => {
    copyButtonText.value = "Copy All";
  }, 2000);
}

async function openPromptRulesModal() {
  try {
    currentPromptRulesForModal.value = await GetCustomPromptRules();
    isPromptRulesModalVisible.value = true;
  } catch (error) {
    console.error("Error fetching prompt rules for modal:", error);
    LogErrorRuntime(
      `Error fetching prompt rules for modal: ${error.message || error}`,
    );
    currentPromptRulesForModal.value = rulesContent.value || DEFAULT_RULES;
    isPromptRulesModalVisible.value = true;
  }
}

async function handleSavePromptRules(newRules) {
  try {
    await SetCustomPromptRules(newRules);
    rulesContent.value = newRules;
    isPromptRulesModalVisible.value = false;
    LogInfoRuntime("Custom prompt rules saved successfully.");
  } catch (error) {
    console.error("Error saving prompt rules:", error);
    LogErrorRuntime(`Error saving prompt rules: ${error.message || error}`);
  }
}

function handleCancelPromptRules() {
  isPromptRulesModalVisible.value = false;
}

function updateRulesContent(event) {
  rulesContent.value = event?.target?.value ?? "";
}

function openLlmSettings() {
  isLlmSettingsModalVisible.value = true;
}

async function handleExecutePrompt() {
  if (!hasExecutePrerequisites.value) {
    return;
  }
  if (isExecuting.value) {
    return;
  }

  isExecuting.value = true;
  streamReceivedAnyChunk.value = false;
  llmStore.clearStream();
  isResponseModalVisible.value = true;
  LogInfoRuntime("Executing LLM prompt...");
  try {
    const requestId = await ExecuteLLMPromptStream(
      localUserTask.value,
      finalPrompt.value,
    );
    llmStore.startStream(requestId);
    LogInfoRuntime(`LLM stream started: ${requestId}`);
  } catch (err) {
    LogErrorRuntime(`Streaming execute failed, using fallback: ${err?.message || err}`);
    try {
      const result = await ExecuteLLMPrompt(
        localUserTask.value,
        finalPrompt.value,
      );
      if (result && result.response) {
        llmStore.finalizeStream(result.response);
        LogInfoRuntime("LLM Execution successful (non-stream fallback).");
      } else {
        throw new Error("Received empty response from backend.");
      }
    } catch (fallbackErr) {
      console.error("Error executing prompt:", fallbackErr);
      LogErrorRuntime(`Error executing prompt: ${fallbackErr.message || fallbackErr}`);
      currentResponse.value = `Error: ${fallbackErr.message || fallbackErr}`;
    } finally {
      isExecuting.value = false;
    }
    return;
  }
}

function bindStreamListeners() {
  unlistenStreamChunk = EventsOn("llmPromptStreamChunk", (payload) => {
    if (!payload || payload.requestId !== activeStreamRequestId.value) return;
    if (payload.chunk) {
      llmStore.appendStreamChunk(payload.chunk);
      streamReceivedAnyChunk.value = true;
    }
  });

  unlistenStreamEnd = EventsOn("llmPromptStreamEnd", (payload) => {
    if (!payload || payload.requestId !== activeStreamRequestId.value) return;
    if (!streamReceivedAnyChunk.value && payload.response) {
      llmStore.finalizeStream(payload.response);
    } else {
      llmStore.finalizeStream("");
    }
    LogInfoRuntime("LLM stream completed.");
    isExecuting.value = false;
  });

  unlistenStreamError = EventsOn("llmPromptStreamError", (payload) => {
    if (!payload || payload.requestId !== activeStreamRequestId.value) return;
    const partial = payload.partialResponse || "";
    const message = payload.message || "Unknown stream error";
    llmStore.failStream(partial, message);
    LogErrorRuntime(`LLM stream error: ${message}`);
    isExecuting.value = false;
  });
}

function debouncedEstimatePromptTokens() {
  if (tokenEstimateDebounceTimer) clearTimeout(tokenEstimateDebounceTimer);
  tokenEstimateDebounceTimer = setTimeout(() => {
    estimatePromptTokens();
  }, 300);
}

async function estimatePromptTokens() {
  const text = finalPrompt.value || "";
  if (!text) {
    promptTokenCount.value = 0;
    promptTokenMethod.value = "heuristic";
    return;
  }

  try {
    const estimate = await estimateTokensForText(text);
    const tokens = Number(estimate?.tokens || 0);
    const method = estimate?.method || "heuristic";
    promptTokenCount.value = tokens;
    promptTokenMethod.value = method;
  } catch (err) {
    const fallback = Math.round(text.length / 3);
    promptTokenCount.value = fallback;
    promptTokenMethod.value = "heuristic";
    LogErrorRuntime(`Failed to estimate prompt tokens on backend: ${err?.message || err}`);
  } finally {
    // no-op
  }
}

async function closeResponseModal() {
  if (isExecuting.value && activeStreamRequestId.value) {
    try {
      await CancelLLMPromptStream(activeStreamRequestId.value);
    } catch {
      // Ignore cancellation race where stream already ended.
    }
    isExecuting.value = false;
  }
  llmStore.clearStream();
  isResponseModalVisible.value = false;
}

async function copyResponse() {
  if (!currentResponse.value) return;
  try {
    await navigator.clipboard.writeText(currentResponse.value);
    copyResponseButtonText.value = "Copied!";
    setTimeout(() => {
      copyResponseButtonText.value = "Copy Response";
    }, 2000);
  } catch (err) {
    console.error("Failed to copy response:", err);
    copyResponseButtonText.value = "Failed!";
    setTimeout(() => {
      copyResponseButtonText.value = "Copy Response";
    }, 2000);
  }
}

function toggleFinalPromptVisibility() {
  isFinalPromptCollapsed.value = !isFinalPromptCollapsed.value;
}

defineExpose({});
</script>
