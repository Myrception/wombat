<script>
  import Select from "svelte-select";
  import InputLabel from "./InputLabel.svelte";

  export let label = undefined;
  export let hint = "";
  export let frameless = false;
  export let items = [];
  export let selectedValue = undefined;
  export let placeholder = "";
  export let isClearable = false;
  export let showIndicator = true;
  export let isSearchable = false;
  export let labelColor = undefined;
  export let removeable = false;
  export let Item = undefined;
  export let titleProp = "value";
</script>

<style>
  :global(.dropdown div) {
    display: block
  }
  .dropdown {
    display: block;
    width: 100%;
    max-width: 100%;
    margin-bottom: var(--padding);
    --height: 40px;
    --background: var(--bg-input-color);
    --borderRadius: 0;
    --borderFocusColor: var(--border-color);
    --borderHoverColor: var(--border-color);
    --inputColor: var(--text-color);
    --listBackground: var(--bg-inverse-color);
    --listBorderRadius: 0;
    --itemColor: var(--text-inverse-color);
    --itemHoverBG: var(--bg-inverse-color2);
    --itemIsActiveColor: var(--text-inverse-color);
    --itemIsActiveBG: var(--bg-inverse-color3);
    --placeholderColor: var(--border-color);
    --indicatorColor: var(--border-color);
    --clearSelectRight: calc(var(--padding) * 3);
    --clearSelectBottom: 0;
    --clearSelectTop: 3px;
    --clearSelectColor: var(--border-color);
    --clearSelectFocusColor: var(--border-color);
  }

  .frameless {
    width: auto;
    min-width: min(150px, 30%);
    --border: none;
    --background: var(--bg-color);
    color: var(--primary-color);
    margin: 0;
  }
  /* TODO: v2 */
  :global(.frameless .selectContainer) {
    border: none !important;
    background-color: var(--bg-color) !important;
  }
</style>

<div class="dropdown" class:frameless title={selectedValue ? selectedValue[titleProp] : ''} >
  {#if label}
    <InputLabel on:remove {removeable} {label} {hint} color={labelColor} />
  {/if}
  <Select on:select on:clear {items} {Item} bind:selectedValue {placeholder} {isClearable} {showIndicator} {isSearchable} />
</div>
