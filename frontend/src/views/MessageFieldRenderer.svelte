<script>
  // No imports of field components here!
  
  export let field = {};
  export let state;
  export let mapItems = {};
  export let oneof = false;
  export let idx = -1;
  export let key = undefined;
  export let components = {}; // Pass components from parent

  const isWellKnown = full_name => {
    // Same function as before
    // ...
  }

  $: if(!field) field = {};
</script>
{#if field.repeated && idx < 0 && components.RepeatedField}
  <svelte:component this={components.RepeatedField} {field} {state} on:remove />

{:else if field.kind === "map" && components.FieldMap}
  <svelte:component this={components.FieldMap} {field} {state} {mapItems} on:remove />

{:else if field.kind === "oneof" && components.FieldOneof}
  <svelte:component this={components.FieldOneof} {field} {state} on:remove />

{:else if (field.kind === "group" || field.kind === "message")}
  {#if isWellKnown(field.message.full_name) && components.FieldWellKnown}
    <svelte:component this={components.FieldWellKnown} name={field.name} message={field.message} {state} {key} {idx} on:remove />
  {:else if components.FieldMessage}
    <svelte:component this={components.FieldMessage} on:remove name={field.name} message={field.message} {state} {key} {oneof} {idx} />
  {/if}

{:else if field.kind === "enum" && components.FieldEnum}
  <svelte:component this={components.FieldEnum} on:remove {field} {state} {key} {idx} />

{:else if field.kind === "bool" && components.FieldBool}
  <svelte:component this={components.FieldBool} on:remove {field} {state} {key} {idx} />
  
{:else if components.FieldText}
  <svelte:component this={components.FieldText} on:remove {field} {state} {key} {idx} multiline={field.kind === "bytes"} />

{:else}
  <div>Loading component...</div>
{/if} 
