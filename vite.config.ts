import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import path from 'node:path'

export default defineConfig({
  root: "frontend",

  plugins: [vue()],

  resolve: {
    alias: {
      '@': path.resolve(__dirname, './frontend/src'),
    },
  },
  build: {
    outDir: "dist",

    manifest: true,

    emptyOutDir: true,
    rolldownOptions: {
      input: "frontend/src/main.ts",
    },
  },
});
