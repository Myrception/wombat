<script>
  import { getContext, onMount } from 'svelte';

  import Button from '../controls/Button.svelte';
  import { ImportCommand } from '../../wailsjs/go/app/api';
  import { monaco, baseOptions } from '../monaco';

  let EditorContainer;
  let Editor;

  export let commands;

  onMount(() => {
    const model = monaco.editor.createModel(commands.grpcurl, "shell");
    Editor = monaco.editor.create(EditorContainer, { ...baseOptions, model });
    return () => {
      Editor.dispose();
      model.dispose();
    };
  });

  const { close } = getContext('modal');
  const onImportClicked = async () => {
    close();
    await ImportCommand("grpcurl", Editor.getValue())
  };
</script>

<div class="code-edit-panel">
  <h1>Edit as GRPCURL</h1>
  <div bind:this={EditorContainer} class="editor-container"></div>
  <div class="spacer"></div>
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
    width: min(calc(90vw / var(--zoom)), 800px);
    /* The modal's content is at most 100vh - 4rem tall (App.svelte),
       including its padding; fill that, but no more. */
    height: min(calc(100vh / var(--zoom) - 4rem - 2 * var(--padding)), 650px);
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
