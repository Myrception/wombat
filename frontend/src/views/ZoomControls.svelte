<script>
  import { zoom, zoomIn, zoomOut, resetZoom } from "../zoom";
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
  <button on:click={zoomOut} title="Zoom out (Ctrl+-)">
    <svg width="16" height="16" viewBox="0 0 16 16">
      <path d="M2 8h12" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    </svg>
  </button>
  
  <div class="zoom-level" title="Current zoom level">
    {Math.round($zoom * 100)}%
  </div>
  
  <button on:click={zoomIn} title="Zoom in (Ctrl+=)">
    <svg width="16" height="16" viewBox="0 0 16 16">
      <path d="M2 8h12M8 2v12" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    </svg>
  </button>
  
  <button on:click={resetZoom} title="Reset zoom (Ctrl+0)">
    <svg width="16" height="16" viewBox="0 0 16 16">
      <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" stroke-width="2" />
      <path d="M5 8h6" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
    </svg>
  </button>
</div>
