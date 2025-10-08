<template>
  <div class="p-4 h-full flex flex-col">
    <!-- API Key Management -->
    <div class="mb-4 p-3 border rounded-md bg-gray-50">
      <label for="api-key" class="block text-sm font-medium text-gray-700">Google AI API Key:</label>
      <div class="flex items-center space-x-2 mt-1">
        <input
          id="api-key"
          type="password"
          v-model="apiKey"
          class="flex-grow p-2 border border-gray-300 rounded-md shadow-sm text-sm"
          placeholder="Enter your API key"
        />
        <button @click="saveApiKeyHandler" class="px-4 py-2 bg-blue-500 text-white text-sm font-semibold rounded-md hover:bg-blue-600">Save</button>
      </div>
      <p v-if="apiKeyStatus" :class="['text-xs mt-1', apiKeyStatus.type === 'success' ? 'text-green-600' : 'text-red-600']">{{ apiKeyStatus.message }}</p>
    </div>

    <!-- Chat Area -->
    <div class="flex-grow flex flex-col border rounded-md overflow-hidden">
      <!-- Message History -->
      <div class="flex-grow p-4 overflow-y-auto bg-white" ref="chatHistoryRef">
        <div v-for="(msg, index) in messageHistory" :key="index" class="mb-4">
          <div :class="['p-3 rounded-lg max-w-xl relative group', msg.role === 'user' ? 'bg-blue-100 ml-auto' : 'bg-gray-100']">
            <button
              @click="copyMessage(msg.parts[0].text, $event)"
              class="absolute bottom-1 right-1 text-xs bg-gray-300 hover:bg-gray-400 text-gray-700 px-2 py-0.5 rounded opacity-0 group-hover:opacity-100 transition-opacity z-10"
            >
              Copy
            </button>
            <div class="prose prose-sm max-w-none" v-html="renderMarkdown(msg.parts[0].text)"></div>
          </div>
        </div>
        <!-- Streaming Response -->
        <div v-if="isStreaming" class="mb-4">
          <div class="p-3 rounded-lg bg-gray-100 max-w-xl">
            <div class="prose prose-sm max-w-none whitespace-pre-wrap font-mono">{{ currentStreamContent }}</div>
            <span class="blinking-cursor">|</span>
          </div>
        </div>
        <div v-if="streamError" class="text-red-500 text-sm p-2 bg-red-50 rounded">
          <strong>Error:</strong> {{ streamError }}
        </div>
      </div>

      <!-- Message Input -->
      <ChatInput
        :is-loading="isLoading"
        :can-finalize="canFinalize"
        :initial-temperature="props.temperature"
        @send-message="handleSendMessage"
        @finalize="finalizePrompt"
        @temperature-change="handleTemperatureChange"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick, computed } from 'vue';
import { SaveApiKey, LoadApiKey, CommunicateWithGoogleAI } from '../../../wailsjs/go/main/App';
import { EventsOn } from '../../../wailsjs/runtime/runtime';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import ChatInput from './ChatInput.vue';

const props = defineProps({
  initialPrompt: {
    type: String,
    default: ''
  },
  temperature: {
    type: Number,
    default: 0.1
  }
});

const emit = defineEmits(['action']);

const apiKey = ref('');
const apiKeyStatus = ref(null);
const messageHistory = ref([]);
const isLoading = ref(false);
const isStreaming = ref(false);
const currentStreamContent = ref('');
const streamError = ref('');
const chatHistoryRef = ref(null);
const lastUsedTemperature = ref(props.temperature);

let unlistenNewChunk;
let unlistenStreamEnd;
let unlistenStreamError;

let streamBuffer = '';
let updateInterval = null;

const canFinalize = computed(() => {
  const lastMessage = messageHistory.value[messageHistory.value.length - 1];
  return !isLoading.value && lastMessage && lastMessage.role === 'model';
});

