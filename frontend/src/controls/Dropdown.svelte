<script>
  // Our wrapper around svelte-select (v6). Its props and events are our own
  // API, so the components using it do not depend on svelte-select's.
  //
  // `select` fires only when the user picks an option, never when
  // `selectedValue` is changed from code.
  import { createEventDispatcher } from "svelte";
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
  /** Optional component rendering a list item, given `item`. */
  export let Item = undefined;
  export let titleProp = "value";

  const dispatch = createEventDispatcher();
  // svelte-select passes the payload directly (not as an event), and for
  // plain string items it is the string itself. Our `select`/`clear` events
  // always carry an item object: { value, label, ... }.
  const asItem = v => (v !== null && typeof v === "object" ? v : { value: v, label: v });
  const onChange = value => dispatch("select", asItem(value));
  const onClear = value => dispatch("clear", value == null ? value : asItem(value));

  $: title = selectedValue && typeof selectedValue === "object" ? selectedValue[titleProp] ?? "" : selectedValue ?? "";
</script>

<style>
  .dropdown {
    display: block;
    width: 100%;
    max-width: 100%;
    margin-bottom: var(--padding);
  }

  .frameless {
    width: auto;
    min-width: min(150px, 30%);
    color: var(--primary-color);
    margin: 0;
  }

  /* svelte-select positions its list with floating-ui, which computes in
     unzoomed pixels and puts the list in the wrong place under CSS zoom
     (see src/zoom.js). The list always opens below the select anyway. */
  .dropdown :global(.svelte-select-list) {
    position: absolute !important;
    top: calc(100% + 1px) !important;
    left: 0 !important;
    right: 0 !important;
    width: auto !important;
    transform: none !important;
  }
  /* Line the list up with the select's outer border. */
  .dropdown:not(.frameless) :global(.svelte-select-list) {
    left: -1px !important;
    right: -1px !important;
  }
</style>

<div class="dropdown" class:frameless {title}>
  {#if label}
    <InputLabel on:remove {removeable} {label} {hint} color={labelColor} />
  {/if}
  <!-- svelte-select reads generic names (--padding, --border, --height, ...)
       that our theme also defines, so all of them are set here, on the
       Select only. -->
  <Select
    {items}
    bind:value={selectedValue}
    {placeholder}
    clearable={isClearable}
    searchable={isSearchable}
    showChevron={showIndicator}
    onchange={onChange}
    onclear={onClear}
    --height="var(--input-height)"
    --box-sizing="border-box"
    --chevron-height="calc(var(--input-height) - 2px)"
    --padding="0 0 0 12px"
    --margin="0"
    --border={frameless ? "none" : "1px solid var(--border-color)"}
    --border-hover={frameless ? "none" : "1px solid var(--border-color)"}
    --border-focused={frameless ? "none" : "1px solid var(--border-color)"}
    --border-radius="0"
    --background={frameless ? "var(--bg-color)" : "var(--bg-input-color)"}
    --input-color="var(--text-color)"
    --selected-item-color={frameless ? "var(--primary-color)" : "var(--text-color)"}
    --placeholder-color="var(--border-color)"
    --chevron-color="var(--border-color)"
    --clear-select-color="var(--border-color)"
    --list-background="var(--bg-inverse-color)"
    --list-border-radius="0"
    --list-z-index="100"
    --item-color="var(--text-inverse-color)"
    --item-hover-bg="var(--bg-inverse-color2)"
    --item-hover-color="var(--text-inverse-color)"
    --item-is-active-bg="var(--bg-inverse-color3)"
    --item-is-active-color="var(--text-inverse-color)"
    --item-padding="0 12px"
  >
    {#snippet item({ item })}
      {#if Item}
        <Item {item} />
      {:else}
        {item.label}
      {/if}
    {/snippet}
  </Select>
</div>
