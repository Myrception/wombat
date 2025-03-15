<script>
  import KeyValList from "../controls/KeyValList.svelte";

  export let metadata;

</script>

<style>
  .request-metadata {
    padding: var(--padding);
    overflow: auto; /* Change from scroll to auto */
    height: calc(100% - 106px);
    width: calc(100% - 2 * var(--padding));
  }
</style>

<div class="request-metadata">
  <KeyValList bind:list={metadata} />
</div>
