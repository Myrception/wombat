<script>
  // A method in the method dropdown: an icon for its streaming type and its
  // name. Rendered inside svelte-select's item, which handles hover/active.
  import UnaryIcon from "../icons/Unary.svelte";
  import ClientStreamIcon from "../icons/ClientStream.svelte";
  import ServerStreamIcon from "../icons/ServerStream.svelte";
  import BidirectionalIcon from "../icons/Bidirectional.svelte";

  export let item = undefined;

  let reqType;
  let Icon;
  $: {
    if (item.client_stream && item.server_stream) {
      reqType = "Bidirectional streaming";
      Icon = BidirectionalIcon;
    } else if (item.client_stream) {
      reqType = "Client streaming";
      Icon = ClientStreamIcon;
    } else if (item.server_stream) {
      reqType = "Server streaming";
      Icon = ServerStreamIcon;
    } else {
      reqType = "Unary";
      Icon = UnaryIcon;
    }
  }
</script>

<style>
  .method {
    display: flex;
    justify-content: flex-start;
    align-items: center;
    min-width: 0;
  }
  .text {
    text-overflow: ellipsis;
    overflow: hidden;
    white-space: nowrap;
  }
  .icon-wrapper {
    width: 18px;
    height: 18px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 12px;
    margin-left: -6px;
  }
  .icon-wrapper :global(.icon) {
    fill: var(--accent-color3);
  }
</style>

<div class="method" title={item.value}>
  <div class="icon-wrapper" title={reqType}>
    <svelte:component this={Icon} class="icon" />
  </div>
  <div class="text">{item.label}</div>
</div>
