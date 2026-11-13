<script>
  // Modal dialogs. Components inside get { open, close } with
  // getContext("modal"):
  //
  //   open(Component, props?)  shows <Component {...props} /> as a dialog
  //   close()                  closes it
  //
  // While a dialog is open the rest of the page is inert (not clickable or
  // focusable, and hidden from screen readers), Tab stays inside the dialog,
  // and Escape or a click on the backdrop closes it. Focus then returns to
  // where it was. The dialog is named after its first heading.
  import { setContext, tick } from "svelte";

  let { children } = $props();

  let Component = $state(null);
  let componentProps = $state({});
  let dialog = $state();
  let labelledBy = $state(undefined);
  let returnFocus = null;

  const FOCUSABLE =
    'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

  const focusables = () =>
    dialog ? [...dialog.querySelectorAll(FOCUSABLE)].filter(el => el.getClientRects().length > 0) : [];

  async function open(component, props = {}) {
    if (!Component) returnFocus = document.activeElement;
    Component = component;
    componentProps = props;
    await tick();
    const heading = dialog?.querySelector("h1, h2");
    if (heading) {
      heading.id ||= "modal-title";
      labelledBy = heading.id;
    } else {
      labelledBy = undefined;
    }
    (focusables()[0] ?? dialog)?.focus();
  }

  async function close() {
    if (!Component) return;
    Component = null;
    componentProps = {};
    const el = returnFocus;
    returnFocus = null;
    await tick(); // the page is no longer inert, so it can take focus again
    if (el?.isConnected) el.focus();
  }

  setContext("modal", { open, close });

  function onKeydown(e) {
    if (!Component) return;
    if (e.key === "Escape") {
      e.preventDefault();
      close();
    } else if (e.key === "Tab") {
      // Keep focus inside the dialog.
      const items = focusables();
      if (items.length === 0) {
        e.preventDefault();
        return;
      }
      const first = items[0];
      const last = items[items.length - 1];
      const inside = dialog.contains(document.activeElement);
      if (e.shiftKey && (document.activeElement === first || !inside)) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && (document.activeElement === last || !inside)) {
        e.preventDefault();
        first.focus();
      }
    }
  }

  // Close on a click on the backdrop, but not when a drag (e.g. selecting
  // text) started inside the dialog and ended outside.
  let pressedOnBackdrop = false;
  const onBackdropDown = e => (pressedOnBackdrop = e.target === e.currentTarget);
  const onBackdropUp = e => {
    if (pressedOnBackdrop && e.target === e.currentTarget) close();
    pressedOnBackdrop = false;
  };
</script>

<svelte:window onkeydown={onKeydown} />

<div class="page" inert={Component ? true : undefined}>
  {@render children?.()}
</div>

{#if Component}
  <!-- The backdrop is not a control: Escape is its keyboard equivalent. -->
  <div class="bg" role="presentation" onmousedown={onBackdropDown} onmouseup={onBackdropUp}>
    <div class="window" role="dialog" aria-modal="true" aria-labelledby={labelledBy} tabindex="-1" bind:this={dialog}>
      <div class="content">
        <Component {...componentProps} />
      </div>
    </div>
  </div>
{/if}

<style>
  /* The page keeps its layout; the wrapper only carries `inert`. */
  .page {
    display: contents;
  }

  .bg {
    position: fixed;
    inset: 0;
    z-index: 1000;
    background-color: #22222377;
  }

  .window {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    box-sizing: border-box; /* the border counts towards max-width */
    width: 838px;
    /* Viewport units are scaled by CSS zoom (see src/zoom.js). */
    max-width: calc(100vw / var(--zoom));
    overflow-x: hidden;
    background-color: var(--bg-color);
    border: 1px solid var(--border-color);
    color: var(--text-color);
    outline: none;
  }

  .content {
    position: relative;
    box-sizing: border-box;
    width: 100%;
    padding: var(--padding);
    max-height: calc(100vh / var(--zoom) - 4rem);
    overflow: auto;
  }
</style>