onMounted(async () => {
  apiKey.value = await LoadApiKey();
  if (apiKey.value) {
    apiKeyStatus.value = { message: 'API Key loaded.', type: 'success' };
  }

  unlistenNewChunk = EventsOn('newChunk', (chunk) => {
    if (!isStreaming.value) {
      isStreaming.value = true;
      isLoading.value = true;

      updateInterval = setInterval(() => {
        if (streamBuffer.length > 0) {
          currentStreamContent.value += streamBuffer;
          streamBuffer = '';
          scrollToBottom();
        }
      }, 100);
    }
    streamBuffer += chunk;
  });

  unlistenStreamEnd = EventsOn('streamEnd', () => {
    clearInterval(updateInterval);
    updateInterval = null;

    if (streamBuffer.length > 0) {
      currentStreamContent.value += streamBuffer;
      streamBuffer = '';
    }

    if (currentStreamContent.value) {
      messageHistory.value.push({ role: 'model', parts: [{ text: currentStreamContent.value }] });
    }
    isStreaming.value = false;
    isLoading.value = false;
    currentStreamContent.value = '';
    streamError.value = '';
    scrollToBottom();
  });

  unlistenStreamError = EventsOn('streamError', (error) => {
    streamError.value = error;
    isStreaming.value = false;
    isLoading.value = false;
    scrollToBottom();
  });

  if (props.initialPrompt) {
    handleSendMessage({
      message: props.initialPrompt,
      temperature: props.temperature
    });
  }
});

onBeforeUnmount(() => {
  if (updateInterval) {
    clearInterval(updateInterval);
  }
  if (unlistenNewChunk) unlistenNewChunk();
  if (unlistenStreamEnd) unlistenStreamEnd();
  if (unlistenStreamError) unlistenStreamError();
});

async function saveApiKeyHandler() {
  try {
    await SaveApiKey(apiKey.value);
    apiKeyStatus.value = { message: 'API Key saved successfully!', type: 'success' };
  } catch (err) {
    apiKeyStatus.value = { message: `Failed to save API Key: ${err}`, type: 'error' };
  }
}

async function handleSendMessage({ message, temperature }) {
  const tempToUse = typeof temperature === 'number' ? temperature : lastUsedTemperature.value;
  if (!message.trim() || isLoading.value) return;
  if (!apiKey.value) {
    streamError.value = 'API Key is not set. Please enter and save your Google AI API key.';
    return;
  }

  const userMessage = { role: 'user', parts: [{ text: message }] };
  messageHistory.value.push(userMessage);
  isLoading.value = true;
  streamError.value = '';

  await nextTick();
  scrollToBottom();

  try {
    await CommunicateWithGoogleAI({
      history: messageHistory.value.slice(0, -1),
      message,
      temperature: tempToUse
    });
  } catch (err) {
    streamError.value = `Failed to send message: ${err}`;
    isLoading.value = false;
  }
}

function handleTemperatureChange(newTemperature) {
  lastUsedTemperature.value = newTemperature;
}

function finalizePrompt() {
  const lastMessage = messageHistory.value
    .slice()
    .reverse()
    .find((msg) => msg.role === 'model');

  if (!lastMessage) {
    streamError.value = 'No model response to finalize.';
    return;
  }

  emit('action', 'finalizePrompt', { finalPrompt: lastMessage.parts[0].text });
}

function renderMarkdown(text) {
  if (!text) return '';
  const rawHtml = marked.parse(text, { breaks: true, gfm: true });
  return DOMPurify.sanitize(rawHtml);
}

async function copyMessage(text, event) {
  if (!text) return;
  const button = event.currentTarget;
  if (!(button instanceof HTMLElement)) return;
  try {
    await navigator.clipboard.writeText(text);
    button.textContent = 'Copied!';
    setTimeout(() => {
      button.textContent = 'Copy';
    }, 2000);
  } catch (err) {
    console.error('Failed to copy message: ', err);
    button.textContent = 'Failed!';
    setTimeout(() => {
      button.textContent = 'Copy';
    }, 2000);
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (chatHistoryRef.value) {
      chatHistoryRef.value.scrollTop = chatHistoryRef.value.scrollHeight;
    }
  });
}
</script>

<style scoped>
/* Ensure prose styles don't get too opinionated for our layout */
:deep(.prose p) {
  margin-top: 0.5em;
  margin-bottom: 0.5em;
}

:deep(.prose pre) {
  margin-top: 0.75em;
  margin-bottom: 0.75em;
}

:deep(.prose ul),
:deep(.prose ol) {
  margin-top: 0.75em;
  margin-bottom: 0.75em;
}

.blinking-cursor {
  animation: blink 1s step-end infinite;
}
@keyframes blink {
  50% {
    opacity: 0;
  }
}
</style>
