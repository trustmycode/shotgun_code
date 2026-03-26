import { defineStore } from 'pinia';

export const useProjectStore = defineStore('project', {
  state: () => ({
    projectRoot: '',
    fileTree: [],
    loadingError: '',
    useGitignore: true,
    useCustomIgnore: true,
    isFileTreeLoading: false,
    projectFilesChangedPendingReload: false,
    platform: 'unknown',
  }),
});
