<script>
  import { getContext, onMount } from 'svelte';

  import Button from '../controls/Button.svelte';
  import { ImportCommand } from '../../wailsjs/go/app/api';

  let EditorContainer;
  let Editor

  export let commands;

  onMount(() => {
    const model = monaco.editor.createModel(commands.grpcurl, "shell");
    Editor = monaco.editor.create(EditorContainer, {
      model,
      minimap: { enabled: false },
      wordWrap: 'on',
      theme: 'nord-dark',
      links: false,
      matchBrackets: "never",
      renderIndentGuides: false,
      renderValidationDecorations: 'off',
      scrollBeyondLastLine: false,
      selectionHighlight: false,
      automaticLayout: true,
      hideCursorInOverviewRuler: true,
      overviewRulerBorder: false,
      padding: {
        top: 12,
        bottom: 12,
      },
      scrollbar: {
        useShadows: false,
      },
    });
  });

  const { close } = getContext('modal');
  const onImportClicked = async () => {
    close();
    await ImportCommand("grpcurl", Editor.getValue())
  };
</script>

<div class="code-edit-panel">
  <h1>Edit as GRPCURL</h1>
  <div bind:this={EditorContainer} class="editor-container" />
  <div class="spacer" />
  <footer>
    <Button text="Close" on:click={close} />
    <Button
      text="Import"
      bgColor="var(--accent-color3)"
      on:click={onImportClicked}
    />
  </footer>
</div>
<style>
  .code-edit-panel {
    width: min(90vw, 800px);
    height: min(90vh, 650px);
    display: flex;
    flex-flow: column;
  }

  .editor-container {
    height: 100%;
    user-select: text;
    -webkit-user-select: text;
    margin-bottom: 5px;
  }

  footer {
    display: flex;
    justify-content: flex-end;
    padding-top: var(--padding);
    border-top: var(--border);
    margin-top: calc(-1 * var(--padding));
    height: 52px;
  }

  .spacer {
    flex-grow: 1;
  }
</style>
