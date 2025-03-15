<script>
  import Checkbox from "../controls/Checkbox.svelte";
  import TextArea from "../controls/TextArea.svelte";

  export let options = {}
</script>

<style>
  .workspace-options-tls {
    flex-flow: column;
    padding: var(--padding) 0;
    width: 100%;
  }
  .client-certs {
    display: flex;
    flex-direction: column;
    width: 100%;
  }
  @media (min-width: 768px) {
    .client-certs {
      flex-direction: row;
      justify-content: space-between;
    }
    .client-certs > * {
      width: calc(50% - var(--padding) / 2);
    }
  }
</style>

<div class="workspace-options-tls">
  <Checkbox bind:checked={options.plaintext} label="Use plain-text HTTP/2 when connecting to server (no TLS)" rhs />
  <Checkbox bind:checked={options.insecure} label="Skip server certificate and domain verification (insecure)" rhs />
  <TextArea label="Trusted root certificate(s):" bind:value={options.rootca} />
  <div class="client-certs">
    <TextArea label="Client certificate (public key):" bind:value={options.clientcert} />
    <TextArea label="Client private key:" bind:value={options.clientkey} />
  </div>
</div>
