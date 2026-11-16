<script>
  // A message edited as JSON text: google.protobuf.Struct, Value and
  // ListValue, and recursive messages (see wellKnown.js).
  import { getContext } from "svelte";
  import InputLabel from "../controls/InputLabel.svelte";
  import Checkbox from "../controls/Checkbox.svelte";
  import TextArea from "../controls/TextArea.svelte";
  import { unknownFields } from "./jsonCheck";
  import { InvalidJSON } from "./invalidJSON";

  export let field;
  export let state;
  export let key;
  export let idx;
  export let message = undefined; // the field's message type

  const messages = getContext("messages"); // a store; missing outside a method form

  // A list item or map value: always present (removed with its ✕, not
  // unticked), since null is not a valid item.
  $: item = idx >= 0 || key !== undefined;
  $: val = key !== undefined ? key : idx >= 0 ? idx : field.name;
  $: labelColor =
    key !== undefined ? "var(--accent-color3)" : idx >= 0 ? "var(--accent-color2)" : undefined;
  $: removeable = idx >= 0;

  // What a newly enabled field or new item starts with (valid for the type).
  const empty = () => (field.kind === "google.protobuf.ListValue" ? [] : {});

  const UNSEEN = {};
  let written = UNSEEN; // the value the text box shows
  let v; // the text box
  let error = "";
  let warning = "";

  const check = value => {
    const unknown = message?.recursive && messages ? unknownFields(value, message, $messages) : [];
    warning = unknown.length
      ? `Unknown field${unknown.length > 1 ? "s" : ""} ${unknown.map(u => `"${u}"`).join(", ")}, not sent`
      : "";
  };

  const invalid = message =>
    idx >= 0
      ? `Invalid JSON, the request cannot be sent until it is fixed: ${message}`
      : `Invalid JSON, not sent: ${message}`;

  // Show the value, and show it again when it is changed from outside, e.g.
  // when an item before this one is removed from a list.
  $: sync(state[val]);
  function sync(cur) {
    if (cur === written) return;
    if (written === UNSEEN && cur == null && (idx >= 0 || key !== undefined)) {
      cur = empty();
      state[val] = cur;
    }
    written = cur;
    if (cur instanceof InvalidJSON) {
      v = cur.text;
      error = invalid(cur.error);
      warning = "";
      return;
    }
    v = cur === undefined ? undefined : JSON.stringify(cur);
    error = "";
    check(cur);
  }

  const set = value => {
    written = value;
    state[val] = value;
  };

  const onEnabledChanged = ({ detail: checked }) => {
    set(checked ? empty() : undefined);
    v = checked ? JSON.stringify(written) : undefined;
    error = warning = "";
  };

  const onDataChange = ({ detail: event }) => {
    v = event.target.value;
    try {
      const parsed = JSON.parse(v);
      set(parsed);
      error = "";
      check(parsed);
    } catch (e) {
      // Kept with its text; not sent while it is invalid. (null would be
      // sent: it is a valid google.protobuf.Value.)
      set(new InvalidJSON(v, e.message));
      warning = "";
      error = invalid(e.message);
    }
  };

  // A list item or map value always has its text box.
  $: shown = item || state[val] !== undefined;
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
  {#if !item}
    <Checkbox style="margin-bottom: 0" checked={shown} on:check={onEnabledChanged} />
  {/if}
</div>

{#if shown}
  <TextArea on:remove {removeable} on:change={onDataChange} value={v} />
  {#if error}
    <div class="note error" role="alert">{error}</div>
  {:else if warning}
    <div class="note warning" role="status">{warning}</div>
  {/if}
{/if}

<style>
  .msg-label {
    display: flex;
    align-items: center;
    min-width: 400px;
    margin-bottom: var(--padding);
  }
  .note {
    margin: calc(var(--padding) * -0.5) 0 var(--padding);
  }
  .error {
    color: var(--red-color);
  }
  .warning {
    color: var(--yellow-color);
  }
</style>
