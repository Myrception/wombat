<script>
  import InputLabel from "./InputLabel.svelte";
  import { createEventDispatcher } from "svelte";
  export let label = undefined;
  export let hint = "";
  export let value = "";
  export let width = "100%";
  export let labelColor = undefined;
  export let removeable = false;
  const dispatch = createEventDispatcher();
  const onDataChange = (event) => dispatch("change", event);
</script>

<style>
  .text-area {
    flex-flow: column;
    margin-bottom: var(--padding);
    width: 100%;
  }
  textarea {
    width: 100%;
    box-sizing: border-box;
    background-color: var(--bg-input-color);
    border: var(--border);
    height: min(150px, 20vh);
    outline: none;
    color: var(--text-color);
    padding: var(--padding);
    resize: none;
  }

  textarea::selection {
    background-color: var(--accent-color2);
  }
</style>

<div class="text-area" style="width:{width};">
  {#if label}
    <InputLabel on:remove {removeable} {label} {hint} color={labelColor} />
  {/if}
  <textarea bind:value on:change={onDataChange} />
</div>
