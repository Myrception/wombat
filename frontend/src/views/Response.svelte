<script>
  import { onMount, onDestroy } from "svelte";
  import { zoom, editorFontSize } from "../zoom";

  export let model;

  let Response;
  let editor;

  onMount(() => {
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
      fontSize: editorFontSize(),
      padding: {
        top: 12,
        bottom: 12,
      },
      scrollbar: {
        useShadows: false,
      },
    });
  });

  // The host cancels CSS zoom (Monaco can't handle it), so scale the font.
  const unsubscribeZoom = zoom.subscribe((z) => {
    if (editor) editor.updateOptions({ fontSize: editorFontSize(z) });
  });

  onDestroy(() => {
    unsubscribeZoom();
    if (editor) editor.dispose();
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
  <div bind:this={Response} class="container unzoomed"></div>
</div>
