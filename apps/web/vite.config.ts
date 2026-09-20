import path from 'node:path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@ai-stock/types': path.resolve(__dirname, '../../packages/types/src/index.ts'),
      '@ai-stock/api-client': path.resolve(__dirname, '../../packages/api-client/src/index.ts'),
      '@ai-stock/business': path.resolve(__dirname, '../../packages/business/src/index.ts'),
    },
  },
  server: {
    port: 5273,
    proxy: {
      '/api': 'http://127.0.0.1:18080',
    },
  },
})
