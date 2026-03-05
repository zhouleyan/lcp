import path from "path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api/": {
        target: "http://localhost:8428",
        changeOrigin: true,
      },
      "/docs": {
        target: "http://localhost:8428",
        changeOrigin: true,
      },
      "/oidc": {
        target: "http://localhost:8428",
        changeOrigin: true,
      },
      "/.well-known": {
        target: "http://localhost:8428",
        changeOrigin: true,
      },
    },
  },
})
