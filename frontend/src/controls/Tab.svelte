<script>
  import { getContext } from 'svelte';
  import { TABS } from './Tabs.svelte';

  const tab = {};
  const { registerTab, selectTab, selectedTab } = getContext(TABS);

  registerTab(tab);
</script>

<style>
  button {
    background: none;
    border: none;
    border-radius: 0;
    margin: -1px;
    min-width: clamp(60px, 15vw, 100px);
    padding: var(--padding);
    color: var(--text-color2);
    font-size: var(--font-size);
    justify-content: center;
    outline: none;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .selected {
    border-left: var(--border);
    border-right: var(--border);
    border-top: var(--border);
    border-bottom: 1px solid var(--bg-color);
    color: var(--text-color);
    font-weight: 600;
  }
</style>

<button class:selected="{$selectedTab === tab}" on:click="{() => selectTab(tab)}">
  <slot></slot>
  </button>
