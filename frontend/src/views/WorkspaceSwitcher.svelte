<script>
  import WorkspaceList from "./WorkspaceList.svelte";

  import { GetWorkspaceOptions, ListWorkspaces, SelectWorkspace, DeleteWorkspace} from '../../wailsjs/go/app/api';
  export let visible = false;

  let workspaces = [];
  let current = undefined;
  const loadWorkspacesList = async () => {
      workspaces = [];
      current = await GetWorkspaceOptions()
      workspaces = await ListWorkspaces() 
  }

  $: visible && loadWorkspacesList();

  const onWorkspaceSelected = ({detail: wksp}) => {
    SelectWorkspace(wksp.id);
    visible = false;
  }

  const onWorkspaceDeleted = ({detail:wksp}) => {
    DeleteWorkspace(wksp.id);
    visible = false;
  }

</script>

<style>
.overlay {
  position: fixed;
  left: 0;
  top: 0;
  right: 0;
  bottom: 0;
  z-index: 999;
  background-color: #22222377;
}

.panel {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  border: var(--border);
  background-color: var(--bg-color);
  width: min(400px, calc(90vw / var(--zoom)));
  height: min(300px, calc(80vh / var(--zoom)));
  padding: var(--padding);
  display: flex;
  flex-flow: column;
}
  h1 {
    width: 100%;
    padding: 0 12px 12px 12px;
    margin: 0 -12px 12px -12px;
    font-size: calc(var(--font-size) + 4px);
    font-weight: 600;
    border-bottom: var(--border);
  }
</style>

<svelte:window on:keydown={e => visible && e.key === "Escape" && (visible = false)} />

{#if visible}
  <!-- Clicking the backdrop closes the panel; Escape is the keyboard equivalent. -->
  <!-- svelte-ignore a11y-click-events-have-key-events a11y-no-static-element-interactions -->
  <div class="overlay" on:click|self={() => visible = false}>
    <div class="panel">
      <h1>Select Workspace</h1>
      <WorkspaceList on:select={onWorkspaceSelected} on:delete={onWorkspaceDeleted} {current} {workspaces} />
    </div>
</div>
{/if}
