<script>
  import { onMount, onDestroy } from "svelte";
  import { EventsOn } from '../../wailsjs/runtime/runtime';

  let zoomLevel = 1.0;
  let zoomPercent = "100%";
  
  // Subscribe to zoom change events
  const unsubscribeZoom = EventsOn("wombat:zoom_changed", (newZoomLevel) => {
    updateZoomDisplay(newZoomLevel);
  });
  
  // Cleanup on destroy
  onDestroy(() => {
    unsubscribeZoom();
  });
  
  // Initialize on mount
  onMount(() => {
    // Initialize from current app zoom level
    if (window.appZoom) {
      updateZoomDisplay(window.appZoom.getZoomLevel());
    }
  });
  
  function updateZoomDisplay(zoom) {
    zoomLevel = zoom;
    zoomPercent = Math.round(zoom * 100) + "%";
  }
  
  function handleZoomIn() {
    if (window.appZoom) window.appZoom.zoomIn();
  }
  
  function handleZoomOut() {
    if (window.appZoom) window.appZoom.zoomOut();
  }
  
  function handleResetZoom() {
    if (window.appZoom) window.appZoom.resetZoom();
  }
</script>

<style>
  .zoom-controls {
    display: flex;
    align-items: center;
    padding: 0 calc(var(--padding) * 0.25);
    background-color: var(--bg-color2);
    border-radius: 4px;
    margin-left: var(--padding);
  }
  
  button {
    background: none;
    border: none;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    padding: 0;
    margin: 0 2px;
    color: var(--text-color2);
  }
  
  button:hover {
    color: var(--primary-color);
  }
  
  .zoom-level {
    margin: 0 4px;
    min-width: 36px;
    text-align: center;
    font-size: calc(var(--font-size) * 0.9);
    color: var(--text-color2);
  }
</style>

<div class="zoom-controls">
  <button on:click={handleZoomOut} title="Zoom out (Ctrl+-)">
    <svg width="16" height="16" viewBox="0 0 16 16">
      <path d="M2 8h12" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    </svg>
  </button>
  
  <div class="zoom-level" title="Current zoom level">
    {zoomPercent}
  </div>
  
  <button on:click={handleZoomIn} title="Zoom in (Ctrl+=)">
    <svg width="16" height="16" viewBox="0 0 16 16">
      <path d="M2 8h12M8 2v12" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    </svg>
  </button>
  
  <button on:click={handleResetZoom} title="Reset zoom (Ctrl+0)">
    <svg width="16" height="16" viewBox="0 0 16 16">
      <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" stroke-width="2" />
      <path d="M5 8h6" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    </svg>
  </button>
</div>
