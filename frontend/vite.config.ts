import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    // 直接输出到后端 dist
    outDir: path.resolve(__dirname, "../backend/internal/gui/dist"),
    emptyOutDir: true, // 构建前自动清空目标目录
  },
})
