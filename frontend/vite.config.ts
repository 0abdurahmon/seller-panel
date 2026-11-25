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
    build: {
      rolldownOptions: {
        output: {
          // Libraries change less often than our code: separate chunks stay cached between deploys.
          codeSplitting: {
            groups: [
              { name: 'react', test: /node_modules[\\/](react|react-dom|scheduler)[\\/]/, priority: 3 },
              { name: 'router', test: /node_modules[\\/]react-router[\\/]/, priority: 2 },
              { name: 'ui', test: /node_modules[\\/](@base-ui|@floating-ui)[\\/]/, priority: 2 },
            ],
          },
        },
      },
    },
    server: {
      proxy: {
        '/api': {
          target: backend,
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, ''),
          // The proxy holds the headers back until the first body bytes, which on the order events
          // stream is the ping 25 s later, so the page would not know it is connected. nginx does not.
          configure: (proxy) => {
            proxy.on('proxyRes', (proxyRes, _req, res) => {
              if (!proxyRes.headers['content-type']?.startsWith('text/event-stream')) return
              // This runs before the proxy copies the status and headers, so send them a tick later.
              process.nextTick(() => res.flushHeaders())
              // When the backend dies mid-stream the proxy leaves the browser's side open and silent.
              proxyRes.on('close', () => {
                if (!proxyRes.complete) res.destroy()
              })
            })
          },
        },
      },
    },
  }
})
