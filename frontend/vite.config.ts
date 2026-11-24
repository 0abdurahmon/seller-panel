import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  // Where the dev server sends /api: the backend itself (http://localhost:8080) or a
  // running stack's nginx with its /api prefix (http://192.168.0.93:3000/api).
  // Set it in frontend/.env.local, which git ignores.
  const backend = loadEnv(mode, process.cwd(), '').VITE_API_URL ?? 'http://localhost:8080'

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      // Aliases come from "paths" in tsconfig.app.json, so they are defined once.
      tsconfigPaths: true,
    },
    server: {
      proxy: {
        '/api': {
          target: backend,
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, ''),
        },
      },
    },
  }
})
