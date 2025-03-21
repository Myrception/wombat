<script>
  import { createEventDispatcher } from "svelte";
  import Button from '../controls/Button.svelte';
  import MessageField from "./MessageField.svelte";

  export let methodInput = {
    full_name: "",
    fields: []
  };

  export let state;
  export let mapItems;

  let methodSelected;

  const dispatch = createEventDispatcher();
  const onEdit = () => dispatch("edit");

</script>

<style>
  .method-input {
    padding: var(--padding);
    overflow: auto;
    height: calc(100% - 106px);
    width: calc(100% - 2 * var(--padding));
    position: relative;
  }
  h2 {
    font-size: var(--font-size);
    font-weight: 400;
  }
  .fields {
    margin-left: var(--padding);
    display: flex;
    flex-flow: column;
    width: calc(100% - 2 * var(--padding));
    padding-bottom: 60px; /* Add space for the edit button */
  }
  /* Remove absolute positioning for the edit button */
  .edit {
    margin-top: 20px;
    display: flex;
    justify-content: flex-end;
    /* Position at the bottom of the content, not absolutely positioned */
  }
</style>

<div class="method-input">
  <h2>{methodInput.full_name}</h2>
  <div class="fields">
    {#each methodInput.fields || [] as field}
      <MessageField {field} {state} {mapItems} />
    {/each}
  </div>
  <div class="edit">
    <Button 
      text="Edit"
      color={isWin ? "#81a1c1" : "var(--accent-color2)"}
      bgColor="transparent"
      style="min-width:auto"
      on:click={onEdit}
    />
  </div>
</div>
