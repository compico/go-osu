<script setup lang="ts">

import { watch } from 'vue'
import { RouterView } from 'vue-router'
import Navbar from './elements/Navbar.vue'
import { useThemeStore } from './stores/theme'

const themeStore = useThemeStore()

watch(
  () => themeStore.theme,
  (val) => document.documentElement.setAttribute('data-bs-theme', val),
  { immediate: true },
)
</script>

<template>
  <Navbar />
  <main class="main-content">
    <RouterView />
  </main>
</template>
