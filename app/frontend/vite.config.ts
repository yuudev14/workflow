import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// The browser tests run without nginx. The dev server then sends the API paths
// to the API, the same as nginx does.
const api = process.env.API_URL ?? "http://localhost:8080";
const toApi = { target: api, rewrite: (p: string) => p.replace(/^\/[a-z]+-api/, "") };

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(import.meta.dirname, "src") },
  },
  // nginx proxies :9999 to the dev server on the host at :3000.
  server: {
    port: 3000,
    strictPort: true,
    proxy: {
      "/playbook-api": toApi,
      "/connector-api": toApi,
      "/auth-api": toApi,
      "/ws": { target: api, ws: true },
    },
  },
});
