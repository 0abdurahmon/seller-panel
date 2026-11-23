import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

const backend = process.env.VITE_API_URL ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: backend,
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
