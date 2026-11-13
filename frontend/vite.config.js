import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

// `wails dev` starts this dev server (frontend:dev:serverUrl "auto") and
// proxies it, so the app gets hot reload. `wails build` embeds dist/.
export default defineConfig({
  plugins: [svelte()],
  build: {
    // The webviews Wombat runs in (WebKitGTK, WKWebView, WebView2) are all
    // evergreen, so no transpiling or polyfills are needed.
    target: "es2020",
    outDir: "dist",
    emptyOutDir: true,
    chunkSizeWarningLimit: 4096,
  },
  server: {
    strictPort: true,
  },
});
