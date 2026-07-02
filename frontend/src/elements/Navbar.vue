<script setup lang="ts">
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()

function toggleTheme() {
  themeStore.toggle()
  themeStore.apply()
}
</script>

<template>
  <nav class="navbar fixed-top bg-body border-bottom shadow-sm px-3">
    <div class="container-fluid">

      <!-- Brand -->
      <span class="navbar-brand fw-semibold mb-0 d-flex align-items-center gap-2">
        Go<i class="bi bi-play-circle-fill"></i>su
      </span>

      <!-- Right side -->
      <div class="d-flex align-items-center gap-1">
        <button
          class="btn btn-link nav-link px-2 py-1"
          :title="themeStore.isDark ? 'Switch to light mode' : 'Switch to dark mode'"
          @click="toggleTheme"
        >
          <i
            class="bi fs-5"
            :class="themeStore.isDark ? 'bi-sun' : 'bi-moon'"
          />
        </button>
      </div>

    </div>
  </nav>
</template>
