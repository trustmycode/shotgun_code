import { defineStore } from 'pinia';

const MAX_LOG_MESSAGES = 500;

export const useLogStore = defineStore('logs', {
  state: () => ({
    logMessages: [],
  }),
  actions: {
    add(entry) {
      this.logMessages.push(entry);
      if (this.logMessages.length > MAX_LOG_MESSAGES) {
        this.logMessages.splice(0, this.logMessages.length - MAX_LOG_MESSAGES);
      }
    },
    clear() {
      this.logMessages = [];
    },
  },
});
