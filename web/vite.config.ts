import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: { license: { fileName: "THIRD_PARTY_NOTICES.md" } },
  resolve: {
    alias: {
      "@components": path.resolve(import.meta.dirname, "./src/components"),
      "@lib": path.resolve(import.meta.dirname, "./src/lib"),
      "@hooks": path.resolve(import.meta.dirname, "./src/hooks"),
    },
  },
  server: {
    host: "127.0.0.1",
    port: 5177,
    strictPort: true,
    proxy: { "/api": { target: "http://127.0.0.1:8787" } },
  },
});
