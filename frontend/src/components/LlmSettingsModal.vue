<template>
  <div
    v-if="isVisible"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50"
    @click.self="handleCancel"
  >
    <div class="bg-white rounded-lg shadow-xl w-full max-w-xl p-6">
      <h2 class="text-xl font-semibold text-gray-800 mb-4">Настройки моделей</h2>

      <div class="mb-4">
        <label class="block text-sm font-medium text-gray-700 mb-1" for="provider-select">Поставщик</label>
        <select
          id="provider-select"
          v-model="localProvider"
          @change="handleProviderChange"
          class="w-full border border-gray-300 rounded-md p-2 text-sm"
          data-testid="provider-select"
        >
          <option v-for="option in providerOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </div>

      <div class="mb-4">
        <label class="block text-sm font-medium text-gray-700 mb-1" for="api-key-input">Ключ доступа</label>
        <input
          id="api-key-input"
          type="password"
          v-model="localApiKeys[localProvider]"
          :placeholder="keyPresence[localProvider] ? 'Ключ уже сохранён; оставьте поле пустым, чтобы не менять его' : 'Вставьте ключ выбранного поставщика'"
          class="w-full border border-gray-300 rounded-md p-2 text-sm"
          data-testid="api-key-input"
        />
        <p class="text-xs text-gray-500 mt-1">Ключ хранится только в локальном файле настроек с ограниченными правами доступа.</p>
      </div>

      <div class="mb-4">
        <label class="block text-sm font-medium text-gray-700 mb-1" for="base-url-input">
          Пользовательский базовый адрес (необязательно)
        </label>
        <input
          id="base-url-input"
          type="text"
          v-model="localBaseUrl"
          placeholder="https://example.com/v1"
          class="w-full border border-gray-300 rounded-md p-2 text-sm"
        />
        <p class="text-xs text-amber-700 mt-1">Ключ будет отправляться на этот адрес. Разрешены только защищённые адреса HTTPS.</p>
      </div>

      <div class="mb-4">
        <div class="flex justify-between items-center">
          <label class="block text-sm font-medium text-gray-700" for="model-input">Модель</label>
          <button
            class="text-xs text-blue-600 hover:underline disabled:text-gray-400"
            :disabled="isLoadingModels"
            @click="fetchModels"
          >
            {{ isLoadingModels ? 'Загрузка…' : 'Обновить список' }}
          </button>
        </div>
        <input
          id="model-input"
          type="text"
          v-model="localModel"
          placeholder="Введите название модели"
          class="w-full border border-gray-300 rounded-md p-2 text-sm"
          :disabled="isLoadingModels"
          data-testid="model-select"
        />
        <div
          v-if="filteredModelSuggestions.length"
          class="mt-2 border border-gray-200 rounded-md max-h-40 overflow-y-auto divide-y divide-gray-100"
        >
          <button
            v-for="model in filteredModelSuggestions"
            :key="model"
            type="button"
            class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50"
            @click="selectSuggestion(model)"
          >
            {{ model }}
          </button>
        </div>
        <p class="text-xs text-gray-500 mt-1">Начните ввод, чтобы отфильтровать подсказки, либо укажите своё значение.</p>
      </div>

      <p v-if="errorMessage" class="text-red-600 text-sm mb-4 whitespace-pre-wrap">{{ errorMessage }}</p>

      <div class="flex justify-end space-x-2">
        <button
          class="px-4 py-2 rounded-md border border-gray-300 text-gray-700 text-sm"
          @click="handleCancel"
          data-testid="cancel-btn"
        >
          Отмена
        </button>
        <button
          class="px-4 py-2 rounded-md text-white text-sm"
          :class="isSaving ? 'bg-blue-400' : 'bg-blue-600 hover:bg-blue-700'"
          :disabled="isSaving"
          @click="handleSave"
          data-testid="save-btn"
        >
          {{ isSaving ? 'Сохранение…' : 'Сохранить' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue';
import {
  ListLlmModels,
  SetLlmApiKey,
  SetLlmBaseURL,
  SetLlmModel,
  SetLlmProvider,
} from '../../wailsjs/go/main/App';

const props = defineProps({
  isVisible: {
    type: Boolean,
    default: false,
  },
  initialSettings: {
    type: Object,
    default: () => ({}),
  },
});

const emit = defineEmits(['close', 'saved']);

const providerOptions = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'openrouter', label: 'OpenRouter' },
  { value: 'gemini', label: 'Google Gemini' },
];

