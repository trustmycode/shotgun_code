import { storeToRefs } from 'pinia';
import { EstimateTokens } from '../../wailsjs/go/main/App';
import { useLLMStore } from '../stores/llmStore';

const sharedTokenEstimateCache = new Map();
const LARGE_TEXT_HEURISTIC_THRESHOLD = 2_000_000;

export function useTokenEstimator() {
  const llmStore = useLLMStore();
  const { llmSettings } = storeToRefs(llmStore);

  function hashText(value) {
    let hash = 2166136261;
    for (let i = 0; i < value.length; i += 1) {
      hash ^= value.charCodeAt(i);
      hash += (hash << 1) + (hash << 4) + (hash << 7) + (hash << 8) + (hash << 24);
    }
    return (hash >>> 0).toString(16);
  }

  async function estimateTokensForText(text, overrides = {}) {
    const sourceText = text || '';
    if (!sourceText) {
      return { tokens: 0, method: 'heuristic', model: '' };
    }

    const provider = (overrides.provider || llmSettings.value?.activeProvider || '').toString();
    const model = (overrides.model || llmSettings.value?.model || '').toString();
    const key = `${provider}|${model}|${sourceText.length}|${hashText(sourceText)}`;

    const cached = sharedTokenEstimateCache.get(key);
    if (cached) {
      return cached;
    }

    if (sourceText.length >= LARGE_TEXT_HEURISTIC_THRESHOLD) {
      const fastFallback = {
        tokens: Math.max(1, Math.round(sourceText.length / 3)),
        method: 'heuristic',
        model,
      };
      sharedTokenEstimateCache.set(key, fastFallback);
      return fastFallback;
    }

    try {
      const estimate = await EstimateTokens(provider, model, sourceText);
      const result = {
        tokens: Number(estimate?.tokens || 0),
        method: estimate?.method || 'heuristic',
        model: estimate?.model || model,
      };
      sharedTokenEstimateCache.set(key, result);
      return result;
    } catch {
      const fallback = {
        tokens: Math.round(sourceText.length / 3),
        method: 'heuristic',
        model,
      };
      sharedTokenEstimateCache.set(key, fallback);
      return fallback;
    }
  }

  return {
    estimateTokensForText,
  };
}
