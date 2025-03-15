<script>
  import { onMount, onDestroy } from "svelte";
  import { EventsOn } from '../../wailsjs/runtime/runtime';

  export let model;

  let Response;
  let editor;
  
  // Subscribe to zoom change events
  const unsubscribeZoom = EventsOn("wombat:zoom_changed", (newZoomLevel) => {
    updateEditorFontSize(newZoomLevel);
  });
  
  // Clean up on component destroy
  onDestroy(() => {
    unsubscribeZoom();
  });
  
  function updateEditorFontSize(zoomLevel) {
    if (editor) {
      // Scale font size inversely with zoom to maintain relative size
      const fontSize = Math.max(12, Math.floor(14 / zoomLevel));
      editor.updateOptions({ fontSize: fontSize });
    }
  }

  onMount(() => {
    // Get the current zoom level
    const currentZoom = window.appZoom ? window.appZoom.getZoomLevel() : 1.0;
    
    // Scale font size inversely with zoom to maintain relative size
    const fontSize = Math.max(12, Math.floor(14 / currentZoom));
    
    editor = monaco.editor.create(Response, {
      model: model,
      readOnly: true,
      minimap: { enabled: false },
      wordWrap: "on",
      theme: "nord-dark",
      links: false,
      matchBrackets: "never",
      renderIndentGuides: false,
      renderLineHighlight: "none",
      renderValidationDecorations: "off",
      scrollBeyondLastLine: false,
      selectionHighlight: false,
      automaticLayout: true,
      hideCursorInOverviewRuler: true,
      overviewRulerBorder: false,
      lineNumbers: "off",
      fontSize: fontSize, // Set font size based on zoom level
      padding: {
        top: 12,
        bottom: 12,
      },
      scrollbar: {
        useShadows: false,
      },
    });
  });
</script>

<style>
  .response {
    height: calc(100% - 82px);
    width: 100%;
  }

  .response :global(.monaco-editor .cursors-layer > .cursor) {
    display: none !important;
  }

  .container {
    height: 100%;
    width: 100%;
    user-select: text;
    -webkit-user-select: text;
  }
</style>

<div class="response">
  <div bind:this={Response} class="container" />
</div>
