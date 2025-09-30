<template>
  <div class="p-4 h-full flex flex-col">
    <h2 class="text-xl font-semibold text-gray-800 mb-2">Step 3: Chat with AI</h2>
    <p class="text-gray-600 mb-4 text-sm">
      Interact with the AI to refine your request. The initial prompt from Step 2 is automatically sent as the first message.
    </p>
    
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
          <div :class="['p-3 rounded-lg max-w-xl', msg.role === 'user' ? 'bg-blue-100 ml-auto' : 'bg-gray-100']">
            <p class="text-sm text-gray-800 whitespace-pre-wrap">{{ msg.parts[0].text }}</p>
          </div>
        </div>
        <!-- Streaming Response -->
        <div v-if="isStreaming" class="mb-4">
          <div class="p-3 rounded-lg bg-gray-100 max-w-xl">
            <p class="text-sm text-gray-800 whitespace-pre-wrap">{{ currentStreamContent }}<span class="blinking-cursor">|</span></p>
          </div>
        </div>
        <div v-if="streamError" class="text-red-500 text-sm p-2 bg-red-50 rounded">
          <strong>Error:</strong> {{ streamError }}
        </div>
      </div>

      <!-- Message Input -->
      <div class="p-4 border-t bg-gray-50">
        <div class="flex items-start space-x-3">
          <textarea
            v-model="currentUserMessage"
            @keydown.enter.prevent="sendMessage"
            :disabled="isLoading"
            rows="2"
            class="flex-grow p-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 text-sm"
            placeholder="Type your message..."
          ></textarea>
          <button @click="sendMessage" :disabled="isLoading || !currentUserMessage.trim()" class="px-6 py-2 bg-blue-600 text-white font-semibold rounded-md hover:bg-blue-700 disabled:bg-gray-400">
            <span v-if="isLoading">...</span>
            <span v-else>Send</span>
          </button>
        </div>
        <div class="mt-2 flex items-center justify-between">
            <div class="flex items-center">
                <label for="temperature" class="text-sm font-medium text-gray-700 mr-2">Temperature: {{ temperature }}</label>
                <input type="range" id="temperature" min="0" max="1" step="0.1" v-model.number="temperature" class="w-48">
            </div>
            <button 
                @click="finalizePrompt" 
                :disabled="!canFinalize"
                class="px-4 py-2 bg-green-600 text-white text-sm font-semibold rounded-md hover:bg-green-700 disabled:bg-gray-400"
                title="Use the last AI response as the final prompt for Step 4"
            >
                Use Last Response & Proceed
            </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick, computed } from 'vue';
import { SaveApiKey, LoadApiKey, CommunicateWithGoogleAI } from '../../../wailsjs/go/main/App';
import { EventsOn } from '../../../wailsjs/runtime/runtime';

const props = defineProps({
  initialPrompt: {
    type: String,
    default: ''
  }
});

const emit = defineEmits(['action']);

const apiKey = ref('');
const apiKeyStatus = ref(null);
const messageHistory = ref([]);
const currentUserMessage = ref('');
const isLoading = ref(false);
const isStreaming = ref(false);
const currentStreamContent = ref('');
const streamError = ref('');
const temperature = ref(0.1);
const chatHistoryRef = ref(null);

let unlistenNewChunk;
let unlistenStreamEnd;
let unlistenStreamError;

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
    isStreaming.value = true;
    isLoading.value = true;
    currentStreamContent.value += chunk;
    scrollToBottom();
  });

  unlistenStreamEnd = EventsOn('streamEnd', () => {
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
    currentUserMessage.value = props.initialPrompt;
    sendMessage();
  }
});

onBeforeUnmount(() => {
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

async function sendMessage() {
  if (!currentUserMessage.value.trim() || isLoading.value) return;
  if (!apiKey.value) {
    streamError.value = 'API Key is not set. Please enter and save your Google AI API key.';
    return;
  }

  const userMessage = { role: 'user', parts: [{ text: currentUserMessage.value }] };
  messageHistory.value.push(userMessage);
  const messageToSend = currentUserMessage.value;
  currentUserMessage.value = '';
  isLoading.value = true;
  streamError.value = '';

  await nextTick();
  scrollToBottom();

  try {
    await CommunicateWithGoogleAI({
      history: messageHistory.value.slice(0, -1),
      message: messageToSend,
      temperature: temperature.value
    });
  } catch (err) {
    streamError.value = `Failed to send message: ${err}`;
    isLoading.value = false;
  }
}

function finalizePrompt() {
  if (!canFinalize.value) return;
  const lastMessage = messageHistory.value[messageHistory.value.length - 1];
  emit('action', 'finalizePrompt', { finalPrompt: lastMessage.parts[0].text });
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
.blinking-cursor {
  animation: blink 1s step-end infinite;
}
@keyframes blink {
  50% {
    opacity: 0;
  }
}
</style>
