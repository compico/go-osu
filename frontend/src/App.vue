<script setup lang="ts">
import { computed, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import Navbar from './elements/Navbar.vue'
import GlobalPlayer from './elements/GlobalPlayer.vue'
import { useThemeStore } from './stores/theme'

const themeStore = useThemeStore()
const route = useRoute()

watch(
    () => themeStore.theme,
    (val) => document.documentElement.setAttribute('data-bs-theme', val),
    { immediate: true },
)

// страница может отключить нижнюю панель плеера через route meta: { hidePlayer: true }
const showPlayerBar = computed(() => !route.meta?.hidePlayer)
</script>

<template>
  <Navbar />
  <main class="main-content" :class="{ 'main-content--with-player': showPlayerBar }">
    <RouterView />
  </main>
  <GlobalPlayer v-show="showPlayerBar" />
</template>

<style scoped>
/* оставляем место под нижнюю панель плеера, чтобы контент не перекрывался */
.main-content--with-player {
  padding-bottom: 86px;
}
</style>