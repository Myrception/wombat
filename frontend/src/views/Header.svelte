<script>
  import { getContext, onDestroy } from "svelte";
  import Button from "../controls/Button.svelte";
  import WorkspaceOptions from "./WorkspaceOptions.svelte";
  import WorkspaceSwitcher from "./WorkspaceSwitcher.svelte";
  import ZoomControls from "./ZoomControls.svelte"; // Import the new zoom controls
  import { EventsOn } from '../../wailsjs/runtime/runtime';

  let addr = "";
  let status = "";
  
  const unsubscribe1 = EventsOn("wombat:client_connected", data => addr = data);
  const unsubscribe2 = EventsOn("wombat:client_state_changed", data => status = data.toLowerCase());
  const unsubscribe3 = EventsOn("wombat:client_connect_started", data => {
    addr = data;
    status = "connecting";
  });

 onDestroy(() => {
    unsubscribe1();
    unsubscribe2();
    unsubscribe3();
  });

  const { open } = getContext('modal');
  const onWorkspaceClicked = () => open(WorkspaceOptions);
  const onNewWorkspaceClicked = () => open(WorkspaceOptions, {createNew: true});

  let wkspSelectorVisible = false;
</script>

<style>
.header {
    height: auto;
    min-height: 40px;
    max-height: 80px;
    padding: calc(var(--padding) * 0.5);
    border-bottom: var(--border);
    display: flex;
    flex-flow: row nowrap;
    align-items: center;
    justify-content: space-between;
    overflow: visible;
    position: relative;
    z-index: 10;
  }

  .connection {
    display: flex;
    flex-flow: column;
    align-items: center;
  }

  .workspace-select {
    display: flex;
    margin-left: calc(var(--padding) * 0.5);
    overflow: hidden;
    flex-shrink: 1;
    max-width: 50%;
  }

  h1 {
    font-size: calc(var(--font-size) + 2px);
    margin: 0;
    color: var(--primary-color);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 250px; 
  }

  h3 {
    font-size: var(--font-size);
    margin: 0;
  }

  h3.connecting {
    color: var(--yellow-color);
  }

  h3.ready {
    color: var(--green-color);
  }

  h3.transient_failure, h3.shutdown {
    color: var(--red-color);
  }

  .hitem {
    flex: 1;
    display: flex;
    align-items: center;
  }

  line {
    stroke: var(--accent-color3);
    stroke-width: 2;
  }

  path {
    fill: var(--border-color);
  }

  .dropdown-indicator {
    margin-left: var(--padding);
  }
  :global(.header .button) {
    min-width: auto !important;
    padding: calc(var(--padding) * 0.5) var(--padding) !important;
  }

  .hitem {
    flex: 0 0 auto;
    display: flex;
    align-items: center;
    margin-right: var(--padding);
  }

  .right-section {
    display: flex;
    align-items: center;
    flex: 0 0 auto;
    margin-left: auto;
  }
  
  /* Responsive styles */
  @media (max-width: 768px) {
    .header {
      flex-wrap: wrap;
    }
    
    .hitem, .workspace-select, .right-section {
      margin: 4px 0;
    }
    
    h1 {
      max-width: 200px;
    }
  }
</style>

<div class="header">
  <div class="hitem">
    <Button
      text="Workspace"
      bgColor={isWin ? "#5e81ac" : "var(--accent-color3)"}
      on:click={onWorkspaceClicked}
    /><Button
      bgColor={isWin ? "#81a1c1" : "var(--accent-color2)"}
      on:click={onNewWorkspaceClicked}
      style="height:40px;min-width:auto;" >
      <svg width="14" height="14">
        <line x1="0" y1="7" x2="14" y2="7" />
        <line x1="7" y1="0" x2="7" y2="14" /> 
      </svg>
    </Button>
  </div>
  
  <div on:click={() => wkspSelectorVisible = true} class="workspace-select">
    <div class="connection">
      <h1>{addr}</h1>
      <h3 class={status}>{status}</h3>
    </div>
    <svg class="dropdown-indicator" width="20" height="20" viewBox="0 0 20 20">
      <path d="M4.516 7.548c0.436-0.446 1.043-0.481 1.576 0l3.908 3.747
        3.908-3.747c0.533-0.481 1.141-0.446 1.574 0 0.436 0.445 0.408 1.197 0
        1.615-0.406 0.418-4.695 4.502-4.695 4.502-0.217 0.223-0.502
        0.335-0.787 0.335s-0.57-0.112-0.789-0.335c0
        0-4.287-4.084-4.695-4.502s-0.436-1.17 0-1.615z" />
    </svg>
  </div>
  
  <!-- Add the zoom controls to the right side -->
  <div class="right-section">
    <ZoomControls />
  </div>
</div>

<WorkspaceSwitcher bind:visible={wkspSelectorVisible} />
