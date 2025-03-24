<script>
  import SplitPane from "../controls/SplitPane.svelte";
  import HeaderMetadata from "./HeaderMetadata.svelte";
  import { onMount, onDestroy } from "svelte";
  import { EventsOn } from '../../wailsjs/runtime/runtime';

  export let headers = {};
  export let trailers = {};
  
  let container;
  let zoomLevel = 1.0;
  
  // Subscribe to zoom changes
  const unsubscribeZoom = EventsOn("wombat:zoom_changed", (newZoomLevel) => {
    zoomLevel = newZoomLevel;
  });
  
  // Clean up on component destroy
  onDestroy(() => {
    unsubscribeZoom();
  });
  
  onMount(() => {
    // Initialize zoom level from body data attribute
    zoomLevel = parseFloat(document.body.dataset.zoomLevel || "1.0");
  });
  
  // Reactive statement to update font size when zoom level changes
  $: if (container && zoomLevel) {
    const fontSize = Math.max(12, Math.round(14 * zoomLevel));
    container.style.fontSize = `${fontSize}px`;
  }
</script>

<style>
  .headers-trailers {
    height: calc(100% - 106px);
    width: 100%;
    overflow: hidden;
  }
  section {
    padding: var(--padding);
    overflow: auto;
    height: 100%;
  }
  h2 {
    font-size: var(--font-size);
    margin-top: 0;
  }
</style>

<div class="headers-trailers" bind:this={container}>
  <SplitPane type="vertical" min={50} >
    <section slot="a">
      <h2>Header</h2>
      <HeaderMetadata metadata={headers} />
    </section>
    <section slot="b">
      <h2>Trailer</h2>
      <HeaderMetadata metadata={trailers} />
    </section>
  </SplitPane>
</div>
