<script>
  import ModalContext from "svelte-simple-modal";
  import Header from "./Header.svelte";
  import Content from "./Content.svelte";
  import Errors from "./Errors.svelte";
  import Updater from "./Updater.svelte";
  import { setFieldRenderer } from './FieldContext';
  import MessageField from './MessageField.svelte';

  setFieldRenderer(MessageField);

  const modalSettings = {
    key: 'modal',
    transitionBg: () => {},
    transitionWindow: () => {},
    styleBg: {
      backgroundColor: '#22222377',
    },
    styleWindow: {
      borderRadius: 0,
      backgroundColor: "#2e3440",
      border: "1px solid #3b4252",
      color: "#eceff4",
      width: "838px",
    },
    styleContent: {
      padding: "12px",
      width: "100%",
    },
    styleCloseButton: {
      borderRadius: 0,
    },
    closeButton: false,
  }
</script>

<style>
  :root {
    --app-scale: 1;

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

    --padding: calc(12px * var(--app-scale));
    --font-size: calc(10pt * var(--app-scale));
    --border: calc(1px * var(--app-scale)) solid #3b4252;

    --icon-size: calc(16px * var(--app-scale));
    --input-height: calc(40px * var(--app-scale));
    --btn-height: calc(40px * var(--app-scale));
    --header-height: calc(40px * var(--app-scale));
  }

  :global(html,body) {
    margin: 0;
    height: 100vh;
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
  
  :global(button),
  :global(input),
  :global(textarea),
  :global(select) {
    font-size: var(--font-size);
  }
  :global(.monaco-editor) {
  font-size: var(--font-size) !important;
  }
  
  :global(.header),
  :global(.content),
  :global(.footer),
  :global(.workspace-options),
  :global(.method-input),
  :global(.response) {
    padding: var(--padding);
  }
  :global(svg) {
  transform-origin: center;
  transform: scale(var(--app-scale));
  }
  
  /* Ensure fixed dimensions scale with zoom */
  :global([class*="fixed-"]) {
    transform-origin: top left;
    transform: scale(var(--app-scale));
  }
  
  /* Apply zoom transition for smooth scaling */
  :global(body) {
    transition: font-size 0.2s ease;
  }
  
  :global(*) {
    transition: padding 0.2s ease, margin 0.2s ease, height 0.2s ease, width 0.2s ease;
  }

  .app {
    height: 100vh;
    display: flex;
    flex-flow: column;
    overflow: hidden;
  }

  /* Add helper class for fixed-position elements that need special handling when zooming */
  :global(.fixed-position) {
    position: fixed;
    /* These elements will be adjusted by our JavaScript zoom handler */
  }
  
  /* Add a transition for smoother zoom experience */
  :global(body) {
    transition: transform 0.2s ease;
  }
  
  /* Make sure modals and dialogs appear correctly at all zoom levels */
  :global(.modal), :global(.overlay) {
    /* Add properties to ensure modals appear correctly at any zoom level */
  }
  
  /* Prevent unnecessary scrollbars from appearing */
  :global(html) {
    overflow: hidden;
  }
</style>

<main class="app">
  <Errors />
  <ModalContext {...modalSettings} >
    <Header />
    <Content />
  </ModalContext>
  <Updater />
</main>

