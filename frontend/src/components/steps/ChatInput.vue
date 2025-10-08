<template>
  <div class="p-4 border-t bg-gray-50">
    <div class="flex items-start space-x-3">
      <textarea
        v-model="currentUserMessage"
        @keydown.enter.prevent="handleSendMessage"
        :disabled="props.isLoading"
        rows="2"
        class="flex-grow p-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500 text-sm"
        placeholder="Type your message..."
      ></textarea>
      <button
        @click="handleSendMessage"
        :disabled="props.isLoading || !currentUserMessage.trim()"
        class="px-6 py-2 bg-blue-600 text-white font-semibold rounded-md hover:bg-blue-700 disabled:bg-gray-400"
      >
        <span v-if="props.isLoading">...</span>
        <span v-else>Send</span>
      </button>
    </div>
    <div class="mt-2 flex items-center justify-between">
      <div class="flex items-center">
        <label for="temperature" class="text-sm font-medium text-gray-700 mr-2">Temperature: {{ temperature }}</label>
        <input
          type="range"
          id="temperature"
          min="0"
          max="1"
          step="0.1"
          v-model.number="temperature"
          @input="handleTemperatureChange"
          class="w-48"
        >
      </div>
      <button
        @click="handleFinalize"
        :disabled="!props.canFinalize"
        class="px-4 py-2 bg-green-600 text-white text-sm font-semibold rounded-md hover:bg-green-700 disabled:bg-gray-400"
        title="Use the last AI response as the final prompt for Step 4"
      >
        Use Last Response & Proceed
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue';

const props = defineProps({
  isLoading: {
    type: Boolean,
    default: false
  },
  canFinalize: {
    type: Boolean,
    default: false
  },
  initialTemperature: {
    type: Number,
    default: 0.1
  }
});

const emit = defineEmits(['sendMessage', 'finalize', 'temperatureChange']);

const currentUserMessage = ref('');
const temperature = ref(props.initialTemperature);

watch(() => props.initialTemperature, (newVal) => {
  temperature.value = newVal;
});

function handleSendMessage() {
  if (!currentUserMessage.value.trim() || props.isLoading) return;
  emit('sendMessage', { message: currentUserMessage.value, temperature: temperature.value });
  currentUserMessage.value = '';
}

function handleFinalize() {
  emit('finalize');
}

function handleTemperatureChange() {
  emit('temperatureChange', temperature.value);
}
</script>
