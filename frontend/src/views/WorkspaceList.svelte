<script>
  import { createEventDispatcher } from "svelte";
  import CrossButton from "../controls/CrossButton.svelte";

  export let current;
  export let workspaces;

  const dispatch = createEventDispatcher();
  const onWorkspaceClicked = wksp => {
    if (confirming === undefined) dispatch("select", wksp);
  };

  // Deleting cannot be undone, so the ✕ asks first.
  let confirming = undefined; // id of the workspace asked about
  const onDeleteClicked = (e, wksp) => {
    e.stopPropagation();
    confirming = wksp.id;
  };
  const onConfirm = (e, wksp) => {
    e.stopPropagation();
    confirming = undefined;
    dispatch("delete", wksp);
  };
  const onKeep = e => {
    e.stopPropagation();
    confirming = undefined;
  };

</script>

<style>
  div {
    overflow: scroll;
  }
  table {
    width: 100%;
  }
  tr:hover {
    background-color: var(--bg-color3);
  }
  td {
    padding: var(--padding);
    max-width: 100px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  td.selected {
    color: var(--primary-color);
  }
  .confirm {
    color: var(--red-color);
  }
  .confirm button {
    margin-left: var(--padding);
    padding: 2px var(--padding);
    border: var(--border);
    background-color: var(--bg-color);
    color: var(--text-color);
  }
  .confirm button.delete {
    color: var(--red-color);
  }
  .box {
    width: 14px;
    height: 14px;
    background-color: var(--primary-color);
    display: block;
  }
</style>

<div>
  <table cellspacing="0" cellpadding="0">
    <colgroup>
      <col width="20px" />
      <col />
      <col width="20px" />
    </colgroup>
    <tbody>
    {#each workspaces || [] as wksp}
      <tr on:click={() => onWorkspaceClicked(wksp)} title={wksp.addr}>
        <td>
          {#if wksp.id === current.id}
            <span class="box"></span>
          {/if}
        </td>
        {#if confirming === wksp.id}
          <td class="confirm" colspan="2">
            Delete {wksp.addr}?
            <button class="delete" on:click={e => onConfirm(e, wksp)}>Delete</button>
            <button on:click={onKeep}>Keep</button>
          </td>
        {:else}
          <td class:selected={wksp.id === current.id}>{!wksp.addr && wksp.id === "wksp_default" ? "default workspace" : wksp.addr}</td>
          <td>
            {#if wksp.id !== "wksp_default"}
              <CrossButton label="Delete workspace" on:click={e => onDeleteClicked(e, wksp)} color="var(--red-color)" />
            {/if}
          </td>
        {/if}
      </tr>
    {/each}
    </tbody>
  </table>
</div>
