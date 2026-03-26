import { describe, it, expect, beforeEach } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useProjectStore } from '../projectStore';
import { useContextStore } from '../contextStore';
import { useLLMStore } from '../llmStore';
import { useLogStore } from '../logStore';

describe('stores', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('project store has expected defaults', () => {
    const store = useProjectStore();
    expect(store.projectRoot).toBe('');
    expect(store.useGitignore).toBe(true);
    expect(store.useCustomIgnore).toBe(true);
    expect(Array.isArray(store.fileTree)).toBe(true);
  });

  it('context store updates state', () => {
    const store = useContextStore();
    store.userTask = 'fix auth';
    store.finalPrompt = 'prompt';
    expect(store.userTask).toBe('fix auth');
    expect(store.finalPrompt).toBe('prompt');
  });

  it('llm store keeps stream fields', () => {
    const store = useLLMStore();
    store.startStream('req-1');
    store.appendStreamChunk('hel');
    store.appendStreamChunk('lo');
    expect(store.activeStreamRequestId).toBe('req-1');
    expect(store.currentStreamingResponse).toBe('hello');
    store.finalizeStream('');
    expect(store.activeStreamRequestId).toBe('');
  });

  it('log store appends entries', () => {
    const store = useLogStore();
    store.add({ message: 'test', type: 'info', timestamp: '10:00:00' });
    expect(store.logMessages).toHaveLength(1);
    expect(store.logMessages[0].message).toBe('test');
  });
});
