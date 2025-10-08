<template>
  <main class="flex-1 p-0 overflow-y-auto bg-white relative">
    <Step1CopyStructure v-if="currentStep === 1" @action="handleAction" :generated-context="shotgunPromptContext" :is-loading-context="props.isGeneratingContext" :project-root="props.projectRoot" :generation-progress="props.generationProgress" :platform="props.platform" />
    <Step2ComposePrompt v-if="currentStep === 2" @action="handleAction" ref="step2Ref" :file-list-context="props.shotgunPromptContext" @update:finalPrompt="(val) => emit('update-composed-prompt', val)" :platform="props.platform" :user-task="props.userTask" :rules-content="props.rulesContent" :final-prompt="props.finalPrompt" @update:userTask="(val) => emit('update:userTask', val)" @update:rulesContent="(val) => emit('update:rulesContent', val)" />
    <Step3Chat v-if="currentStep === 3" @action="handleAction" :initial-prompt="props.finalPrompt" :temperature="props.chatTemperature" />
    <Step4ApplyPatch v-if="currentStep === 4" @action="handleAction" :platform="props.platform" :final-prompt="props.finalPrompt" :project-root="props.projectRoot" />
  </main>
</template>

<script setup>
import { defineProps, defineEmits, ref } from 'vue';
import Step1CopyStructure from './steps/Step1PrepareContext.vue';
import Step2ComposePrompt from './steps/Step2ComposePrompt.vue';
import Step3Chat from './steps/Step3Chat.vue';
import Step4ApplyPatch from './steps/Step4ApplyPatch.vue';

const props = defineProps({
  currentStep: { type: Number, required: true },
  shotgunPromptContext: { type: String, default: '' },
  isGeneratingContext: { type: Boolean, default: false },
  projectRoot: { type: String, default: '' },
  generationProgress: { type: Object, default: () => ({ current: 0, total: 0 }) },
  platform: { type: String, default: 'unknown' },
  userTask: { type: String, default: '' },
  rulesContent: { type: String, default: '' },
  finalPrompt: { type: String, default: '' },
  chatTemperature: { type: Number, default: 0.1 }
});

const emit = defineEmits(['stepAction', 'update-composed-prompt', 'update:userTask', 'update:rulesContent']);

const step2Ref = ref(null);

function handleAction(actionName, payload) {
  emit('stepAction', actionName, payload);
}

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

defineExpose({ updateStep2DiffOutput, updateStep2ShotgunContext });
</script>
