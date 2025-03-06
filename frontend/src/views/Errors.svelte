<script>
  import Button from "../controls/Button.svelte";
  import { EventsOn } from '../../wailsjs/runtime/runtime';
  import { onDestroy } from 'svelte';

  let errors = [];

  // Set up event listener with cleanup
  const unsubscribe = EventsOn("wombat:error", err => {
    errors = [...errors, err];
  });

  // Clean up on component destroy
  onDestroy(() => {
    unsubscribe();
  });

  const onOKClicked = () => {
    errors.shift();
    errors = errors;
  }
</script>

<style>
  .errors {
    position: fixed;
    left: 0;
    top: 0;
    right: 0;
    bottom: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
    background-color: #22222377;
  }
  .error-box {
    border: var(--border);
    background-color: var(--bg-color);
    width: 450px;
    padding: var(--padding);
  }

  header {
    font-weight: 600;
    color: var(--red-color);
    margin-bottom: var(--padding);
  }
  footer {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--padding);
  }
</style>

{#if errors.length > 0}
<div class="errors">
  <div class="error-box">
    <header>{errors[0].title}</header>
    <div>{errors[0].msg}</div>
    <footer>
      <Button on:click={onOKClicked} text="OK" border />
    </footer>
  </div>
</div>
{/if}
