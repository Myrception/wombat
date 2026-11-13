<script>
  import Modal from "../controls/Modal.svelte";
  import Header from "./Header.svelte";
  import Content from "./Content.svelte";
  import Errors from "./Errors.svelte";
  import Updater from "./Updater.svelte";
  import { setFieldRenderer } from './FieldContext';
  import MessageField from './MessageField.svelte';

  setFieldRenderer(MessageField);

</script>

<style>
  :root {
    /* Set by src/zoom.js together with CSS zoom on <html>. */
    --zoom: 1;

    --bg-color: #2e3440;
    --bg-color2: #434c5e;
    --bg-color3: #4c566a;
    --bg-inverse-color: #eceffa;
    --bg-inverse-color2: #e5e9f0;
    --bg-inverse-color3: #d8dee9;
    --bg-input-color: #242933;

    --border-color: #3b4252;

    --text-color: #eceff4;
    --text-color2: #d8dee9;
    --text-color3: #bcc7d9;
    --text-inverse-color: #2e3440;

    --primary-color: #88c0d0;
    --accent-color: #8fbcbb;
    --accent-color2: #81a1c1;
    --accent-color3: #5e81ac;

    --red-color: #bf616a;
    --orange-color: #d08770;
    --yellow-color: #ebcb8b;
    --green-color: #a3be8c;
    --purple-color: #b48ead;

    --padding: 12px;
    --font-size: 10pt;
    --border: 1px solid #3b4252;

    --icon-size: 16px;
    --input-height: 41px; /* text inputs and dropdowns */
    --btn-height: 40px;
    --header-height: 40px;
  }

  :global(html,body) {
    margin: 0;
    /* Not 100vh: viewport units are scaled by CSS zoom and would overflow. */
    height: 100%;
    overflow: hidden;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto", "Oxygen",
    "Ubuntu", "Cantarell", "Fira Sans", "Droid Sans", "Helvetica Neue",
    sans-serif;
    font-size: var(--font-size);
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    user-select: none;
    -webkit-user-select: none;
    background-color: var(--bg-color);
    color: var(--text-color);
    cursor: default;
  }
  
  /* Form controls do not inherit the font by default; without this they use
     the engine's default form font, which differs between WebKit and
     Chromium (and so do their heights). */
  :global(button),
  :global(input),
  :global(textarea),
  :global(select) {
    font-family: inherit;
    font-size: var(--font-size);
  }
  
  :global(.header),
  :global(.content),
  :global(.footer),
  :global(.workspace-options),
  :global(.method-input),
  :global(.response) {
    padding: var(--padding);
  }
  .app {
    height: 100%;
    display: flex;
    flex-flow: column;
    overflow: hidden;
  }
</style>

<main class="app">
  <Errors />
  <Modal>
    <Header />
    <Content />
  </Modal>
  <Updater />
</main>
