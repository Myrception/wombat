<script>
  import { getFieldRenderer } from './FieldContext';
  import InputLabel from "../controls/InputLabel.svelte";
  import CrossButton from "../controls/CrossButton.svelte";

  export let field;
  export let state;

  // Get MessageField component from context
  const MessageFieldComponent = getFieldRenderer();
  
  let hint = '';

  const clone = target => {
    if (typeof target === 'object') {
        let cloneTarget = Array.isArray(target) ? [] : {};
        for (const key in target) {
            cloneTarget[key] = clone(target[key]);
        }
        return cloneTarget;
    } 
    return target;
  }

  $: {
    if (!state[field.name]) {
      state[field.name] = []
    }
    hint = field.kind;
    if (field.kind === "message" || field.kind === "group") {
      hint = field.message.full_name;
    }
  }

  const onAddButtonClicked = () => {
    state[field.name] = [...state[field.name], null]
  }

  const onRemove = idx => {
    state[field.name].splice(idx,1);
    state[field.name] = state[field.name];
  }
</script>

<style>
  .msg-label {
    display: flex;
    align-items: center;
    min-width: auto;
    width: 100%;
    margin-bottom: var(--padding);
  }
  .fields {
    padding-left: var(--padding);
    position: relative;
    width: calc(100% - var(--padding));
  }

  .msg-border {
    position: absolute;
    width: 1px;
    height: calc(100%);
    background-color: var(--accent-color2);
    top: 0;
    left: 5px;
  }
</style>

<div class="msg-label">
  <InputLabel label={field.name} hint={"repeated "+hint} block />
  <CrossButton color={isWin ? "#a3be8c" : "var(--green-color)"} add on:click={onAddButtonClicked} />
</div>

<div class="fields">
  <div class="msg-border" />
  {#each state[field.name] || [] as _, i}
    <svelte:component this={MessageFieldComponent} on:remove={() => onRemove(i)} field={field} state={state[field.name]} idx={i} />
  {/each}
</div>
