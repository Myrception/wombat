<script>
  export let text = "";
  export let color = "var(--text-color)";
  export let bgColor = "var(--bg-color)";
  export let border = false;
  export let style = "";
  export let disabled = false;
</script>

<button
  on:click
  {disabled}
  class:border
  style="background-color:{bgColor}; color:{color}; {style}">
  {text}
  <slot />
</button>

<style>
  button {
    border: 0;
    margin: 0;
    min-width: clamp(60px, 10vw, 100px);
    align-items: center;
    justify-content: center;
    padding: var(--padding);
    font-size: var(--font-size);
    outline: none;
    background-color: var(--bg-color);
    color: var(--text-color);
  }
  button:active {
    background-color: var(--bg-color2) !important;
  }
  button:disabled {
    opacity: 0.5;
  }
  .border {
    border: var(--border);
  }
</style>
