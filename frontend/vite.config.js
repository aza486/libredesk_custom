import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    port: 8000,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:9000',
      },
      '/logout': {
        target: 'http://127.0.0.1:9000',
      },
      '/uploads': {
        target: 'http://127.0.0.1:9000',
      },
      '/ws': {
        target: 'ws://127.0.0.1:9000',
        ws: true,
      },
    },
  },
  plugins: [
    vue(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
})
