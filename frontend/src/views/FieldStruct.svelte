<script>
  // A message edited as JSON text: google.protobuf.Struct, Value and
  // ListValue, and recursive messages (see wellKnown.js).
  import InputLabel from "../controls/InputLabel.svelte";
  import Checkbox from "../controls/Checkbox.svelte";
  import TextArea from "../controls/TextArea.svelte";

  export let field;
  export let state;
  export let key;
  export let idx;

  let val, labelColor, removeable, v;
  let error = "";
  $: {
    val = key !== undefined ? key : idx >= 0 ? idx : field.name;
    v === undefined ? (v = JSON.stringify(state[val])) : (v = v);

    labelColor =
      key !== undefined
        ? "var(--accent-color3)"
        : idx >= 0
        ? "var(--accent-color2)"
        : undefined;
    removeable = idx >= 0;
  }

  // What a newly enabled field starts with (must be valid for the type).
  const empty = () => (field.kind === "google.protobuf.ListValue" ? [] : {});

  const onEnabledChanged = ({ detail: checked }) => {
    state[val] = checked ? empty() : undefined;
    v = checked ? JSON.stringify(state[val]) : undefined;
    error = "";
  };
  const onDataChange = ({ detail: event }) => {
    v = event.target.value;
    try {
      state[val] = JSON.parse(event.target.value);
      error = "";
    } catch (e) {
      // Not sent until it is valid JSON.
      state[val] = null;
      error = `Invalid JSON, the field is not sent: ${e.message}`;
    }
  };
</script>

<div class="msg-label">
  <InputLabel
    on:remove
    {removeable}
    label={field.name}
    color={labelColor}
    hint={field.kind}
    block
  />
  <Checkbox
    style="margin-bottom: 0"
    checked={state[val] !== undefined}
    on:check={onEnabledChanged}
  />
</div>

{#if state[val] !== undefined}
  <TextArea on:remove {removeable} on:change={onDataChange} value={v} />
  {#if error}
    <div class="error" role="alert">{error}</div>
  {/if}
{/if}

<style>
  .msg-label {
    display: flex;
    align-items: center;
    min-width: 400px;
    margin-bottom: var(--padding);
  }
  .error {
    margin: calc(var(--padding) * -0.5) 0 var(--padding);
    color: var(--red-color);
  }
</style>
