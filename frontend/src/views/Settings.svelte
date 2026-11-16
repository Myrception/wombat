<script>
  import { getContext } from "svelte";
  import Button from "../controls/Button.svelte";
  import TextField from "../controls/TextField.svelte";
  import {
    maxMessages, parseMaxMessages, DEFAULT_MAX_MESSAGES, MIN_MAX_MESSAGES, MAX_MAX_MESSAGES,
  } from "../settings";

  const { close } = getContext("modal");

  let messages = String($maxMessages);
  $: valid = parseMaxMessages(messages.trim()) !== undefined;

  const onSave = () => {
    if (!valid) return;
    maxMessages.set(parseMaxMessages(messages.trim()));
    close();
  };
  const onReset = () => (messages = String(DEFAULT_MAX_MESSAGES));
</script>

<div class="settings">
  <h1>Settings</h1>
  <TextField label="Messages kept per call:" bind:value={messages} placeholder={String(DEFAULT_MAX_MESSAGES)} />
  <p class:invalid={!valid}>
    The response and statistics keep the last {valid ? messages.trim() : "N"} sent and received messages of a
    call; older ones are removed, so long streams stay fast. A whole number from {MIN_MAX_MESSAGES} to
    {MAX_MAX_MESSAGES.toLocaleString("en")}, default {DEFAULT_MAX_MESSAGES}.
  </p>
  <footer>
    <Button text="Default" on:click={onReset} />
    <div class="spacer"></div>
    <Button text="Close" on:click={close} />
    <Button text="Save" bgColor="var(--accent-color3)" disabled={!valid} on:click={onSave} />
  </footer>
</div>

<style>
  .settings {
    display: flex;
    flex-flow: column;
  }
  h1 {
    padding: 0 12px 12px 12px;
    margin: 0 -12px 12px -12px;
    font-size: calc(var(--font-size) + 4px);
    font-weight: 600;
    border-bottom: var(--border);
  }
  p {
    margin: 0 0 var(--padding);
    color: var(--text-color3);
  }
  p.invalid {
    color: var(--red-color);
  }
  footer {
    display: flex;
    gap: var(--padding);
  }
  .spacer {
    flex-grow: 1;
  }
</style>
