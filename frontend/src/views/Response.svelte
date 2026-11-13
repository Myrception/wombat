<script>
  import { onMount, onDestroy } from "svelte";
  import { monaco, baseOptions } from "../monaco";

  export let model;

  let Response;
  let editor;

  onMount(() => {
    editor = monaco.editor.create(Response, {
      ...baseOptions,
      model,
      readOnly: true,
      renderLineHighlight: "none",
      lineNumbers: "off",
    });
  });

  onDestroy(() => {
    if (editor) editor.dispose();
  });
</script>

<style>
  .response {
    box-sizing: border-box; /* the global padding (App.svelte) counts towards the size */
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
  <div bind:this={Response} class="container"></div>
</div>
