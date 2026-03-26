import { defineStore } from 'pinia';

export const useLLMStore = defineStore('llm', {
  state: () => ({
    hasActiveLlmKey: false,
    llmSettings: {},
    isLlmSettingsModalVisible: false,
    activeStreamRequestId: '',
    currentStreamingResponse: '',
  }),
  actions: {
    startStream(requestId) {
      this.activeStreamRequestId = requestId || '';
      this.currentStreamingResponse = '';
    },
    appendStreamChunk(chunk) {
      if (!chunk) {
        return;
      }
      this.currentStreamingResponse += chunk;
    },
    finalizeStream(response) {
      if (response && !this.currentStreamingResponse) {
        this.currentStreamingResponse = response;
      }
      this.activeStreamRequestId = '';
    },
    failStream(partialResponse, message) {
      const partial = partialResponse || '';
      const errorMessage = message || 'Unknown stream error';
      this.currentStreamingResponse = partial
        ? `${partial}\n\nError: ${errorMessage}`
        : `Error: ${errorMessage}`;
      this.activeStreamRequestId = '';
    },
    clearStream() {
      this.activeStreamRequestId = '';
      this.currentStreamingResponse = '';
    },
  },
});