const providerDefaultModels = {
  openai: 'gpt-5',
  openrouter: 'openai/gpt-5',
  gemini: 'gemini-2.5-pro',
};

const localProvider = ref('openai');
const localModel = ref('');
const localBaseUrl = ref('');
const localApiKeys = reactive({
  openai: '',
  openrouter: '',
  gemini: '',
});
const keyPresence = reactive({
  openai: false,
  openrouter: false,
  gemini: false,
});

const modelOptions = ref([]);
const isLoadingModels = ref(false);
const isSaving = ref(false);
const errorMessage = ref('');

const hasActiveKey = computed(() => Boolean(localApiKeys[localProvider.value] || keyPresence[localProvider.value]));
const filteredModelSuggestions = computed(() => {
  const query = (localModel.value || '').trim().toLowerCase();
  return modelOptions.value.filter((option) => {
    const normalizedOption = (option || '').toLowerCase();
    if (!normalizedOption) {
      return false;
    }
    if (query && normalizedOption === query) {
      return false;
    }
    if (!query) {
      return true;
    }
    return normalizedOption.includes(query);
  });
});

function syncStateFromProps() {
  const settings = props.initialSettings || {};
  localProvider.value = settings.activeProvider || 'openai';
  localModel.value = settings.model || providerDefaultModels[localProvider.value] || '';
  localBaseUrl.value = settings.baseURL || '';
  localApiKeys.openai = '';
  localApiKeys.openrouter = '';
  localApiKeys.gemini = '';
  keyPresence.openai = Boolean(settings.hasOpenAIKey);
  keyPresence.openrouter = Boolean(settings.hasOpenRouterKey);
  keyPresence.gemini = Boolean(settings.hasGeminiKey);
  modelOptions.value = [];
  errorMessage.value = '';
}

watch(
  () => props.initialSettings,
  () => {
    syncStateFromProps();
  },
  { immediate: true }
);

watch(
  () => props.isVisible,
  (visible) => {
    if (visible) {
      syncStateFromProps();
      fetchModels();
    } else {
      modelOptions.value = [];
      errorMessage.value = '';
    }
  }
);

function handleProviderChange() {
  errorMessage.value = '';
  modelOptions.value = [];
  localModel.value = providerDefaultModels[localProvider.value] || '';
  fetchModels();
}

async function fetchModels() {
  if (!localProvider.value) {
    modelOptions.value = [];
    return;
  }
  isLoadingModels.value = true;
  errorMessage.value = '';
  try {
    const response = await ListLlmModels(localProvider.value);
    const names = Array.isArray(response) ? response.map((m) => m.name || m.Name || '').filter(Boolean) : [];
    modelOptions.value = names;
    if (!localModel.value && names.length) {
      localModel.value = names[0];
    }
  } catch (err) {
    errorMessage.value = `Не удалось загрузить список моделей: ${err?.message || err}`;
  } finally {
    isLoadingModels.value = false;
  }
}

async function handleSave() {
  if (!hasActiveKey.value) {
    errorMessage.value = 'Укажите ключ доступа.';
    return;
  }
  if (!localModel.value) {
    localModel.value = providerDefaultModels[localProvider.value] || '';
  }

  isSaving.value = true;
  errorMessage.value = '';
  try {
    if (localApiKeys[localProvider.value]) {
      await SetLlmApiKey(localProvider.value, localApiKeys[localProvider.value]);
    }
    await SetLlmBaseURL(localBaseUrl.value || '');
    await SetLlmProvider(localProvider.value);
    await SetLlmModel(localProvider.value, localModel.value);
    emit('saved');
    emit('close');
  } catch (err) {
    errorMessage.value = err?.message || `${err}`;
  } finally {
    isSaving.value = false;
  }
}

function handleCancel() {
  emit('close');
}

function selectSuggestion(value) {
  localModel.value = value;
}
</script>
