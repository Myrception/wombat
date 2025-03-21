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
      // Scale font size proportionally with zoom
      const fontSize = Math.max(12, Math.round(14 * zoomLevel));
      editor.updateOptions({ fontSize: fontSize });
    }
  }

  onMount(() => {
    // Get the current zoom level from the data attribute we set in main.js
    const currentZoom = parseFloat(document.body.dataset.zoomLevel || "1.0");
    
    // Set initial font size based on zoom
    const fontSize = Math.max(12, Math.round(14 * currentZoom));
    
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
    
    // Since Monaco editor is within the transformed body, we need to
    // counteract the scaling so that the editor itself isn't double-scaled
    if (Response.parentElement) {
      Response.parentElement.style.transform = `scale(${1/currentZoom})`;
      Response.parentElement.style.transformOrigin = 'top left';
      Response.parentElement.style.width = `${100 * currentZoom}%`;
      Response.parentElement.style.height = `${100 * currentZoom}%`;
    }
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
