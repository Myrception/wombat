<script>
  import { getFieldRenderer } from './FieldContext';
  import InputLabel from "../controls/InputLabel.svelte";
  import Checkbox from "../controls/Checkbox.svelte";

  export let name = "";
  export let message = {};
  export let state;
  export let key;
  export let idx;
  export let oneof = false;

  // Get MessageField component from context
  const MessageFieldComponent = getFieldRenderer();

  let val, labelColor, removeable;
  $: {
    val = key !== undefined ? key : idx >= 0 ? idx : name;
    if (!state[val] && (oneof || idx >= 0)) {
      state[val] = {}
    }
    labelColor = key !== undefined ? "var(--accent-color3)" : idx >= 0 ? "var(--accent-color2)" : undefined;
    removeable = idx >= 0;
  }

  const onEnabledChanged = ({ detail: checked}) => {
    state[val] = checked ? {} : null
  }
</script>

<style>
  .fields {
    padding-left: var(--padding);
    position: relative;
    width: calc(100% - var(--padding));
  }

  .msg-border {
    position: absolute;
    width: 1px;
    height: calc(100% + 5px);
    background-color: var(--accent-color);
    top: -5px;
    left: 5px;
  }
  .msg-label {
    display: flex;
    align-items: center;
    min-width: auto;
    width: 100%;
    margin-bottom: var(--padding);
  }
</style>

<div class="msg-label">
  <InputLabel on:remove {removeable} label={name} color={labelColor} hint={message.full_name} block />
  {#if !oneof}
    <Checkbox style="margin-bottom: 0" checked={!!state[val]} on:check={onEnabledChanged}/>
  {/if}
</div>

{#if state[val] && message.fields }
  <div class="fields">
    <div class="msg-border" />
    {#each message.fields as field }
      <svelte:component this={MessageFieldComponent} {field} state={state[val]} />
    {/each}
  </div>
{/if}
