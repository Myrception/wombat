<script>
  import { getFieldRenderer } from './FieldContext';
  import InputLabel from "../controls/InputLabel.svelte";
  import Dropdown from "../controls/Dropdown.svelte";

  export let field = {};
  export let state;

  // Get MessageField component from context
  const MessageFieldComponent = getFieldRenderer();

  let items;
  let selectedValue = undefined;

  $: {
    items = field.oneof.map(x => x.name);
    const k = Object.keys(state)
    checkSelected:
    for (let i = 0; i < k.length; i++) {
      for (let j = 0; j < items.length; j++) {
        if (k[i] === items[j]) {
          selectedValue = k[i];
          break checkSelected;
        }
      }
    }
  }

  const onSelectChanged = ({ detail: { value }}) => {
    if (value === selectedValue) {
      return
    }
    if (selectedValue) {
      delete state[selectedValue]
      state = state;
    }
    selectedValue = value;
  }
  const onSelectClear = () => {
    if (selectedValue) {
      delete state[selectedValue]
      state = state;
    }
    selectedValue = undefined;
  }
</script>

<InputLabel label={"oneof "+field.name} />
<Dropdown
  {items}
  selectedValue={selectedValue}
  isClearable 
  on:clear={onSelectClear}
  on:select={onSelectChanged} />
{#if selectedValue }
  <svelte:component this={MessageFieldComponent} field={field.oneof.find(x => x.name === selectedValue)} {state} oneof />
{/if}
