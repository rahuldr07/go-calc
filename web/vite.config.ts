import { svelte } from "@sveltejs/vite-plugin-svelte"
import { defineConfig } from "vite"

export default defineConfig({
  base: process.env.BASE_PATH ?? "/",
  plugins: [svelte()],
  worker: { format: "iife" },
  build: { target: "es2022", chunkSizeWarningLimit: 900 },
  test: { environment: "node", include: ["tests/**/*.test.ts"] },
})
