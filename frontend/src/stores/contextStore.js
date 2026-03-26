import { defineStore } from 'pinia';

export const useContextStore = defineStore('context', {
  state: () => ({
    shotgunPromptContext: '',
    isGeneratingContext: false,
    generationProgressData: { current: 0, total: 0 },
    userTask: '',
    rulesContent: '',
    finalPrompt: '',
    isAutoContextLoading: false,
  }),
});
