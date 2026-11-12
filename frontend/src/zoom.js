// UI zoom.
//
// The whole UI is scaled with CSS `zoom` on <html>. WebKitGTK and WebView2
// both lay out, hit-test and report getBoundingClientRect()/mouse coordinates
// consistently in zoomed space, so no per-component compensation is needed.
//
// Two things to keep in mind when writing CSS:
//  - Viewport units are zoomed too: use percentages, or divide by the zoom
//    (`calc(90vh / var(--zoom))`), otherwise things overflow when zoomed in.
//  - Monaco (0.21) does not understand CSS zoom. Its host element cancels the
//    zoom (class `unzoomed`) and the editor's fontSize is scaled instead; see
//    editorFontSize().

import { writable, get } from "svelte/store";

const STORAGE_KEY = "wombat_zoom_level";
const STEP = 0.1;
const MIN = 0.5;
const MAX = 3.0;

const clamp = (z) => Math.min(MAX, Math.max(MIN, Math.round(z * 100) / 100));

function load() {
  try {
    const saved = parseFloat(localStorage.getItem(STORAGE_KEY));
    if (Number.isFinite(saved)) return clamp(saved);
  } catch (e) {
    // storage unavailable
  }
  // First start: pick a sensible default for large screens.
  const { width, height } = window.screen;
  if (width >= 3840 || height >= 2160) return 1.5;
  if (width >= 2560 || height >= 1440) return 1.25;
  return 1.0;
}

export const zoom = writable(load());

zoom.subscribe((z) => {
  const root = document.documentElement;
  root.style.zoom = String(z);
  root.style.setProperty("--zoom", String(z));
  try {
    localStorage.setItem(STORAGE_KEY, String(z));
  } catch (e) {
    // storage unavailable
  }
});

export const zoomIn = () => zoom.update((z) => clamp(z + STEP));
export const zoomOut = () => zoom.update((z) => clamp(z - STEP));
export const resetZoom = () => zoom.set(1.0);

// Font size for a Monaco editor hosted in an `.unzoomed` element.
export const editorFontSize = (z = get(zoom)) => Math.round(14 * z);

document.addEventListener("keydown", (event) => {
  if (!event.ctrlKey && !event.metaKey) return;
  if (event.key === "+" || event.key === "=") {
    event.preventDefault();
    zoomIn();
  } else if (event.key === "-") {
    event.preventDefault();
    zoomOut();
  } else if (event.key === "0") {
    event.preventDefault();
    resetZoom();
  }
});
