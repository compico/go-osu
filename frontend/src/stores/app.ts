import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore(
  'app',
  () => {
    const initialized = ref(false)

    function init() {
      initialized.value = true
    }

    return { initialized, init }
  },
  { persist: true },
)
