import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

export default {
  preprocess: vitePreprocess(),
  onwarn(warning, handler) {
    // Third-party components are not ours to fix.
    if (warning.filename?.includes("node_modules")) return;
    handler(warning);
  },
};
