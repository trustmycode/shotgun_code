<template>
  <main class="flex-1 p-0 overflow-y-auto bg-white relative">
    <Step1CopyStructure
      v-if="currentStep === 1"
      @auto-context="emit('auto-context')"
    />
    <Step2ComposePrompt 
      v-if="currentStep === 2" 
      ref="step2Ref" 
    />
    <Step3ExecutePrompt v-if="currentStep === 3" ref="step3Ref" />
  </main>
</template>

<script setup>
import { ref } from 'vue';
import Step1CopyStructure from './steps/Step1PrepareContext.vue';
import Step2ComposePrompt from './steps/Step2ComposePrompt.vue';
import Step3ExecutePrompt from './steps/Step3ExecutePrompt.vue';

defineProps({
  currentStep: { type: Number, required: true },
});

const emit = defineEmits(['auto-context']);

const step2Ref = ref(null);
const step3Ref = ref(null);

const updateStep2DiffOutput = (output) => {
  if (step2Ref.value && step2Ref.value.setDiffOutput) {
    step2Ref.value.setDiffOutput(output);
  }
};

const updateStep2ShotgunContext = (context) => {
  if (step2Ref.value && step2Ref.value.setShotgunContext) {
    step2Ref.value.setShotgunContext(context);
  }
};

const addLogToStep3Console = (message, type) => {
  if (step3Ref.value && step3Ref.value.addLog) {
    step3Ref.value.addLog(message, type);
  }
};

defineExpose({ updateStep2DiffOutput, addLogToStep3Console, updateStep2ShotgunContext });
</script> 
