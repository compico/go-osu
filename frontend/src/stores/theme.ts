import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

type Theme = 'light' | 'dark'

export const useThemeStore = defineStore(
  'theme',
  () => {
    const theme = ref<Theme>('light')

    const isDark = computed(() => theme.value === 'dark')

    function toggle() {
      theme.value = isDark.value ? 'light' : 'dark'
    }

    function apply() {
      document.documentElement.setAttribute('data-bs-theme', theme.value)
    }

    return { theme, isDark, toggle, apply }
  },
  { persist: true },
)
