<script>
  import InputPane from "./InputPane.svelte";
  import OutputPane from "./OutputPane.svelte";
  import SplitPane from "../controls/SplitPane.svelte";
</script>

<style>
  .content {
    flex: 1;
    min-height: 0;
    height: 0;
    width: 100%;
    overflow: hidden;
    position: relative;
  }

  section {
    width: 100%;
    height: 100%;
    overflow: hidden;
  }
</style>

<div class="content">
  <SplitPane>
    <section slot="a">
      <InputPane />
    </section>
    <section slot="b">
      <OutputPane />
    </section>
  </SplitPane>
</div>
