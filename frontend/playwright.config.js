import { defineConfig, devices } from "@playwright/test";

const port = 4174;

// UI tests: the production build of the frontend, served by `vite preview`,
// running against a fake backend (tests/fake-backend.js).
//
// WebKit is the engine that matters (the app runs on WebKitGTK / WKWebView);
// Chromium is close to WebView2 on Windows. Playwright's WebKit needs Ubuntu
// libraries, so on other Linux distributions use `npm run test:webkit:docker`.
export default defineConfig({
  testDir: "tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    viewport: { width: 1200, height: 820 }, // the app's default window size
    trace: "retain-on-failure",
  },
  projects: [
    { name: "webkit", use: { ...devices["Desktop Safari"], viewport: { width: 1200, height: 820 } } },
    { name: "chromium", use: { ...devices["Desktop Chrome"], viewport: { width: 1200, height: 820 } } },
  ],
  webServer: {
    command: `npm run build && npx vite preview --host 127.0.0.1 --port ${port} --strictPort`,
    url: `http://127.0.0.1:${port}`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
